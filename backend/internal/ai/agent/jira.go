package agent

import (
	"context"
	"encoding/json"
	"fmt"

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
	return `You are a Jira assistant for PDT. You help users explore their Jira sprints, cards, and issues. You can also link commits to Jira cards. Use the available tools to fetch data and provide helpful answers.`
}

func (a *JiraAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Name:        "get_sprints",
			Description: "List all synced Jira sprints, optionally filtered by state",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"state": {"type": "string", "enum": ["active", "closed", "future"], "description": "Filter by sprint state"}
				}
			}`),
		},
		{
			Name:        "get_cards",
			Description: "List Jira cards, optionally filtered by sprint or status",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"sprint_id": {"type": "integer", "description": "Filter by sprint ID"},
					"status": {"type": "string", "description": "Filter by card status (e.g., 'Done', 'In Progress')"},
					"keyword": {"type": "string", "description": "Search keyword in card summary"},
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
		State string `json:"state"`
	}
	json.Unmarshal(args, &params)

	query := a.DB.Where("user_id = ?", a.UserID)
	if params.State != "" {
		query = query.Where("state = ?", params.State)
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
		SprintID int    `json:"sprint_id"`
		Status   string `json:"status"`
		Keyword  string `json:"keyword"`
		Limit    int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 30
	}

	// Get user for project key filtering
	var user models.User
	a.DB.First(&user, a.UserID)

	query := a.DB.Where("user_id = ?", a.UserID)
	if params.SprintID > 0 {
		query = query.Where("sprint_id = ?", params.SprintID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.Keyword != "" {
		query = query.Where("summary LIKE ?", "%"+params.Keyword+"%")
	}

	// Apply project key filter
	if clause, filterArgs := helpers.BuildProjectKeyWhereClauses(user.JiraProjectKeys, "card_key"); clause != "" {
		query = query.Where(clause, filterArgs...)
	}

	var cards []models.JiraCard
	query.Order("created_at desc").Limit(params.Limit).Find(&cards)

	type result struct {
		Key      string `json:"key"`
		Summary  string `json:"summary"`
		Status   string `json:"status"`
		Assignee string `json:"assignee"`
	}
	var results []result
	for _, c := range cards {
		results = append(results, result{
			Key:      c.Key,
			Summary:  c.Summary,
			Status:   c.Status,
			Assignee: c.Assignee,
		})
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

	// Also fetch linked commits
	var commits []models.Commit
	a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.jira_card_key = ?", a.UserID, params.Key).
		Find(&commits)

	type commitInfo struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Date    string `json:"date"`
	}
	var linkedCommits []commitInfo
	for _, c := range commits {
		linkedCommits = append(linkedCommits, commitInfo{
			SHA:     shortSHA(c.SHA),
			Message: c.Message,
			Date:    c.Date.Format("2006-01-02 15:04"),
		})
	}

	result := map[string]any{
		"key":      card.Key,
		"summary":  card.Summary,
		"status":   card.Status,
		"assignee": card.Assignee,
		"commits":  linkedCommits,
	}
	if card.DetailsJSON != "" {
		var details any
		json.Unmarshal([]byte(card.DetailsJSON), &details)
		result["details"] = details
	}
	return result, nil
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
