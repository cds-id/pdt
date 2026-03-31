package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/gorm"
)

type Executor struct {
	DB     *gorm.DB
	Client *minimax.Client
	Agents map[string]agent.Agent
}

type nopWriter struct{}

func (nopWriter) WriteContent(string) error           { return nil }
func (nopWriter) WriteThinking(string) error           { return nil }
func (nopWriter) WriteToolStatus(string, string) error { return nil }
func (nopWriter) WriteDone() error                     { return nil }
func (nopWriter) WriteError(string) error              { return nil }

func (e *Executor) Run(ctx context.Context, schedule models.AgentSchedule, triggerType string) (*models.AgentScheduleRun, error) {
	now := time.Now()
	run := models.AgentScheduleRun{
		ScheduleID:  schedule.ID,
		UserID:      schedule.UserID,
		Status:      "running",
		TriggerType: triggerType,
		StartedAt:   &now,
	}
	if err := e.DB.Create(&run).Error; err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	conv := models.Conversation{
		UserID: schedule.UserID,
		Title:  fmt.Sprintf("Scheduled: %s — %s", schedule.Name, now.Format("2006-01-02")),
	}
	if err := e.DB.Create(&conv).Error; err != nil {
		e.failRun(&run, fmt.Errorf("create conversation: %w", err))
		return &run, nil
	}
	run.ConversationID = conv.ID

	userMsg := models.ChatMessage{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        schedule.Prompt,
	}
	e.DB.Create(&userMsg)

	messages := []minimax.Message{{Role: "user", Content: schedule.Prompt}}
	result, err := e.runAgent(ctx, schedule.AgentName, messages, &run)
	if err != nil {
		e.failRun(&run, err)
		return &run, nil
	}

	if result.FullResponse != "" {
		assistantMsg := models.ChatMessage{
			ConversationID: conv.ID,
			Role:           "assistant",
			Content:        result.FullResponse,
		}
		e.DB.Create(&assistantMsg)
	}

	if len(schedule.ChainConfig) > 0 {
		e.runChain(ctx, schedule.ChainConfig, result.FullResponse, &run)
	}

	completedAt := time.Now()
	usageJSON, _ := json.Marshal(map[string]int{
		"prompt_tokens":     result.Usage.PromptTokens,
		"completion_tokens": result.Usage.CompletionTokens,
	})
	e.DB.Model(&run).Updates(map[string]any{
		"status":          "completed",
		"completed_at":    &completedAt,
		"conversation_id": conv.ID,
		"result_summary":  summarize(result.FullResponse, 500),
		"token_usage":     string(usageJSON),
	})
	run.Status = "completed"
	return &run, nil
}

func (e *Executor) runAgent(ctx context.Context, agentName string, messages []minimax.Message, run *models.AgentScheduleRun) (*agent.LoopResult, error) {
	start := time.Now()
	var result *agent.LoopResult
	var err error

	if agentName == "" {
		orchestrator := agent.NewOrchestrator(e.Client, e.agentSlice()...)
		result, err = orchestrator.HandleMessage(ctx, messages, nopWriter{})
	} else {
		a, ok := e.Agents[agentName]
		if !ok {
			return nil, fmt.Errorf("unknown agent: %s", agentName)
		}
		result, err = agent.RunLoop(ctx, e.Client, a, messages, nopWriter{})
	}

	status := "completed"
	response := ""
	if err != nil {
		status = "failed"
		response = err.Error()
	} else {
		response = result.FullResponse
	}

	step := models.AgentScheduleRunStep{
		RunID:      run.ID,
		AgentName:  agentName,
		Prompt:     messages[len(messages)-1].Content,
		Response:   response,
		Status:     status,
		DurationMs: int(time.Since(start).Milliseconds()),
	}
	e.DB.Create(&step)

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e *Executor) runChain(ctx context.Context, chainConfigJSON json.RawMessage, previousResponse string, run *models.AgentScheduleRun) {
	var steps []models.ChainStep
	if err := json.Unmarshal(chainConfigJSON, &steps); err != nil {
		log.Printf("[scheduler] invalid chain config: %v", err)
		return
	}
	for _, step := range steps {
		if !evaluateCondition(step.Condition, previousResponse, "completed") {
			continue
		}
		messages := []minimax.Message{{Role: "user", Content: step.Prompt}}
		result, err := e.runAgent(ctx, step.Agent, messages, run)
		if err != nil {
			log.Printf("[scheduler] chain step %s failed: %v", step.Agent, err)
			continue
		}
		previousResponse = result.FullResponse
	}
}

func (e *Executor) failRun(run *models.AgentScheduleRun, err error) {
	now := time.Now()
	e.DB.Model(run).Updates(map[string]any{
		"status":       "failed",
		"completed_at": &now,
		"error":        err.Error(),
	})
	run.Status = "failed"
	run.Error = err.Error()
}

func (e *Executor) agentSlice() []agent.Agent {
	agents := make([]agent.Agent, 0, len(e.Agents))
	for _, a := range e.Agents {
		agents = append(agents, a)
	}
	return agents
}

func evaluateCondition(condition, response, status string) bool {
	switch {
	case condition == "always" || condition == "":
		return true
	case condition == "status:completed":
		return status == "completed"
	case condition == "status:failed":
		return status == "failed"
	case strings.HasPrefix(condition, "contains:"):
		keyword := strings.TrimPrefix(condition, "contains:")
		return strings.Contains(strings.ToLower(response), strings.ToLower(keyword))
	default:
		return false
	}
}

func summarize(text string, maxLen int) string {
	lines := strings.SplitN(text, "\n", 5)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimLeft(trimmed, "# ")
			if len(heading) > 0 && len(heading) <= maxLen {
				return heading
			}
		}
	}
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen])
}
