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

type ProofAgent struct {
	DB     *gorm.DB
	UserID uint
}

func (a *ProofAgent) Name() string { return "proof" }

func (a *ProofAgent) SystemPrompt() string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`You are a Proof & Accountability assistant for PDT. Today is %s. You help developers find evidence of discussions, decisions, and requirements stated in Jira comments. You can search comments by author, keyword, and date to find proof of what was said and when. You also detect quality issues like cards with missing descriptions or incomplete requirements. Always cite the exact comment author, date, and card key when presenting evidence.`, today)
}

func (a *ProofAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Name:        "search_comments",
			Description: "Search all Jira comments by keyword, author, card key, and/or date range",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"keyword":  {"type": "string", "description": "Keyword to search in comment body"},
					"author":   {"type": "string", "description": "Filter by comment author name"},
					"card_key": {"type": "string", "description": "Filter by Jira card key (e.g., PDT-123)"},
					"since":    {"type": "string", "description": "Start date filter (YYYY-MM-DD)"},
					"until":    {"type": "string", "description": "End date filter (YYYY-MM-DD)"},
					"limit":    {"type": "integer", "description": "Max results (default 20)"}
				}
			}`),
		},
		{
			Name:        "get_card_comments",
			Description: "Get all comments on a specific Jira card in chronological order",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"card_key": {"type": "string", "description": "Jira card key (e.g., PDT-123)"}
				},
				"required": ["card_key"]
			}`),
		},
		{
			Name:        "find_person_statements",
			Description: "Find all comments made by a specific person (matched by name or email), optionally filtered by keyword",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"person":  {"type": "string", "description": "Person name or email to search for"},
					"keyword": {"type": "string", "description": "Optional keyword to filter comments"},
					"limit":   {"type": "integer", "description": "Max results (default 30)"}
				},
				"required": ["person"]
			}`),
		},
		{
			Name:        "get_comment_timeline",
			Description: "Get a chronological timeline of who said what on a specific Jira card",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"card_key": {"type": "string", "description": "Jira card key (e.g., PDT-123)"}
				},
				"required": ["card_key"]
			}`),
		},
		{
			Name:        "detect_quality_issues",
			Description: "Find cards in active sprints that have quality problems such as missing descriptions, no comments, or parent cards with no description",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"sprint_id": {"type": "integer", "description": "Optional sprint ID to limit the check; defaults to all active sprints"},
					"sprint_name": {"type": "string", "description": "Optional sprint name (e.g., 'BNS Sprint 13')"},
					"assignee": {"type": "string", "description": "Optional assignee name filter (partial match)"}
				}
			}`),
		},
		{
			Name:        "check_requirement_coverage",
			Description: "Check if commits linked to a card cover the card's description and requirements",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"card_key": {"type": "string", "description": "Jira card key (e.g., PDT-123)"}
				},
				"required": ["card_key"]
			}`),
		},
	}
}

func (a *ProofAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "search_comments":
		return a.searchComments(args)
	case "get_card_comments":
		return a.getCardComments(args)
	case "find_person_statements":
		return a.findPersonStatements(args)
	case "get_comment_timeline":
		return a.getCommentTimeline(args)
	case "detect_quality_issues":
		return a.detectQualityIssues(args)
	case "check_requirement_coverage":
		return a.checkRequirementCoverage(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (a *ProofAgent) searchComments(args json.RawMessage) (any, error) {
	var params struct {
		Keyword string `json:"keyword"`
		Author  string `json:"author"`
		CardKey string `json:"card_key"`
		Since   string `json:"since"`
		Until   string `json:"until"`
		Limit   int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 20
	}

	query := a.DB.Where("user_id = ?", a.UserID)
	if params.Keyword != "" {
		query = query.Where("body LIKE ?", "%"+params.Keyword+"%")
	}
	if params.Author != "" {
		query = query.Where("author LIKE ?", "%"+params.Author+"%")
	}
	if params.CardKey != "" {
		query = query.Where("card_key = ?", params.CardKey)
	}
	if params.Since != "" {
		if t, err := time.Parse("2006-01-02", params.Since); err == nil {
			query = query.Where("commented_at >= ?", t)
		}
	}
	if params.Until != "" {
		if t, err := time.Parse("2006-01-02", params.Until); err == nil {
			query = query.Where("commented_at <= ?", t.Add(24*time.Hour-time.Second))
		}
	}

	var comments []models.JiraComment
	query.Order("commented_at DESC").Limit(params.Limit).Find(&comments)

	type result struct {
		CardKey     string `json:"card_key"`
		Author      string `json:"author"`
		AuthorEmail string `json:"author_email"`
		Body        string `json:"body"`
		CommentedAt string `json:"commented_at"`
	}
	var results []result
	for _, c := range comments {
		results = append(results, result{
			CardKey:     c.CardKey,
			Author:      c.Author,
			AuthorEmail: c.AuthorEmail,
			Body:        c.Body,
			CommentedAt: c.CommentedAt.Format("2006-01-02 15:04"),
		})
	}
	return results, nil
}

func (a *ProofAgent) getCardComments(args json.RawMessage) (any, error) {
	var params struct {
		CardKey string `json:"card_key"`
	}
	json.Unmarshal(args, &params)
	if params.CardKey == "" {
		return nil, fmt.Errorf("card_key is required")
	}

	var comments []models.JiraComment
	a.DB.Where("user_id = ? AND card_key = ?", a.UserID, params.CardKey).
		Order("commented_at ASC").
		Find(&comments)

	type result struct {
		Author      string `json:"author"`
		AuthorEmail string `json:"author_email"`
		Body        string `json:"body"`
		CommentedAt string `json:"commented_at"`
	}
	var results []result
	for _, c := range comments {
		results = append(results, result{
			Author:      c.Author,
			AuthorEmail: c.AuthorEmail,
			Body:        c.Body,
			CommentedAt: c.CommentedAt.Format("2006-01-02 15:04"),
		})
	}
	return results, nil
}

func (a *ProofAgent) findPersonStatements(args json.RawMessage) (any, error) {
	var params struct {
		Person  string `json:"person"`
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Person == "" {
		return nil, fmt.Errorf("person is required")
	}
	if params.Limit == 0 {
		params.Limit = 30
	}

	likePattern := "%" + params.Person + "%"
	query := a.DB.Where("user_id = ? AND (author LIKE ? OR author_email LIKE ?)", a.UserID, likePattern, likePattern)
	if params.Keyword != "" {
		query = query.Where("body LIKE ?", "%"+params.Keyword+"%")
	}

	var comments []models.JiraComment
	query.Order("commented_at DESC").Limit(params.Limit).Find(&comments)

	type result struct {
		CardKey     string `json:"card_key"`
		Author      string `json:"author"`
		AuthorEmail string `json:"author_email"`
		Body        string `json:"body"`
		CommentedAt string `json:"commented_at"`
	}
	var results []result
	for _, c := range comments {
		results = append(results, result{
			CardKey:     c.CardKey,
			Author:      c.Author,
			AuthorEmail: c.AuthorEmail,
			Body:        c.Body,
			CommentedAt: c.CommentedAt.Format("2006-01-02 15:04"),
		})
	}
	return results, nil
}

func (a *ProofAgent) getCommentTimeline(args json.RawMessage) (any, error) {
	var params struct {
		CardKey string `json:"card_key"`
	}
	json.Unmarshal(args, &params)
	if params.CardKey == "" {
		return nil, fmt.Errorf("card_key is required")
	}

	var comments []models.JiraComment
	a.DB.Where("user_id = ? AND card_key = ?", a.UserID, params.CardKey).
		Order("commented_at ASC").
		Find(&comments)

	type entry struct {
		Author string `json:"author"`
		Date   string `json:"date"`
		Snippet string `json:"snippet"`
	}
	var timeline []entry
	for _, c := range comments {
		snippet := c.Body
		if len([]rune(snippet)) > 200 {
			runes := []rune(snippet)
			snippet = string(runes[:200]) + "..."
		}
		timeline = append(timeline, entry{
			Author:  c.Author,
			Date:    c.CommentedAt.Format("2006-01-02 15:04"),
			Snippet: snippet,
		})
	}
	return timeline, nil
}

func (a *ProofAgent) detectQualityIssues(args json.RawMessage) (any, error) {
	var params struct {
		SprintID   int    `json:"sprint_id"`
		SprintName string `json:"sprint_name"`
		Assignee   string `json:"assignee"`
	}
	json.Unmarshal(args, &params)

	// Find sprint IDs
	var sprintIDs []uint
	if params.SprintID > 0 {
		sprintIDs = []uint{uint(params.SprintID)}
	} else if params.SprintName != "" {
		var sprint models.Sprint
		if err := a.DB.Where("user_id = ? AND name = ?", a.UserID, params.SprintName).First(&sprint).Error; err == nil {
			sprintIDs = []uint{sprint.ID}
		}
	} else {
		var sprints []models.Sprint
		a.DB.Where("user_id = ? AND state = ?", a.UserID, models.SprintActive).Find(&sprints)
		for _, s := range sprints {
			sprintIDs = append(sprintIDs, s.ID)
		}
	}

	if len(sprintIDs) == 0 {
		return []any{}, nil
	}

	// Fetch cards in those sprints
	query := a.DB.Where("user_id = ? AND sprint_id IN ?", a.UserID, sprintIDs)
	if params.Assignee != "" {
		query = query.Where("assignee LIKE ?", "%"+params.Assignee+"%")
	}
	var cards []models.JiraCard
	query.Find(&cards)

	type issueCard struct {
		Key      string   `json:"key"`
		Summary  string   `json:"summary"`
		Status   string   `json:"status"`
		Assignee string   `json:"assignee"`
		Issues   []string `json:"issues"`
	}

	var problematic []issueCard
	for _, card := range cards {
		var issues []string

		// Check 1: missing description
		hasDescription := false
		if card.DetailsJSON != "" {
			var details map[string]any
			if err := json.Unmarshal([]byte(card.DetailsJSON), &details); err == nil {
				if desc, ok := details["description"]; ok && desc != nil {
					switch v := desc.(type) {
					case string:
						hasDescription = v != ""
					default:
						hasDescription = true
					}
				}
			}
		}
		if !hasDescription {
			issues = append(issues, "no description")
		}

		// Check 2: no comments at all
		var commentCount int64
		a.DB.Model(&models.JiraComment{}).Where("user_id = ? AND card_key = ?", a.UserID, card.Key).Count(&commentCount)
		if commentCount == 0 {
			issues = append(issues, "no comments")
		}

		// Check 3: parent card has no description
		if card.DetailsJSON != "" {
			var details map[string]any
			if err := json.Unmarshal([]byte(card.DetailsJSON), &details); err == nil {
				if parent, ok := details["parent"]; ok && parent != nil {
					parentKey := ""
					switch p := parent.(type) {
					case map[string]any:
						if k, ok := p["key"].(string); ok {
							parentKey = k
						}
					case string:
						parentKey = p
					}
					if parentKey != "" {
						var parentCard models.JiraCard
						if err := a.DB.Where("user_id = ? AND card_key = ?", a.UserID, parentKey).First(&parentCard).Error; err == nil {
							parentHasDesc := false
							if parentCard.DetailsJSON != "" {
								var pd map[string]any
								if json.Unmarshal([]byte(parentCard.DetailsJSON), &pd) == nil {
									if desc, ok := pd["description"]; ok && desc != nil {
										switch v := desc.(type) {
										case string:
											parentHasDesc = v != ""
										default:
											parentHasDesc = true
										}
									}
								}
							}
							if !parentHasDesc {
								issues = append(issues, fmt.Sprintf("parent card %s has no description", parentKey))
							}
						}
					}
				}
			}
		}

		if len(issues) > 0 {
			problematic = append(problematic, issueCard{
				Key:      card.Key,
				Summary:  card.Summary,
				Status:   card.Status,
				Assignee: card.Assignee,
				Issues:   issues,
			})
		}
	}

	return problematic, nil
}

func (a *ProofAgent) checkRequirementCoverage(args json.RawMessage) (any, error) {
	var params struct {
		CardKey string `json:"card_key"`
	}
	json.Unmarshal(args, &params)
	if params.CardKey == "" {
		return nil, fmt.Errorf("card_key is required")
	}

	var card models.JiraCard
	if err := a.DB.Where("user_id = ? AND card_key = ?", a.UserID, params.CardKey).First(&card).Error; err != nil {
		return nil, fmt.Errorf("card not found: %s", params.CardKey)
	}

	// Parse description from DetailsJSON
	description := ""
	if card.DetailsJSON != "" {
		var details map[string]any
		if err := json.Unmarshal([]byte(card.DetailsJSON), &details); err == nil {
			if desc, ok := details["description"]; ok && desc != nil {
				switch v := desc.(type) {
				case string:
					description = v
				default:
					if b, err := json.Marshal(v); err == nil {
						description = string(b)
					}
				}
			}
		}
	}

	// Fetch linked commits
	var commits []models.Commit
	a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.jira_card_key = ?", a.UserID, params.CardKey).
		Find(&commits)

	type commitInfo struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Author  string `json:"author"`
		Date    string `json:"date"`
	}
	var linkedCommits []commitInfo
	for _, c := range commits {
		linkedCommits = append(linkedCommits, commitInfo{
			SHA:     shortSHA(c.SHA),
			Message: c.Message,
			Author:  c.Author,
			Date:    c.Date.Format("2006-01-02 15:04"),
		})
	}

	return map[string]any{
		"key":         card.Key,
		"summary":     card.Summary,
		"description": description,
		"commits":     linkedCommits,
	}, nil
}
