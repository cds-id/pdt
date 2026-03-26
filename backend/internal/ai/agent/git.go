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

type GitAgent struct {
	DB     *gorm.DB
	UserID uint
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func (a *GitAgent) Name() string { return "git" }

func (a *GitAgent) SystemPrompt() string {
	return `You are a Git assistant for PDT. You help users explore their commit history, repository statistics, and code activity. Use the available tools to fetch data and provide insightful answers. Always be specific with numbers and dates.`
}

func (a *GitAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Name:        "search_commits",
			Description: "Search commits by message keyword, author, repo, or date range",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"keyword": {"type": "string", "description": "Search keyword in commit message"},
					"repo": {"type": "string", "description": "Repository name filter"},
					"since": {"type": "string", "description": "Start date (YYYY-MM-DD)"},
					"until": {"type": "string", "description": "End date (YYYY-MM-DD)"},
					"limit": {"type": "integer", "description": "Max results (default 20)"}
				}
			}`),
		},
		{
			Name:        "list_repos",
			Description: "List all tracked repositories for the user",
			InputSchema: json.RawMessage(`{"type": "object", "properties": {}}`),
		},
		{
			Name:        "get_repo_stats",
			Description: "Get commit statistics for a specific repository",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo": {"type": "string", "description": "Repository name"},
					"days": {"type": "integer", "description": "Number of days to look back (default 30)"}
				},
				"required": ["repo"]
			}`),
		},
		{
			Name:        "get_commit_detail",
			Description: "Get detailed information about a specific commit by SHA",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"sha": {"type": "string", "description": "Commit SHA (full or short)"}
				},
				"required": ["sha"]
			}`),
		},
	}
}

func (a *GitAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "search_commits":
		return a.searchCommits(args)
	case "list_repos":
		return a.listRepos()
	case "get_repo_stats":
		return a.getRepoStats(args)
	case "get_commit_detail":
		return a.getCommitDetail(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (a *GitAgent) searchCommits(args json.RawMessage) (any, error) {
	var params struct {
		Keyword string `json:"keyword"`
		Repo    string `json:"repo"`
		Since   string `json:"since"`
		Until   string `json:"until"`
		Limit   int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 20
	}

	query := a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ?", a.UserID).
		Preload("Repository")

	if params.Keyword != "" {
		query = query.Where("commits.message LIKE ?", "%"+params.Keyword+"%")
	}
	if params.Repo != "" {
		query = query.Where("repositories.name = ?", params.Repo)
	}
	if params.Since != "" {
		if t, err := time.Parse("2006-01-02", params.Since); err == nil {
			query = query.Where("commits.date >= ?", t)
		}
	}
	if params.Until != "" {
		if t, err := time.Parse("2006-01-02", params.Until); err == nil {
			query = query.Where("commits.date < ?", t.Add(24*time.Hour))
		}
	}

	var commits []models.Commit
	query.Order("commits.date desc").Limit(params.Limit).Find(&commits)

	type result struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Author  string `json:"author"`
		Date    string `json:"date"`
		Repo    string `json:"repo"`
		Branch  string `json:"branch"`
		JiraKey string `json:"jira_key,omitempty"`
	}
	var results []result
	for _, c := range commits {
		repoName := ""
		if c.Repository.Name != "" {
			repoName = c.Repository.Owner + "/" + c.Repository.Name
		}
		results = append(results, result{
			SHA:     shortSHA(c.SHA),
			Message: c.Message,
			Author:  c.Author,
			Date:    c.Date.Format("2006-01-02 15:04"),
			Repo:    repoName,
			Branch:  c.Branch,
			JiraKey: c.JiraCardKey,
		})
	}
	return results, nil
}

func (a *GitAgent) listRepos() (any, error) {
	var repos []models.Repository
	a.DB.Where("user_id = ?", a.UserID).Find(&repos)

	type result struct {
		Name     string `json:"name"`
		Owner    string `json:"owner"`
		Provider string `json:"provider"`
		URL      string `json:"url"`
	}
	var results []result
	for _, r := range repos {
		results = append(results, result{
			Name:     r.Name,
			Owner:    r.Owner,
			Provider: string(r.Provider),
			URL:      r.URL,
		})
	}
	return results, nil
}

func (a *GitAgent) getRepoStats(args json.RawMessage) (any, error) {
	var params struct {
		Repo string `json:"repo"`
		Days int    `json:"days"`
	}
	json.Unmarshal(args, &params)
	if params.Days == 0 {
		params.Days = 30
	}

	since := time.Now().AddDate(0, 0, -params.Days)

	var repo models.Repository
	if err := a.DB.Where("user_id = ? AND name = ?", a.UserID, params.Repo).First(&repo).Error; err != nil {
		return nil, fmt.Errorf("repository not found: %s", params.Repo)
	}

	var totalCommits int64
	a.DB.Model(&models.Commit{}).Where("repo_id = ? AND date >= ?", repo.ID, since).Count(&totalCommits)

	var linkedCommits int64
	a.DB.Model(&models.Commit{}).Where("repo_id = ? AND date >= ? AND has_link = ?", repo.ID, since, true).Count(&linkedCommits)

	type branchStat struct {
		Branch string `json:"branch"`
		Count  int64  `json:"count"`
	}
	var branches []branchStat
	a.DB.Model(&models.Commit{}).
		Select("branch, count(*) as count").
		Where("repo_id = ? AND date >= ?", repo.ID, since).
		Group("branch").Order("count desc").Limit(10).
		Scan(&branches)

	return map[string]any{
		"repo":           params.Repo,
		"period_days":    params.Days,
		"total_commits":  totalCommits,
		"linked_to_jira": linkedCommits,
		"top_branches":   branches,
	}, nil
}

func (a *GitAgent) getCommitDetail(args json.RawMessage) (any, error) {
	var params struct {
		SHA string `json:"sha"`
	}
	json.Unmarshal(args, &params)

	var commit models.Commit
	if err := a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.sha LIKE ?", a.UserID, params.SHA+"%").
		Preload("Repository").
		First(&commit).Error; err != nil {
		return nil, fmt.Errorf("commit not found: %s", params.SHA)
	}

	return map[string]any{
		"sha":          commit.SHA,
		"message":      commit.Message,
		"author":       commit.Author,
		"author_email": commit.AuthorEmail,
		"date":         commit.Date.Format("2006-01-02 15:04:05"),
		"branch":       commit.Branch,
		"repo":         commit.Repository.Owner + "/" + commit.Repository.Name,
		"jira_key":     commit.JiraCardKey,
		"has_link":     commit.HasLink,
	}, nil
}
