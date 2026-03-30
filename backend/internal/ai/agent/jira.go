package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/helpers"
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/gorm"
)

type JiraAgent struct {
	DB     *gorm.DB
	UserID uint
}

func (a *JiraAgent) Name() string { return "jira" }

func (a *JiraAgent) SystemPrompt() string {
	// List configured workspaces for context
	var workspaces []models.JiraWorkspaceConfig
	a.DB.Where("user_id = ? AND is_active = ?", a.UserID, true).Find(&workspaces)

	wsList := ""
	for _, ws := range workspaces {
		wsList += fmt.Sprintf("\n- %s (ID: %d, keys: %s)", ws.Name, ws.ID, ws.ProjectKeys)
	}
	if wsList == "" {
		wsList = "\n- No workspaces configured"
	}

	return fmt.Sprintf(`You are a Jira assistant for PDT. You help users explore their Jira sprints, cards, and issues. You can also link commits to Jira cards. Use the available tools to fetch data and provide helpful answers.

WORKSPACES:%s

When the user asks about a specific workspace or project, use the workspace_id filter. When not specified, results come from all workspaces.`, wsList)
}

func (a *JiraAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Name:        "get_sprints",
			Description: "List all synced Jira sprints, optionally filtered by state and workspace",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"state": {"type": "string", "enum": ["active", "closed", "future"], "description": "Filter by sprint state"},
					"workspace_id": {"type": "integer", "description": "Filter by workspace ID (optional, omit for all workspaces)"}
				}
			}`),
		},
		{
			Name:        "get_cards",
			Description: "List Jira cards, optionally filtered by sprint, status, assignee, or workspace",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"sprint_id": {"type": "integer", "description": "Filter by sprint ID"},
					"sprint_name": {"type": "string", "description": "Filter by sprint name (e.g., 'BNS Sprint 13')"},
					"status": {"type": "string", "description": "Filter by card status (case-insensitive, e.g., 'Done', 'In Progress', 'READY TO TEST')"},
					"assignee": {"type": "string", "description": "Filter by assignee name (partial match)"},
					"keyword": {"type": "string", "description": "Search keyword in card summary"},
					"workspace_id": {"type": "integer", "description": "Filter by workspace ID (optional)"},
					"limit": {"type": "integer", "description": "Max results (default 30)"}
				}
			}`),
		},
		{
			Name:        "get_card_detail",
			Description: "Get detailed information about a specific Jira card by key",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"key": {"type": "string", "description": "Jira card key (e.g., PDT-123)"}
				},
				"required": ["key"]
			}`),
		},
		{
			Name:        "search_cards",
			Description: "Search Jira cards by keyword across all sprints",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"keyword": {"type": "string", "description": "Search keyword"},
					"limit": {"type": "integer", "description": "Max results (default 20)"}
				},
				"required": ["keyword"]
			}`),
		},
		{
			Name:        "link_commit_to_card",
			Description: "Link a commit to a Jira card by SHA and card key",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"sha": {"type": "string", "description": "Commit SHA (full or short)"},
					"card_key": {"type": "string", "description": "Jira card key (e.g., PDT-123)"}
				},
				"required": ["sha", "card_key"]
			}`),
		},
	}
}

func (a *JiraAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "get_sprints":
		return a.getSprints(args)
	case "get_cards":
		return a.getCards(args)
	case "get_card_detail":
		return a.getCardDetail(args)
	case "search_cards":
		return a.searchCards(args)
	case "link_commit_to_card":
		return a.linkCommitToCard(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (a *JiraAgent) getSprints(args json.RawMessage) (any, error) {
	var params struct {
		State       string `json:"state"`
		WorkspaceID int    `json:"workspace_id"`
	}
	json.Unmarshal(args, &params)

	query := a.DB.Where("user_id = ?", a.UserID)
	if params.State != "" {
		query = query.Where("state = ?", params.State)
	}
	if params.WorkspaceID > 0 {
		query = query.Where("workspace_id = ?", params.WorkspaceID)
	}

	var sprints []models.Sprint
	query.Order("start_date desc").Find(&sprints)

	type result struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		State     string `json:"state"`
		StartDate string `json:"start_date,omitempty"`
		EndDate   string `json:"end_date,omitempty"`
		CardCount int64  `json:"card_count"`
	}
	var results []result
	for _, s := range sprints {
		var count int64
		a.DB.Model(&models.JiraCard{}).Where("sprint_id = ?", s.ID).Count(&count)
		r := result{
			ID:        s.ID,
			Name:      s.Name,
			State:     string(s.State),
			CardCount: count,
		}
		if s.StartDate != nil {
			r.StartDate = s.StartDate.Format("2006-01-02")
		}
		if s.EndDate != nil {
			r.EndDate = s.EndDate.Format("2006-01-02")
		}
		results = append(results, r)
	}
	return results, nil
}

func (a *JiraAgent) getCards(args json.RawMessage) (any, error) {
	var params struct {
		SprintID    int    `json:"sprint_id"`
		SprintName  string `json:"sprint_name"`
		Status      string `json:"status"`
		Assignee    string `json:"assignee"`
		Keyword     string `json:"keyword"`
		WorkspaceID int    `json:"workspace_id"`
		Limit       int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 30
	}

	query := a.DB.Where("jira_cards.user_id = ?", a.UserID)

	if params.WorkspaceID > 0 {
		query = query.Where("jira_cards.workspace_id = ?", params.WorkspaceID)
	}

	// Resolve sprint by name if provided
	if params.SprintName != "" && params.SprintID == 0 {
		var sprint models.Sprint
		if err := a.DB.Where("user_id = ? AND name = ?", a.UserID, params.SprintName).First(&sprint).Error; err == nil {
			params.SprintID = int(sprint.ID)
		}
	}
	if params.SprintID > 0 {
		query = query.Where("sprint_id = ?", params.SprintID)
	}
	if params.Status != "" {
		query = query.Where("LOWER(status) = LOWER(?)", params.Status)
	}
	if params.Assignee != "" {
		query = query.Where("assignee LIKE ?", "%"+params.Assignee+"%")
	}
	if params.Keyword != "" {
		query = query.Where("summary LIKE ?", "%"+params.Keyword+"%")
	}

	// Apply project key filter from workspaces
	projectKeys := a.getProjectKeys(params.WorkspaceID)
	if clause, filterArgs := helpers.BuildProjectKeyWhereClauses(projectKeys, "card_key"); clause != "" {
		query = query.Where(clause, filterArgs...)
	}

	var cards []models.JiraCard
	query.Order("created_at desc").Limit(params.Limit).Find(&cards)

	type result struct {
		Key         string `json:"key"`
		Summary     string `json:"summary"`
		Status      string `json:"status"`
		Assignee    string `json:"assignee"`
		Description string `json:"description,omitempty"`
	}
	var results []result
	for _, c := range cards {
		r := result{
			Key:      c.Key,
			Summary:  c.Summary,
			Status:   c.Status,
			Assignee: c.Assignee,
		}
		// Extract description from DetailsJSON
		if c.DetailsJSON != "" {
			var details map[string]any
			if json.Unmarshal([]byte(c.DetailsJSON), &details) == nil {
				if desc, ok := details["description"].(string); ok {
					r.Description = desc
				}
			}
		}
		results = append(results, r)
	}
	return results, nil
}

func (a *JiraAgent) getCardDetail(args json.RawMessage) (any, error) {
	var params struct {
		Key string `json:"key"`
	}
	json.Unmarshal(args, &params)

	var card models.JiraCard
	if err := a.DB.Where("user_id = ? AND card_key = ?", a.UserID, params.Key).First(&card).Error; err != nil {
		return nil, fmt.Errorf("card not found: %s", params.Key)
	}

	result := a.buildCardContext(card)
	return result, nil
}

// extractTransitions pulls status transitions from DetailsJSON changelog
func (a *JiraAgent) extractTransitions(detailsJSON string) []map[string]string {
	if detailsJSON == "" {
		return nil
	}
	var details struct {
		Changelog []struct {
			Author  string `json:"author"`
			Created string `json:"created"`
			Items   []struct {
				Field      string `json:"field"`
				FromString string `json:"from_string"`
				ToString   string `json:"to_string"`
			} `json:"items"`
		} `json:"changelog"`
	}
	if json.Unmarshal([]byte(detailsJSON), &details) != nil {
		return nil
	}

	var transitions []map[string]string
	for _, h := range details.Changelog {
		date := h.Created
		if t, err := time.Parse("2006-01-02T15:04:05.000-0700", h.Created); err == nil {
			date = t.Format("2006-01-02 15:04")
		}
		for _, item := range h.Items {
			transitions = append(transitions, map[string]string{
				"field": item.Field,
				"from":  item.FromString,
				"to":    item.ToString,
				"by":    h.Author,
				"date":  date,
			})
		}
	}
	return transitions
}

// buildCardContext returns full context for a card including description, commits, comments, parent, and subtasks
func (a *JiraAgent) buildCardContext(card models.JiraCard) map[string]any {
	type commitInfo struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Date    string `json:"date"`
	}
	type commentInfo struct {
		Author string `json:"author"`
		Date   string `json:"date"`
		Body   string `json:"body"`
	}
	type childCard struct {
		Key         string              `json:"key"`
		Summary     string              `json:"summary"`
		Status      string              `json:"status"`
		Description string              `json:"description,omitempty"`
		Commits     []commitInfo        `json:"commits,omitempty"`
		Comments    []commentInfo       `json:"comments,omitempty"`
		Transitions []map[string]string `json:"transitions,omitempty"`
	}

	// Commits
	var commits []models.Commit
	a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.jira_card_key = ?", a.UserID, card.Key).
		Order("commits.date desc").Limit(10).Find(&commits)

	var linkedCommits []commitInfo
	for _, c := range commits {
		linkedCommits = append(linkedCommits, commitInfo{
			SHA:     shortSHA(c.SHA),
			Message: c.Message,
			Date:    c.Date.Format("2006-01-02 15:04"),
		})
	}

	// Comments
	var jiraComments []models.JiraComment
	a.DB.Where("user_id = ? AND card_key = ?", a.UserID, card.Key).
		Order("commented_at desc").Limit(10).Find(&jiraComments)

	var cardComments []commentInfo
	for _, c := range jiraComments {
		body := c.Body
		if len([]rune(body)) > 300 {
			body = string([]rune(body)[:300]) + "..."
		}
		cardComments = append(cardComments, commentInfo{
			Author: c.Author,
			Date:   c.CommentedAt.Format("2006-01-02 15:04"),
			Body:   body,
		})
	}

	result := map[string]any{
		"key":         card.Key,
		"summary":     card.Summary,
		"status":      card.Status,
		"assignee":    card.Assignee,
		"commits":     linkedCommits,
		"comments":    cardComments,
		"transitions": a.extractTransitions(card.DetailsJSON),
	}

	// Parse DetailsJSON for description, parent, subtasks
	if card.DetailsJSON != "" {
		var details map[string]any
		if json.Unmarshal([]byte(card.DetailsJSON), &details) == nil {
			if desc, ok := details["description"].(string); ok {
				result["description"] = desc
			}

			// Parent card with full context
			if parent, ok := details["parent"].(map[string]any); ok {
				parentKey, _ := parent["key"].(string)
				if parentKey != "" {
					var parentCard models.JiraCard
					if a.DB.Where("user_id = ? AND card_key = ?", a.UserID, parentKey).First(&parentCard).Error == nil {
						result["parent"] = a.buildCardSummary(parentCard)
					} else {
						result["parent"] = parent
					}
				}
			}

			// Subtasks with commits and comments
			if subtasks, ok := details["subtasks"].([]any); ok {
				var children []childCard
				for _, st := range subtasks {
					stMap, ok := st.(map[string]any)
					if !ok {
						continue
					}
					stKey, _ := stMap["key"].(string)
					stSummary, _ := stMap["summary"].(string)
					stStatus, _ := stMap["status"].(string)

					child := childCard{Key: stKey, Summary: stSummary, Status: stStatus}

					// Get subtask commits
					var stCommits []models.Commit
					a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
						Where("repositories.user_id = ? AND commits.jira_card_key = ?", a.UserID, stKey).
						Order("commits.date desc").Limit(5).Find(&stCommits)
					for _, c := range stCommits {
						child.Commits = append(child.Commits, commitInfo{
							SHA: shortSHA(c.SHA), Message: c.Message, Date: c.Date.Format("2006-01-02 15:04"),
						})
					}

					// Get subtask comments
					var stComments []models.JiraComment
					a.DB.Where("user_id = ? AND card_key = ?", a.UserID, stKey).
						Order("commented_at desc").Limit(5).Find(&stComments)
					for _, c := range stComments {
						body := c.Body
						if len([]rune(body)) > 200 {
							body = string([]rune(body)[:200]) + "..."
						}
						child.Comments = append(child.Comments, commentInfo{
							Author: c.Author, Date: c.CommentedAt.Format("2006-01-02 15:04"), Body: body,
						})
					}

					// Get subtask description
					var stCard models.JiraCard
					if a.DB.Where("user_id = ? AND card_key = ?", a.UserID, stKey).First(&stCard).Error == nil {
						if stCard.DetailsJSON != "" {
							var stDetails map[string]any
							if json.Unmarshal([]byte(stCard.DetailsJSON), &stDetails) == nil {
								if desc, ok := stDetails["description"].(string); ok {
									child.Description = desc
								}
							}
							child.Transitions = a.extractTransitions(stCard.DetailsJSON)
						}
					}

					children = append(children, child)
				}
				result["subtasks"] = children
			}
		}
	}

	return result
}

// buildCardSummary returns a lighter context for parent cards
func (a *JiraAgent) buildCardSummary(card models.JiraCard) map[string]any {
	result := map[string]any{
		"key":      card.Key,
		"summary":  card.Summary,
		"status":   card.Status,
		"assignee": card.Assignee,
	}
	if card.DetailsJSON != "" {
		var details map[string]any
		if json.Unmarshal([]byte(card.DetailsJSON), &details) == nil {
			if desc, ok := details["description"].(string); ok {
				result["description"] = desc
			}
		}
	}
	return result
}

func (a *JiraAgent) searchCards(args json.RawMessage) (any, error) {
	var params struct {
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 20
	}

	var cards []models.JiraCard
	a.DB.Where("user_id = ? AND (card_key LIKE ? OR summary LIKE ?)",
		a.UserID, "%"+params.Keyword+"%", "%"+params.Keyword+"%").
		Limit(params.Limit).Find(&cards)

	type result struct {
		Key     string `json:"key"`
		Summary string `json:"summary"`
		Status  string `json:"status"`
	}
	var results []result
	for _, c := range cards {
		results = append(results, result{Key: c.Key, Summary: c.Summary, Status: c.Status})
	}
	return results, nil
}

// getProjectKeys returns combined project keys for filtering.
// If workspaceID > 0, returns keys for that workspace only.
// Otherwise returns all keys from all active workspaces.
func (a *JiraAgent) getProjectKeys(workspaceID int) string {
	if workspaceID > 0 {
		var ws models.JiraWorkspaceConfig
		if a.DB.Where("id = ? AND user_id = ?", workspaceID, a.UserID).First(&ws).Error == nil {
			return ws.ProjectKeys
		}
		return ""
	}

	var workspaces []models.JiraWorkspaceConfig
	a.DB.Where("user_id = ? AND is_active = ?", a.UserID, true).Find(&workspaces)

	var allKeys []string
	for _, ws := range workspaces {
		if ws.ProjectKeys != "" {
			allKeys = append(allKeys, ws.ProjectKeys)
		}
	}
	if len(allKeys) == 0 {
		// Legacy fallback
		var user models.User
		if a.DB.First(&user, a.UserID).Error == nil {
			return user.JiraProjectKeys
		}
	}
	return strings.Join(allKeys, ",")
}

func (a *JiraAgent) linkCommitToCard(args json.RawMessage) (any, error) {
	var params struct {
		SHA     string `json:"sha"`
		CardKey string `json:"card_key"`
	}
	json.Unmarshal(args, &params)

	var commit models.Commit
	if err := a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.sha LIKE ?", a.UserID, params.SHA+"%").
		First(&commit).Error; err != nil {
		return nil, fmt.Errorf("commit not found: %s", params.SHA)
	}

	commit.JiraCardKey = params.CardKey
	commit.HasLink = true
	a.DB.Save(&commit)

	return map[string]string{
		"status":   "linked",
		"sha":      shortSHA(commit.SHA),
		"card_key": params.CardKey,
	}, nil
}
