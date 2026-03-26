package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

	// Get user info for self-awareness
	var user models.User
	a.DB.First(&user, a.UserID)
	username := user.JiraUsername
	if username == "" {
		username = "the user"
	}

	return fmt.Sprintf(`You are a Morning Briefing assistant for PDT. Today is %s. You understand Indonesian and English.

CRITICAL RULES — NEVER BREAK THESE:
1. NEVER fabricate or hallucinate data. Only present information that comes from tool results.
2. If a tool returns empty results, say "tidak ada data" — do NOT make up comments, dates, or feedback.
3. NEVER invent QA feedback, manager comments, or any text that is not in the tool results.
4. When quoting comments, use the EXACT text from tool results. Do not paraphrase or embellish.

SELF-AWARENESS:
- The current user is "%s". When you see comments authored by "%s" or similar names, those are the USER'S OWN comments — they are explaining or responding to others, NOT external pressure.
- Comments from other people (especially "siswamedia product", PMs, QA) are external — these may contain requests, questions, or pressure directed at the user.
- Distinguish clearly: "Anda berkomentar: ..." vs "siswamedia product berkomentar: ..."

WORKFLOW:
- Prefer calling "full_report" tool which gathers all data in one call.
- Present results based ONLY on tool output.

FORMAT (respond in user's language):
## Laporan Morning Briefing — [date]
### ✅ Selesai (Done) — within requested timeframe
- [KEY] Summary — completed on [date], commits: [evidence]
### 🔄 Sedang Dikerjakan (In Progress)
- [KEY] Summary — last commit [date]
### 📋 Belum Dimulai (To Do)
- [KEY] Summary
### ⚠️ Blocker / Risiko
- [KEY] Issue — severity, suggestion
### 💬 Komentar Eksternal (dari orang lain)
- [KEY] [author] on [date]: "[exact quote]"
### 📝 Komentar Anda
- [KEY] on [date]: "[exact quote]"

If no data exists for a section, write "Tidak ada" — do NOT fill it with made-up content.`, today, username, username)
}

func (a *BriefingAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Name:        "full_report",
			Description: "Generate a complete morning briefing report with status, blockers, audit, and comment analysis — all in one call. Use this as the default tool for any briefing/report request.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"sprint_name": {"type": "string", "description": "Sprint name (e.g., 'BNS Sprint 13')"},
					"sprint_id": {"type": "integer", "description": "Sprint ID"},
					"assignee": {"type": "string", "description": "Assignee name"},
					"days_back": {"type": "integer", "description": "How many days back to look (default 7)"}
				}
			}`),
		},
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
		{
			Name:        "search_comments",
			Description: "Search Jira comments to find miscommunication, unanswered questions, or escalation from product/PM. Use this to find communication gaps.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"author": {"type": "string", "description": "Filter by comment author name (e.g., 'siswamedia product')"},
					"card_key": {"type": "string", "description": "Filter by Jira card key"},
					"keyword": {"type": "string", "description": "Keyword to search in comment body"},
					"since": {"type": "string", "description": "Start date (YYYY-MM-DD)"},
					"until": {"type": "string", "description": "End date (YYYY-MM-DD)"},
					"limit": {"type": "integer", "description": "Max results (default 20)"}
				}
			}`),
		},
	}
}

func (a *BriefingAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "full_report":
		return a.fullReport(args)
	case "generate_briefing":
		return a.generateBriefing(args)
	case "audit_sprint_cards":
		return a.auditSprintCards(args)
	case "find_blockers":
		return a.findBlockers(args)
	case "search_comments":
		return a.searchComments(args)
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
		// Try active sprints first
		var sprints []models.Sprint
		a.DB.Where("user_id = ? AND state = ?", a.UserID, models.SprintActive).Find(&sprints)
		for _, s := range sprints {
			ids = append(ids, s.ID)
		}
		// If no active sprints, use the most recent sprints (active or closed)
		if len(ids) == 0 {
			a.DB.Where("user_id = ? AND state IN ?", a.UserID, []string{string(models.SprintActive), string(models.SprintClosed)}).
				Order("COALESCE(start_date, created_at) DESC").Limit(3).Find(&sprints)
			for _, s := range sprints {
				ids = append(ids, s.ID)
			}
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

func (a *BriefingAgent) fullReport(args json.RawMessage) (any, error) {
	var params struct {
		SprintID   int    `json:"sprint_id"`
		SprintName string `json:"sprint_name"`
		Assignee   string `json:"assignee"`
		DaysBack   int    `json:"days_back"`
	}
	json.Unmarshal(args, &params)
	if params.DaysBack == 0 {
		params.DaysBack = 7
	}

	// Re-marshal for sub-tools
	subArgs, _ := json.Marshal(params)

	briefing, _ := a.generateBriefing(subArgs)
	audit, _ := a.auditSprintCards(subArgs)
	blockers, _ := a.findBlockers(subArgs)

	// Search all comments in the time window
	assignee := a.getAssignee(params.Assignee)
	since := time.Now().AddDate(0, 0, -params.DaysBack).Format("2006-01-02")
	commentArgs, _ := json.Marshal(map[string]any{
		"since": since,
		"limit": 50,
	})
	allComments, _ := a.searchComments(commentArgs)

	// Split comments into user's own vs external
	type commentEntry struct {
		CardKey string `json:"card_key"`
		Author  string `json:"author"`
		Date    string `json:"date"`
		Body    string `json:"body"`
	}
	var externalComments, myComments []commentEntry
	// Convert through JSON to handle the local type
	if raw, err := json.Marshal(allComments); err == nil {
		var commentList []commentEntry
		if json.Unmarshal(raw, &commentList) == nil {
			for _, c := range commentList {
				if assignee != "" && strings.Contains(strings.ToLower(c.Author), strings.ToLower(assignee)) {
					myComments = append(myComments, c)
				} else {
					externalComments = append(externalComments, c)
				}
			}
		}
	}

	return map[string]any{
		"briefing":           briefing,
		"audit":              audit,
		"blockers":           blockers,
		"external_comments":  externalComments,
		"my_comments":        myComments,
		"assignee":           assignee,
		"date":               time.Now().Format("2006-01-02"),
		"days_back":          params.DaysBack,
	}, nil
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
		Key            string `json:"key"`
		Summary        string `json:"summary"`
		Status         string `json:"status"`
		CompletedDate  string `json:"completed_date,omitempty"`
		LastTransition string `json:"last_transition,omitempty"`
		LastCommitDate string `json:"last_commit_date,omitempty"`
		RecentCommits  []string `json:"recent_commits,omitempty"`
	}

	var done, inProgress, todo []cardEntry
	for _, c := range cards {
		entry := cardEntry{Key: c.Key, Summary: c.Summary, Status: c.Status}

		// Find recent commits for this card (within days_back window)
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
			entry.RecentCommits = append(entry.RecentCommits, fmt.Sprintf("%s: %s (%s)", cm.SHA[:7], msg, cm.Date.Format("2006-01-02")))
		}

		// Get last commit date for this card (any time)
		var lastCommit models.Commit
		if a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
			Where("repositories.user_id = ? AND commits.jira_card_key = ?", a.UserID, c.Key).
			Order("commits.date desc").First(&lastCommit).Error == nil {
			entry.LastCommitDate = lastCommit.Date.Format("2006-01-02")
		}

		// Determine completion date from changelog
		status := strings.ToLower(c.Status)
		isDone := status == "done" || status == "ready to test" || status == "in review"
		if isDone && c.DetailsJSON != "" {
			var details struct {
				Changelog []struct {
					Created string `json:"created"`
					Items   []struct {
						Field      string `json:"field"`
						FromString string `json:"from_string"`
						ToString   string `json:"to_string"`
					} `json:"items"`
				} `json:"changelog"`
			}
			if json.Unmarshal([]byte(c.DetailsJSON), &details) == nil {
				var latestTransitionTime time.Time
				for _, h := range details.Changelog {
					for _, item := range h.Items {
						if item.Field == "status" {
							if t, err := time.Parse("2006-01-02T15:04:05.000-0700", h.Created); err == nil {
								if t.After(latestTransitionTime) {
									latestTransitionTime = t
									entry.LastTransition = fmt.Sprintf("%s → %s on %s", item.FromString, item.ToString, t.Format("2006-01-02"))
								}
								if item.ToString == c.Status {
									entry.CompletedDate = t.Format("2006-01-02")
								}
							}
						}
					}
				}
			}
		}

		switch {
		case isDone:
			// Only include in "done" if completed within the days_back window
			if entry.CompletedDate != "" {
				if completedTime, err := time.Parse("2006-01-02", entry.CompletedDate); err == nil {
					if completedTime.After(since) {
						done = append(done, entry)
						continue
					}
				}
			}
			// Fallback: if no completion date found, check last commit date
			if len(commits) > 0 {
				done = append(done, entry)
			}
			// Otherwise skip old done cards
		case status == "in progress":
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

func (a *BriefingAgent) searchComments(args json.RawMessage) (any, error) {
	var params struct {
		Author  string `json:"author"`
		CardKey string `json:"card_key"`
		Keyword string `json:"keyword"`
		Since   string `json:"since"`
		Until   string `json:"until"`
		Limit   int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 20
	}

	query := a.DB.Where("user_id = ?", a.UserID)
	if params.Author != "" {
		query = query.Where("author LIKE ?", "%"+params.Author+"%")
	}
	if params.CardKey != "" {
		query = query.Where("card_key = ?", params.CardKey)
	}
	if params.Keyword != "" {
		query = query.Where("body LIKE ?", "%"+params.Keyword+"%")
	}
	if params.Since != "" {
		if t, err := time.Parse("2006-01-02", params.Since); err == nil {
			query = query.Where("commented_at >= ?", t)
		}
	}
	if params.Until != "" {
		if t, err := time.Parse("2006-01-02", params.Until); err == nil {
			query = query.Where("commented_at <= ?", t.Add(24*time.Hour))
		}
	}

	var comments []models.JiraComment
	query.Order("commented_at desc").Limit(params.Limit).Find(&comments)

	type entry struct {
		CardKey string `json:"card_key"`
		Author  string `json:"author"`
		Date    string `json:"date"`
		Body    string `json:"body"`
	}
	var results []entry
	for _, c := range comments {
		body := c.Body
		if len([]rune(body)) > 300 {
			body = string([]rune(body)[:300]) + "..."
		}
		results = append(results, entry{
			CardKey: c.CardKey,
			Author:  c.Author,
			Date:    c.CommentedAt.Format("2006-01-02 15:04"),
			Body:    body,
		})
	}
	return results, nil
}
