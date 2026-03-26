package report

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/jira"
	"gorm.io/gorm"
)

const DefaultTemplate = `# Daily Report — {{.DateFormatted}}

**Author:** {{.Author}}

## Summary
- **Commits:** {{.Stats.TotalCommits}}
- **Jira Cards:** {{.Stats.TotalCards}}
- **Repositories:** {{range $i, $r := .Stats.Repos}}{{if $i}}, {{end}}{{$r}}{{end}}

## Work Details
{{range .Cards}}
### {{.Key}} — {{.Summary}}
**Status:** {{.Status}}
{{range .Commits}}
- ` + "`{{.SHA}}`" + ` {{.Message}} ({{.Branch}}, {{.Time}})
{{end}}
{{end}}
{{if .UnlinkedCommits}}
## Other Commits
{{range .UnlinkedCommits}}
- ` + "`{{.SHA}}`" + ` {{.Message}} ({{.Repo}}/{{.Branch}}, {{.Time}})
{{end}}
{{end}}`

type ReportData struct {
	Date            string
	DateFormatted   string
	Author          string
	Cards           []CardReport
	UnlinkedCommits []CommitReport
	Stats           ReportStats
}

type CardReport struct {
	Key     string
	Summary string
	Status  string
	Commits []CommitReport
}

type CommitReport struct {
	SHA     string
	Message string
	Branch  string
	Repo    string
	Time    string
}

type ReportStats struct {
	TotalCommits int
	TotalCards   int
	Repos        []string
}

type Generator struct {
	DB        *gorm.DB
	Encryptor *crypto.Encryptor // nil = skip Jira API fallback
}

func NewGenerator(db *gorm.DB, enc *crypto.Encryptor) *Generator {
	return &Generator{DB: db, Encryptor: enc}
}

// BuildReportData aggregates commits and Jira cards for a user on a given date.
func (g *Generator) BuildReportData(userID uint, date time.Time) (*ReportData, error) {
	var user models.User
	if err := g.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	var commits []models.Commit
	g.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.date >= ? AND commits.date < ?", userID, dayStart, dayEnd).
		Preload("Repository").
		Order("commits.date asc").
		Find(&commits)

	cardCommits := map[string][]CommitReport{}
	var unlinked []CommitReport
	repoSet := map[string]bool{}

	for _, c := range commits {
		repoName := fmt.Sprintf("%s/%s", c.Repository.Owner, c.Repository.Name)
		repoSet[repoName] = true

		cr := CommitReport{
			SHA:     shortSHA(c.SHA),
			Message: firstLine(c.Message),
			Branch:  c.Branch,
			Repo:    repoName,
			Time:    c.Date.Format("15:04"),
		}

		if c.JiraCardKey != "" {
			cardCommits[c.JiraCardKey] = append(cardCommits[c.JiraCardKey], cr)
		} else {
			unlinked = append(unlinked, cr)
		}
	}

	// Build Jira client for API fallback (if user has Jira configured)
	var jiraClient *jira.Client
	if g.Encryptor != nil && user.JiraToken != "" && user.JiraWorkspace != "" && user.JiraEmail != "" {
		token, err := g.Encryptor.Decrypt(user.JiraToken)
		if err == nil {
			jiraClient = jira.New(user.JiraWorkspace, user.JiraEmail, token)
		}
	}

	var cards []CardReport
	for key, crs := range cardCommits {
		card := CardReport{
			Key:     key,
			Commits: crs,
		}

		// Try DB first
		var jiraCard models.JiraCard
		if err := g.DB.Where("user_id = ? AND card_key = ?", userID, key).First(&jiraCard).Error; err == nil {
			card.Summary = jiraCard.Summary
			card.Status = jiraCard.Status
		} else if jiraClient != nil {
			// Fallback: fetch from Jira API
			if issue, err := jiraClient.FetchIssue(key); err == nil {
				card.Summary = issue.Summary
				card.Status = issue.Status
				log.Printf("[report] fetched Jira card %s from API: %s (%s)", key, issue.Summary, issue.Status)
			} else {
				log.Printf("[report] failed to fetch Jira card %s: %v", key, err)
			}
		}

		cards = append(cards, card)
	}

	var repos []string
	for r := range repoSet {
		repos = append(repos, r)
	}

	data := &ReportData{
		Date:            date.Format("2006-01-02"),
		DateFormatted:   date.Format("Monday, 02 January 2006"),
		Author:          user.Email,
		Cards:           cards,
		UnlinkedCommits: unlinked,
		Stats: ReportStats{
			TotalCommits: len(commits),
			TotalCards:   len(cards),
			Repos:        repos,
		},
	}

	return data, nil
}

type MonthlyReportData struct {
	Month           int
	Year            int
	MonthName       string
	Author          string
	TotalCommits    int
	TotalCards      int
	CardsCompleted  int
	CardsInProgress int
	WeeklyBreakdown []WeekStats
	RepoBreakdown   []RepoStats
	TopCards        []CardReport
	DailyReports    []DailyReportSummary
}

type WeekStats struct {
	WeekNumber int    `json:"week_number"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Commits    int    `json:"commits"`
	Cards      int    `json:"cards"`
}

type RepoStats struct {
	Repo    string `json:"repo"`
	Commits int    `json:"commits"`
}

type DailyReportSummary struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Commits int    `json:"commits"`
	Cards   int    `json:"cards"`
}

func (g *Generator) BuildMonthlyReportData(userID uint, month, year int) (*MonthlyReportData, error) {
	var user models.User
	if err := g.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, 0)

	// Fetch all commits for the month
	var commits []models.Commit
	g.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.date >= ? AND commits.date < ?", userID, monthStart, monthEnd).
		Preload("Repository").
		Order("commits.date asc").
		Find(&commits)

	// Fetch all cards worked on (cards with commits this month)
	cardKeys := make(map[string]bool)
	for _, c := range commits {
		if c.JiraCardKey != "" {
			cardKeys[c.JiraCardKey] = true
		}
	}

	var cards []models.JiraCard
	if len(cardKeys) > 0 {
		keys := make([]string, 0, len(cardKeys))
		for k := range cardKeys {
			keys = append(keys, k)
		}
		g.DB.Where("user_id = ? AND card_key IN ?", userID, keys).Find(&cards)
	}

	// Build weekly breakdown
	var weeklyBreakdown []WeekStats
	current := monthStart
	weekNum := 1
	for current.Before(monthEnd) {
		weekEnd := current.AddDate(0, 0, 7)
		if weekEnd.After(monthEnd) {
			weekEnd = monthEnd
		}
		weekCommits := 0
		weekCards := make(map[string]bool)
		for _, c := range commits {
			if !c.Date.Before(current) && c.Date.Before(weekEnd) {
				weekCommits++
				if c.JiraCardKey != "" {
					weekCards[c.JiraCardKey] = true
				}
			}
		}
		weeklyBreakdown = append(weeklyBreakdown, WeekStats{
			WeekNumber: weekNum,
			StartDate:  current.Format("2006-01-02"),
			EndDate:    weekEnd.AddDate(0, 0, -1).Format("2006-01-02"),
			Commits:    weekCommits,
			Cards:      len(weekCards),
		})
		current = weekEnd
		weekNum++
	}

	// Build repo breakdown
	repoMap := make(map[string]int)
	for _, c := range commits {
		repoName := c.Repository.Owner + "/" + c.Repository.Name
		repoMap[repoName]++
	}
	var repoBreakdown []RepoStats
	for repo, count := range repoMap {
		repoBreakdown = append(repoBreakdown, RepoStats{Repo: repo, Commits: count})
	}

	// Build top cards by commit count
	var topCards []CardReport
	for _, card := range cards {
		topCards = append(topCards, CardReport{
			Key:     card.Key,
			Summary: card.Summary,
			Status:  card.Status,
		})
	}

	// Count completed and in-progress
	completed := 0
	inProgress := 0
	for _, card := range cards {
		if card.Status == "Done" || card.Status == "Closed" {
			completed++
		} else {
			inProgress++
		}
	}

	// Fetch daily reports for the month
	var dailyReports []models.Report
	g.DB.Where("user_id = ? AND report_type = ? AND date >= ? AND date < ?",
		userID, "daily", monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02")).
		Order("date asc").Find(&dailyReports)

	var dailySummaries []DailyReportSummary
	for _, r := range dailyReports {
		dailySummaries = append(dailySummaries, DailyReportSummary{
			Date:  r.Date,
			Title: r.Title,
		})
	}

	return &MonthlyReportData{
		Month:           month,
		Year:            year,
		MonthName:       time.Month(month).String(),
		Author:          user.Email,
		TotalCommits:    len(commits),
		TotalCards:      len(cards),
		CardsCompleted:  completed,
		CardsInProgress: inProgress,
		WeeklyBreakdown: weeklyBreakdown,
		RepoBreakdown:   repoBreakdown,
		TopCards:        topCards,
		DailyReports:    dailySummaries,
	}, nil
}

const DefaultMonthlyTemplate = `# Monthly Report — {{.MonthName}} {{.Year}}

**Author:** {{.Author}}

## Summary
- **Total Commits:** {{.TotalCommits}}
- **Total Jira Cards Worked On:** {{.TotalCards}}
- **Cards Completed:** {{.CardsCompleted}}
- **Cards In Progress:** {{.CardsInProgress}}

## Weekly Breakdown
{{range .WeeklyBreakdown}}
### Week {{.WeekNumber}} ({{.StartDate}} — {{.EndDate}})
- Commits: {{.Commits}}
- Cards: {{.Cards}}
{{end}}

## Repository Activity
{{range .RepoBreakdown}}
- **{{.Repo}}**: {{.Commits}} commits
{{end}}

## Top Jira Cards
{{range .TopCards}}
- **{{.Key}}** — {{.Summary}} ({{.Status}})
{{end}}
`

func (g *Generator) GetMonthlyTemplateContent(userID uint) string {
	var tmpl models.ReportTemplate
	if g.DB.Where("user_id = ? AND name = ? AND is_default = ?", userID, "Monthly Default", true).First(&tmpl).Error == nil {
		return tmpl.Content
	}
	return DefaultMonthlyTemplate
}

// RenderMonthly renders a template string with MonthlyReportData.
func (g *Generator) RenderMonthly(templateContent string, data *MonthlyReportData) (string, error) {
	tmpl, err := template.New("monthly_report").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("invalid template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.String(), nil
}

// Render renders a template string with the given data.
func (g *Generator) Render(templateContent string, data *ReportData) (string, error) {
	tmpl, err := template.New("report").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("invalid template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.String(), nil
}

// GetTemplateContent returns the template content for a user.
// Priority: specific template_id > user's default > built-in default.
func (g *Generator) GetTemplateContent(userID uint, templateID *uint) (string, *uint) {
	if templateID != nil {
		var tmpl models.ReportTemplate
		if err := g.DB.Where("id = ? AND user_id = ?", *templateID, userID).First(&tmpl).Error; err == nil {
			return tmpl.Content, &tmpl.ID
		}
	}

	var tmpl models.ReportTemplate
	if err := g.DB.Where("user_id = ? AND is_default = ?", userID, true).First(&tmpl).Error; err == nil {
		return tmpl.Content, &tmpl.ID
	}

	return DefaultTemplate, nil
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func firstLine(msg string) string {
	lines := strings.SplitN(msg, "\n", 2)
	line := lines[0]
	if len(line) > 80 {
		return line[:77] + "..."
	}
	return line
}
