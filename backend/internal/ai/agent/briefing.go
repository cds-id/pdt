package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/gorm"
)

type BriefingAgent struct {
	DB     *gorm.DB
	UserID uint
}

func (a *BriefingAgent) Name() string { return "briefing" }

func (a *BriefingAgent) SystemPrompt() string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`You are a Morning Briefing assistant for PDT. Today is %s. You help developers prepare for standup/morning briefings by:

1. Generating structured standup reports (done, in progress, blockers)
2. Auditing sprint cards to find risks or weak points that could be questioned
3. Identifying blockers and suggesting who/what is responsible

When generating a briefing report, organize it as:
- **Done yesterday**: Cards moved to Done/Ready to Test with commit evidence
- **In progress today**: Cards currently being worked on with status
- **Blockers/Risks**: Cards with issues, missing info, or stale progress

When auditing, be direct about risks. Flag cards that:
- Have no recent commits but are "In Progress"
- Were assigned recently but have no activity
- Have comments from product/PM asking for updates
- Are overdue or near sprint end without completion

Always include card keys, status, and concrete evidence (commits, comments, dates).`, today)
}

func (a *BriefingAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Name:        "generate_briefing",
			Description: "Generate a morning briefing/standup report for the user showing done, in progress, and blockers",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"sprint_name": {"type": "string", "description": "Sprint name (e.g., 'BNS Sprint 13')"},
					"sprint_id": {"type": "integer", "description": "Sprint ID"},
					"assignee": {"type": "string", "description": "Assignee name (default: current user's jira_username)"},
					"days_back": {"type": "integer", "description": "How many days back to look for 'done' items (default 1)"}
				}
			}`),
		},
		{
			Name:        "audit_sprint_cards",
			Description: "Audit cards in a sprint to find risky or weak cards that could be questioned in a briefing. Returns cards with risk analysis.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"sprint_name": {"type": "string", "description": "Sprint name (e.g., 'BNS Sprint 13')"},
					"sprint_id": {"type": "integer", "description": "Sprint ID"},
					"assignee": {"type": "string", "description": "Assignee name filter"}
				}
			}`),
		},
		{
			Name:        "find_blockers",
			Description: "Analyze cards to find actual or potential blockers — stale cards, missing info, dependency issues, or PM pressure in comments",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"sprint_name": {"type": "string", "description": "Sprint name"},
					"sprint_id": {"type": "integer", "description": "Sprint ID"},
					"assignee": {"type": "string", "description": "Assignee name filter"}
				}
			}`),
		},
	}
}

func (a *BriefingAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "generate_briefing":
		return a.generateBriefing(args)
	case "audit_sprint_cards":
		return a.auditSprintCards(args)
	case "find_blockers":
		return a.findBlockers(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (a *BriefingAgent) resolveSprintID(sprintID int, sprintName string) []uint {
	var ids []uint
	if sprintID > 0 {
		ids = append(ids, uint(sprintID))
	} else if sprintName != "" {
		var sprint models.Sprint
		if a.DB.Where("user_id = ? AND name = ?", a.UserID, sprintName).First(&sprint).Error == nil {
			ids = append(ids, sprint.ID)
		}
	} else {
		var sprints []models.Sprint
		a.DB.Where("user_id = ? AND state = ?", a.UserID, models.SprintActive).Find(&sprints)
		for _, s := range sprints {
			ids = append(ids, s.ID)
		}
	}
	return ids
}

func (a *BriefingAgent) getAssignee(assignee string) string {
	if assignee != "" {
		return assignee
	}
	var user models.User
	if a.DB.First(&user, a.UserID).Error == nil && user.JiraUsername != "" {
		return user.JiraUsername
	}
	return ""
}

func (a *BriefingAgent) generateBriefing(args json.RawMessage) (any, error) {
	var params struct {
		SprintID   int    `json:"sprint_id"`
		SprintName string `json:"sprint_name"`
		Assignee   string `json:"assignee"`
		DaysBack   int    `json:"days_back"`
	}
	json.Unmarshal(args, &params)
	if params.DaysBack == 0 {
		params.DaysBack = 1
	}

	sprintIDs := a.resolveSprintID(params.SprintID, params.SprintName)
	if len(sprintIDs) == 0 {
		return map[string]any{"error": "no sprint found"}, nil
	}

	assignee := a.getAssignee(params.Assignee)

	// Get cards
	query := a.DB.Where("user_id = ? AND sprint_id IN ?", a.UserID, sprintIDs)
	if assignee != "" {
		query = query.Where("assignee LIKE ?", "%"+assignee+"%")
	}
	var cards []models.JiraCard
	query.Find(&cards)

	since := time.Now().AddDate(0, 0, -params.DaysBack)

	type cardEntry struct {
		Key           string   `json:"key"`
		Summary       string   `json:"summary"`
		Status        string   `json:"status"`
		RecentCommits []string `json:"recent_commits,omitempty"`
	}

	var done, inProgress, todo []cardEntry
	for _, c := range cards {
		entry := cardEntry{Key: c.Key, Summary: c.Summary, Status: c.Status}

		// Find recent commits for this card
		var commits []models.Commit
		a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
			Where("repositories.user_id = ? AND commits.jira_card_key = ? AND commits.date >= ?",
				a.UserID, c.Key, since).
			Order("commits.date desc").Limit(5).Find(&commits)

		for _, cm := range commits {
			msg := cm.Message
			if len([]rune(msg)) > 80 {
				msg = string([]rune(msg)[:80]) + "..."
			}
			entry.RecentCommits = append(entry.RecentCommits, fmt.Sprintf("%s: %s", cm.SHA[:7], msg))
		}

		switch {
		case c.Status == "Done" || c.Status == "READY TO TEST" || c.Status == "IN REVIEW":
			done = append(done, entry)
		case c.Status == "In Progress":
			inProgress = append(inProgress, entry)
		default:
			todo = append(todo, entry)
		}
	}

	return map[string]any{
		"sprint":      params.SprintName,
		"assignee":    assignee,
		"date":        time.Now().Format("2006-01-02"),
		"done":        done,
		"in_progress": inProgress,
		"todo":        todo,
		"total_cards": len(cards),
	}, nil
}

func (a *BriefingAgent) auditSprintCards(args json.RawMessage) (any, error) {
	var params struct {
		SprintID   int    `json:"sprint_id"`
		SprintName string `json:"sprint_name"`
		Assignee   string `json:"assignee"`
	}
	json.Unmarshal(args, &params)

	sprintIDs := a.resolveSprintID(params.SprintID, params.SprintName)
	if len(sprintIDs) == 0 {
		return map[string]any{"error": "no sprint found"}, nil
	}

	assignee := a.getAssignee(params.Assignee)

	query := a.DB.Where("user_id = ? AND sprint_id IN ?", a.UserID, sprintIDs)
	if assignee != "" {
		query = query.Where("assignee LIKE ?", "%"+assignee+"%")
	}
	var cards []models.JiraCard
	query.Find(&cards)

	type risk struct {
		Key      string   `json:"key"`
		Summary  string   `json:"summary"`
		Status   string   `json:"status"`
		Risks    []string `json:"risks"`
		Severity string   `json:"severity"` // high, medium, low
	}

	var riskyCards []risk
	for _, c := range cards {
		var risks []string
		severity := "low"

		// Check: no description
		hasDescription := false
		if c.DetailsJSON != "" {
			var details map[string]any
			if json.Unmarshal([]byte(c.DetailsJSON), &details) == nil {
				if desc, ok := details["description"].(string); ok && desc != "" {
					hasDescription = true
				}
			}
		}
		if !hasDescription {
			risks = append(risks, "no description — product may ask why requirements are unclear")
		}

		// Check: no commits
		var commitCount int64
		a.DB.Model(&models.Commit{}).
			Joins("JOIN repositories ON repositories.id = commits.repo_id").
			Where("repositories.user_id = ? AND commits.jira_card_key = ?", a.UserID, c.Key).
			Count(&commitCount)

		if c.Status == "In Progress" && commitCount == 0 {
			risks = append(risks, "In Progress but no commits — looks like no work started")
			severity = "high"
		}

		// Check: no comments
		var commentCount int64
		a.DB.Model(&models.JiraComment{}).Where("user_id = ? AND card_key = ?", a.UserID, c.Key).Count(&commentCount)
		if commentCount == 0 && c.Status != "Done" {
			risks = append(risks, "no comments — no communication trail")
		}

		// Check: stale (In Progress but last commit > 3 days ago)
		if c.Status == "In Progress" && commitCount > 0 {
			var lastCommit models.Commit
			a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
				Where("repositories.user_id = ? AND commits.jira_card_key = ?", a.UserID, c.Key).
				Order("commits.date desc").First(&lastCommit)
			if time.Since(lastCommit.Date) > 3*24*time.Hour {
				risks = append(risks, fmt.Sprintf("stale — last commit was %s, %d days ago",
					lastCommit.Date.Format("2006-01-02"), int(time.Since(lastCommit.Date).Hours()/24)))
				severity = "high"
			}
		}

		// Check: PM/product pressure in comments
		var pressureComments []models.JiraComment
		a.DB.Where("user_id = ? AND card_key = ? AND (LOWER(author) LIKE '%product%' OR LOWER(author) LIKE '%pm%' OR LOWER(author) LIKE '%manager%')",
			a.UserID, c.Key).Find(&pressureComments)
		if len(pressureComments) > 0 {
			risks = append(risks, fmt.Sprintf("%d comment(s) from product/PM — may ask about progress", len(pressureComments)))
			if severity == "low" {
				severity = "medium"
			}
		}

		if len(risks) > 0 {
			if len(risks) >= 3 && severity != "high" {
				severity = "high"
			} else if len(risks) >= 2 && severity == "low" {
				severity = "medium"
			}
			riskyCards = append(riskyCards, risk{
				Key:      c.Key,
				Summary:  c.Summary,
				Status:   c.Status,
				Risks:    risks,
				Severity: severity,
			})
		}
	}

	return map[string]any{
		"total_cards":  len(cards),
		"risky_cards":  len(riskyCards),
		"audit_result": riskyCards,
	}, nil
}

func (a *BriefingAgent) findBlockers(args json.RawMessage) (any, error) {
	var params struct {
		SprintID   int    `json:"sprint_id"`
		SprintName string `json:"sprint_name"`
		Assignee   string `json:"assignee"`
	}
	json.Unmarshal(args, &params)

	sprintIDs := a.resolveSprintID(params.SprintID, params.SprintName)
	if len(sprintIDs) == 0 {
		return map[string]any{"error": "no sprint found"}, nil
	}

	assignee := a.getAssignee(params.Assignee)

	query := a.DB.Where("user_id = ? AND sprint_id IN ?", a.UserID, sprintIDs)
	if assignee != "" {
		query = query.Where("assignee LIKE ?", "%"+assignee+"%")
	}
	var cards []models.JiraCard
	query.Find(&cards)

	type blocker struct {
		Key         string `json:"key"`
		Summary     string `json:"summary"`
		Status      string `json:"status"`
		BlockerType string `json:"blocker_type"`
		Detail      string `json:"detail"`
		Suggestion  string `json:"suggestion"`
	}

	var blockers []blocker
	for _, c := range cards {
		// Skip done cards
		if c.Status == "Done" {
			continue
		}

		// Check: In Progress with no recent commits (stale work)
		var recentCommitCount int64
		threeDaysAgo := time.Now().AddDate(0, 0, -3)
		a.DB.Model(&models.Commit{}).
			Joins("JOIN repositories ON repositories.id = commits.repo_id").
			Where("repositories.user_id = ? AND commits.jira_card_key = ? AND commits.date >= ?",
				a.UserID, c.Key, threeDaysAgo).
			Count(&recentCommitCount)

		if c.Status == "In Progress" && recentCommitCount == 0 {
			blockers = append(blockers, blocker{
				Key:         c.Key,
				Summary:     c.Summary,
				Status:      c.Status,
				BlockerType: "stale_progress",
				Detail:      "No commits in the last 3 days while In Progress",
				Suggestion:  "Either push recent work, move to blocked, or explain the delay in briefing",
			})
		}

		// Check: To Do but sprint is running (not started)
		if c.Status == "To Do" {
			blockers = append(blockers, blocker{
				Key:         c.Key,
				Summary:     c.Summary,
				Status:      c.Status,
				BlockerType: "not_started",
				Detail:      "Card is still To Do in an active sprint",
				Suggestion:  "Prioritize or explain why it hasn't started. Move to In Progress if starting today",
			})
		}

		// Check: comments asking for updates or raising concerns
		var recentComments []models.JiraComment
		weekAgo := time.Now().AddDate(0, 0, -7)
		a.DB.Where("user_id = ? AND card_key = ? AND commented_at >= ? AND author != ?",
			a.UserID, c.Key, weekAgo, assignee).
			Order("commented_at desc").Limit(5).Find(&recentComments)

		for _, comment := range recentComments {
			body := comment.Body
			if len([]rune(body)) > 150 {
				body = string([]rune(body)[:150]) + "..."
			}
			blockers = append(blockers, blocker{
				Key:         c.Key,
				Summary:     c.Summary,
				Status:      c.Status,
				BlockerType: "external_comment",
				Detail:      fmt.Sprintf("%s commented on %s: \"%s\"", comment.Author, comment.CommentedAt.Format("2006-01-02"), body),
				Suggestion:  "Review this comment and prepare a response for briefing",
			})
		}

		// Check: missing description (can't defend scope)
		if c.DetailsJSON != "" {
			var details map[string]any
			if json.Unmarshal([]byte(c.DetailsJSON), &details) == nil {
				if desc, _ := details["description"].(string); desc == "" {
					blockers = append(blockers, blocker{
						Key:         c.Key,
						Summary:     c.Summary,
						Status:      c.Status,
						BlockerType: "no_requirements",
						Detail:      "Card has no description — unclear requirements",
						Suggestion:  "Ask product to add description, or clarify scope in briefing",
					})
				}
			}
		}
	}

	return map[string]any{
		"total_cards":    len(cards),
		"blocker_count":  len(blockers),
		"blockers":       blockers,
	}, nil
}
