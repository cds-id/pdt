package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// ScheduleEngine is the interface the scheduler agent needs from the engine.
// Defined here to avoid an import cycle with the scheduler package.
type ScheduleEngine interface {
	RefreshEventSubscriptions()
	RunScheduleNow(schedule models.AgentSchedule)
}

type SchedulerAgent struct {
	DB     *gorm.DB
	UserID uint
	Engine ScheduleEngine
}

func (a *SchedulerAgent) Name() string { return "scheduler" }

func (a *SchedulerAgent) SystemPrompt() string {
	return `You are a Schedule Manager for PDT. You help users create, view, modify, and manage scheduled agent tasks.

CAPABILITIES:
- List all schedules
- Create new schedules (cron, interval, or event-triggered)
- Enable/disable schedules
- Delete schedules
- Trigger a schedule to run immediately
- View run history for a schedule

TRIGGER TYPES:
- "once": Run immediately one time, then auto-disable. Use for one-off tasks.
- "cron": Standard 5-field cron expressions (minute hour day-of-month month day-of-week)
  Examples: "0 8 * * 1-5" (weekdays 8am), "0 9 * * 1" (Monday 9am), "0 * * * *" (every hour)
- "interval": Run every N seconds. Common values: 900 (15min), 1800 (30min), 3600 (1hr)
- "event": Triggered by system events. Available events: commit_synced, jira_synced, report_generated, schedule_completed

AVAILABLE AGENTS:
- "briefing": Morning briefing, standup prep, blocker analysis
- "git": Commit analysis, repo insights
- "jira": Sprint/card queries, linking
- "report": Daily/monthly report generation
- "proof": Evidence search, quality checks
- "whatsapp": Message sending, chat analytics
- "" (empty): Auto-route via orchestrator

CHAIN CONDITIONS:
- "always": Always run the chain step
- "contains:<keyword>": Run if previous response contains keyword (case-insensitive)
- "status:completed": Run if previous step completed successfully
- "status:failed": Run if previous step failed

When creating schedules, help the user by suggesting appropriate cron expressions, agents, and prompts.
Respond in the user's language (Indonesian or English).`
}

func (a *SchedulerAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Name:        "list_schedules",
			Description: "List all scheduled tasks for the current user",
			InputSchema: json.RawMessage(`{"type": "object", "properties": {}}`),
		},
		{
			Name:        "create_schedule",
			Description: "Create a new scheduled agent task",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "Human-readable name for the schedule"},
					"agent_name": {"type": "string", "description": "Agent to run (briefing, git, jira, report, proof, whatsapp, or empty for auto-route)"},
					"prompt": {"type": "string", "description": "The message/instruction to send to the agent"},
					"trigger_type": {"type": "string", "enum": ["cron", "interval", "event", "once"], "description": "Type of trigger. Use 'once' to run immediately one time."},
					"cron_expr": {"type": "string", "description": "Cron expression (for cron trigger type)"},
					"interval_seconds": {"type": "integer", "description": "Interval in seconds (for interval trigger type)"},
					"event_name": {"type": "string", "description": "Event name (for event trigger type): commit_synced, jira_synced, report_generated, schedule_completed"},
					"chain_config": {
						"type": "array",
						"description": "Optional chain steps to run after the main agent",
						"items": {
							"type": "object",
							"properties": {
								"agent": {"type": "string"},
								"prompt": {"type": "string"},
								"condition": {"type": "string"}
							}
						}
					}
				},
				"required": ["name", "prompt", "trigger_type"]
			}`),
		},
		{
			Name:        "toggle_schedule",
			Description: "Enable or disable a schedule",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"schedule_id": {"type": "string", "description": "ID of the schedule to toggle"}
				},
				"required": ["schedule_id"]
			}`),
		},
		{
			Name:        "delete_schedule",
			Description: "Delete a schedule and all its run history",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"schedule_id": {"type": "string", "description": "ID of the schedule to delete"}
				},
				"required": ["schedule_id"]
			}`),
		},
		{
			Name:        "run_schedule_now",
			Description: "Trigger a schedule to run immediately",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"schedule_id": {"type": "string", "description": "ID of the schedule to run"}
				},
				"required": ["schedule_id"]
			}`),
		},
		{
			Name:        "list_schedule_runs",
			Description: "View run history for a schedule",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"schedule_id": {"type": "string", "description": "ID of the schedule"}
				},
				"required": ["schedule_id"]
			}`),
		},
	}
}

func (a *SchedulerAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "list_schedules":
		return a.listSchedules()
	case "create_schedule":
		return a.createSchedule(args)
	case "toggle_schedule":
		return a.toggleSchedule(args)
	case "delete_schedule":
		return a.deleteSchedule(args)
	case "run_schedule_now":
		return a.runScheduleNow(args)
	case "list_schedule_runs":
		return a.listScheduleRuns(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (a *SchedulerAgent) listSchedules() (any, error) {
	var schedules []models.AgentSchedule
	a.DB.Where("user_id = ?", a.UserID).Order("created_at desc").Find(&schedules)

	type scheduleInfo struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Agent       string  `json:"agent"`
		TriggerType string  `json:"trigger_type"`
		CronExpr    string  `json:"cron_expr,omitempty"`
		Interval    int     `json:"interval_seconds,omitempty"`
		EventName   string  `json:"event_name,omitempty"`
		Enabled     bool    `json:"enabled"`
		NextRun     *string `json:"next_run,omitempty"`
		Prompt      string  `json:"prompt"`
	}

	results := make([]scheduleInfo, 0, len(schedules))
	for _, s := range schedules {
		info := scheduleInfo{
			ID:          s.ID,
			Name:        s.Name,
			Agent:       s.AgentName,
			TriggerType: s.TriggerType,
			CronExpr:    s.CronExpr,
			Interval:    s.IntervalSeconds,
			EventName:   s.EventName,
			Enabled:     s.Enabled,
			Prompt:      s.Prompt,
		}
		if s.NextRunAt != nil {
			t := s.NextRunAt.Format(time.RFC3339)
			info.NextRun = &t
		}
		results = append(results, info)
	}

	if len(results) == 0 {
		return map[string]string{"message": "No schedules found"}, nil
	}
	return results, nil
}

func (a *SchedulerAgent) createSchedule(args json.RawMessage) (any, error) {
	var req struct {
		Name            string             `json:"name"`
		AgentName       string             `json:"agent_name"`
		Prompt          string             `json:"prompt"`
		TriggerType     string             `json:"trigger_type"`
		CronExpr        string             `json:"cron_expr"`
		IntervalSeconds int                `json:"interval_seconds"`
		EventName       string             `json:"event_name"`
		ChainConfig     []models.ChainStep `json:"chain_config"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	schedule := models.AgentSchedule{
		UserID:          a.UserID,
		Name:            req.Name,
		AgentName:       req.AgentName,
		Prompt:          req.Prompt,
		TriggerType:     req.TriggerType,
		CronExpr:        req.CronExpr,
		IntervalSeconds: req.IntervalSeconds,
		EventName:       req.EventName,
		Enabled:         true,
	}

	if len(req.ChainConfig) > 0 {
		chainJSON, _ := json.Marshal(req.ChainConfig)
		schedule.ChainConfig = chainJSON
	}

	// Compute next run
	if schedule.TriggerType == "once" {
		now := time.Now()
		schedule.NextRunAt = &now
	} else if schedule.TriggerType != "event" {
		nextRun, err := computeNextRun(schedule.TriggerType, schedule.CronExpr, schedule.IntervalSeconds, time.Now())
		if err != nil {
			return map[string]string{"error": "Invalid schedule config: " + err.Error()}, nil
		}
		schedule.NextRunAt = nextRun
	}

	if err := a.DB.Create(&schedule).Error; err != nil {
		return nil, fmt.Errorf("create schedule: %w", err)
	}

	if a.Engine != nil {
		a.Engine.RefreshEventSubscriptions()
	}

	return map[string]any{
		"message":     fmt.Sprintf("Schedule %q created successfully", schedule.Name),
		"id":          schedule.ID,
		"name":        schedule.Name,
		"trigger":     schedule.TriggerType,
		"next_run_at": schedule.NextRunAt,
		"enabled":     true,
	}, nil
}

func (a *SchedulerAgent) toggleSchedule(args json.RawMessage) (any, error) {
	var req struct {
		ScheduleID string `json:"schedule_id"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	var s models.AgentSchedule
	if err := a.DB.Where("id = ? AND user_id = ?", req.ScheduleID, a.UserID).First(&s).Error; err != nil {
		return map[string]string{"error": "Schedule not found"}, nil
	}

	newEnabled := !s.Enabled
	a.DB.Model(&s).Update("enabled", newEnabled)

	if a.Engine != nil {
		a.Engine.RefreshEventSubscriptions()
	}

	status := "disabled"
	if newEnabled {
		status = "enabled"
	}
	return map[string]string{
		"message": fmt.Sprintf("Schedule %q is now %s", s.Name, status),
		"status":  status,
	}, nil
}

func (a *SchedulerAgent) deleteSchedule(args json.RawMessage) (any, error) {
	var req struct {
		ScheduleID string `json:"schedule_id"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	var s models.AgentSchedule
	if err := a.DB.Where("id = ? AND user_id = ?", req.ScheduleID, a.UserID).First(&s).Error; err != nil {
		return map[string]string{"error": "Schedule not found"}, nil
	}

	name := s.Name
	var runIDs []string
	a.DB.Model(&models.AgentScheduleRun{}).Where("schedule_id = ?", req.ScheduleID).Pluck("id", &runIDs)
	if len(runIDs) > 0 {
		a.DB.Where("run_id IN ?", runIDs).Delete(&models.AgentScheduleRunStep{})
	}
	a.DB.Where("schedule_id = ?", req.ScheduleID).Delete(&models.AgentScheduleRun{})
	a.DB.Delete(&s)

	if a.Engine != nil {
		a.Engine.RefreshEventSubscriptions()
	}

	return map[string]string{"message": fmt.Sprintf("Schedule %q deleted", name)}, nil
}

func (a *SchedulerAgent) runScheduleNow(args json.RawMessage) (any, error) {
	var req struct {
		ScheduleID string `json:"schedule_id"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	var s models.AgentSchedule
	if err := a.DB.Where("id = ? AND user_id = ?", req.ScheduleID, a.UserID).First(&s).Error; err != nil {
		return map[string]string{"error": "Schedule not found"}, nil
	}

	if a.Engine != nil {
		a.Engine.RunScheduleNow(s)
	}

	return map[string]string{
		"message": fmt.Sprintf("Schedule %q triggered. It will run in the background.", s.Name),
	}, nil
}

func (a *SchedulerAgent) listScheduleRuns(args json.RawMessage) (any, error) {
	var req struct {
		ScheduleID string `json:"schedule_id"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	var s models.AgentSchedule
	if err := a.DB.Where("id = ? AND user_id = ?", req.ScheduleID, a.UserID).First(&s).Error; err != nil {
		return map[string]string{"error": "Schedule not found"}, nil
	}

	var runs []models.AgentScheduleRun
	a.DB.Where("schedule_id = ?", req.ScheduleID).Order("created_at desc").Limit(10).Find(&runs)

	type runInfo struct {
		ID        string  `json:"id"`
		Status    string  `json:"status"`
		Trigger   string  `json:"trigger_type"`
		StartedAt *string `json:"started_at,omitempty"`
		Duration  string  `json:"duration,omitempty"`
		Summary   string  `json:"summary,omitempty"`
		Error     string  `json:"error,omitempty"`
	}

	results := make([]runInfo, 0, len(runs))
	for _, r := range runs {
		info := runInfo{
			ID:      r.ID,
			Status:  r.Status,
			Trigger: r.TriggerType,
			Summary: r.ResultSummary,
			Error:   r.Error,
		}
		if r.StartedAt != nil {
			t := r.StartedAt.Format(time.RFC3339)
			info.StartedAt = &t
		}
		if r.StartedAt != nil && r.CompletedAt != nil {
			info.Duration = r.CompletedAt.Sub(*r.StartedAt).Round(time.Second).String()
		}
		results = append(results, info)
	}

	if len(results) == 0 {
		return map[string]string{"message": fmt.Sprintf("No runs found for schedule %q", s.Name)}, nil
	}
	return map[string]any{"schedule": s.Name, "runs": results}, nil
}

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func computeNextRun(triggerType, cronExpr string, intervalSeconds int, after time.Time) (*time.Time, error) {
	switch triggerType {
	case "cron":
		schedule, err := cronParser.Parse(cronExpr)
		if err != nil {
			return nil, fmt.Errorf("invalid cron expression: %w", err)
		}
		next := schedule.Next(after)
		return &next, nil
	case "interval":
		next := after.Add(time.Duration(intervalSeconds) * time.Second)
		return &next, nil
	case "event":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown trigger type: %s", triggerType)
	}
}
