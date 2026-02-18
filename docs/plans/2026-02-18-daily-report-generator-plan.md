# Daily Report Generator Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a template-based daily report generator that aggregates commits and Jira cards into formal Markdown reports for fee release.

**Architecture:** Report service aggregates commit + Jira data for a date, renders via Go `text/template`, stores in DB. API for on-demand generation + template CRUD. Worker auto-generates daily at configurable time.

**Tech Stack:** Go `text/template`, GORM models, existing Gin handlers pattern.

---

### Task 1: Add Report models and migrate

**Files:**
- Create: `backend/internal/models/report.go`
- Modify: `backend/internal/database/database.go`

**Step 1: Create the report models**

Create `backend/internal/models/report.go`:

```go
package models

import "time"

type ReportTemplate struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	IsDefault bool      `gorm:"default:false" json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
}

type Report struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	TemplateID *uint     `gorm:"index" json:"template_id"`
	Date       string    `gorm:"type:varchar(10);index;not null" json:"date"`
	Title      string    `gorm:"type:varchar(500)" json:"title"`
	Content    string    `gorm:"type:text" json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	User       User      `gorm:"foreignKey:UserID" json:"-"`
}
```

**Step 2: Add models to migration**

In `backend/internal/database/database.go`, add to the `Migrate` function:

```go
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Repository{},
		&models.Commit{},
		&models.CommitCardLink{},
		&models.Sprint{},
		&models.JiraCard{},
		&models.ReportTemplate{},
		&models.Report{},
	)
}
```

**Step 3: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 4: Commit**

```bash
git add backend/internal/models/report.go backend/internal/database/database.go
git commit -m "feat: add Report and ReportTemplate models"
```

---

### Task 2: Create the report generator service

**Files:**
- Create: `backend/internal/services/report/report.go`

**Step 1: Create the report service**

Create `backend/internal/services/report/report.go`:

```go
package report

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/cds-id/pdt/backend/internal/models"
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
	DB *gorm.DB
}

func NewGenerator(db *gorm.DB) *Generator {
	return &Generator{DB: db}
}

// BuildReportData aggregates commits and Jira cards for a user on a given date.
func (g *Generator) BuildReportData(userID uint, date time.Time) (*ReportData, error) {
	var user models.User
	if err := g.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	// Fetch commits for the day across all user's repos
	var commits []models.Commit
	g.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.date >= ? AND commits.date < ?", userID, dayStart, dayEnd).
		Preload("Repository").
		Order("commits.date asc").
		Find(&commits)

	// Group commits by Jira card key
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

	// Build card reports with Jira info from DB
	var cards []CardReport
	for key, crs := range cardCommits {
		card := CardReport{
			Key:     key,
			Commits: crs,
		}

		// Try to get Jira card info from DB
		var jiraCard models.JiraCard
		if err := g.DB.Where("user_id = ? AND card_key = ?", userID, key).First(&jiraCard).Error; err == nil {
			card.Summary = jiraCard.Summary
			card.Status = jiraCard.Status
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

	// Try user's default template
	var tmpl models.ReportTemplate
	if err := g.DB.Where("user_id = ? AND is_default = ?", userID, true).First(&tmpl).Error; err == nil {
		return tmpl.Content, &tmpl.ID
	}

	// Built-in default
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
```

**Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 3: Commit**

```bash
git add backend/internal/services/report/report.go
git commit -m "feat: add report generator service with template rendering"
```

---

### Task 3: Create report and template API handlers

**Files:**
- Create: `backend/internal/handlers/report.go`

**Step 1: Create the report handler**

Create `backend/internal/handlers/report.go`:

```go
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReportHandler struct {
	DB        *gorm.DB
	Generator *report.Generator
}

// --- Report Generation ---

func (h *ReportHandler) Generate(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Date       string `json:"date"`
		TemplateID *uint  `json:"template_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Date == "" {
		req.Date = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	data, err := h.Generator.BuildReportData(userID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	templateContent, templateID := h.Generator.GetTemplateContent(userID, req.TemplateID)

	rendered, err := h.Generator.Render(templateContent, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template error: " + err.Error()})
		return
	}

	rpt := models.Report{
		UserID:     userID,
		TemplateID: templateID,
		Date:       req.Date,
		Title:      "Daily Report — " + date.Format("Monday, 02 January 2006"),
		Content:    rendered,
	}

	// Upsert: if report for this date already exists, update it
	var existing models.Report
	if err := h.DB.Where("user_id = ? AND date = ?", userID, req.Date).First(&existing).Error; err == nil {
		existing.Content = rendered
		existing.Title = rpt.Title
		existing.TemplateID = templateID
		h.DB.Save(&existing)
		c.JSON(http.StatusOK, existing)
		return
	}

	h.DB.Create(&rpt)
	c.JSON(http.StatusCreated, rpt)
}

func (h *ReportHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	from := c.Query("from")
	to := c.Query("to")

	query := h.DB.Where("user_id = ?", userID)
	if from != "" {
		query = query.Where("date >= ?", from)
	}
	if to != "" {
		query = query.Where("date <= ?", to)
	}

	var reports []models.Report
	query.Order("date desc").Find(&reports)

	c.JSON(http.StatusOK, reports)
}

func (h *ReportHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var rpt models.Report
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&rpt).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	c.JSON(http.StatusOK, rpt)
}

func (h *ReportHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	result := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Report{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "report deleted"})
}

// --- Template Management ---

func (h *ReportHandler) ListTemplates(c *gin.Context) {
	userID := c.GetUint("user_id")

	var templates []models.ReportTemplate
	h.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&templates)

	c.JSON(http.StatusOK, templates)
}

func (h *ReportHandler) CreateTemplate(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Name      string `json:"name" binding:"required"`
		Content   string `json:"content" binding:"required"`
		IsDefault bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and content are required"})
		return
	}

	// If setting as default, unset other defaults
	if req.IsDefault {
		h.DB.Model(&models.ReportTemplate{}).Where("user_id = ?", userID).Update("is_default", false)
	}

	tmpl := models.ReportTemplate{
		UserID:    userID,
		Name:      req.Name,
		Content:   req.Content,
		IsDefault: req.IsDefault,
	}
	h.DB.Create(&tmpl)

	c.JSON(http.StatusCreated, tmpl)
}

func (h *ReportHandler) UpdateTemplate(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var tmpl models.ReportTemplate
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&tmpl).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	var req struct {
		Name      *string `json:"name"`
		Content   *string `json:"content"`
		IsDefault *bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Name != nil {
		tmpl.Name = *req.Name
	}
	if req.Content != nil {
		tmpl.Content = *req.Content
	}
	if req.IsDefault != nil && *req.IsDefault {
		h.DB.Model(&models.ReportTemplate{}).Where("user_id = ? AND id != ?", userID, tmpl.ID).Update("is_default", false)
		tmpl.IsDefault = true
	} else if req.IsDefault != nil {
		tmpl.IsDefault = *req.IsDefault
	}

	h.DB.Save(&tmpl)
	c.JSON(http.StatusOK, tmpl)
}

func (h *ReportHandler) DeleteTemplate(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	result := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.ReportTemplate{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
}

func (h *ReportHandler) PreviewTemplate(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Content string `json:"content" binding:"required"`
		Date    string `json:"date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	dateStr := req.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	data, err := h.Generator.BuildReportData(userID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rendered, err := h.Generator.Render(req.Content, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rendered": rendered,
		"stats": gin.H{
			"total_commits": data.Stats.TotalCommits,
			"total_cards":   data.Stats.TotalCards,
		},
	})
}
```

Note: `strconv` import may be unused — remove it if the compiler complains.

**Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors (remove unused imports if any).

**Step 3: Commit**

```bash
git add backend/internal/handlers/report.go
git commit -m "feat: add report and template API handlers"
```

---

### Task 4: Wire report routes into main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

**Step 1: Add report handler and routes**

In `backend/cmd/server/main.go`:

1. Add import: `"github.com/cds-id/pdt/backend/internal/services/report"`

2. After the jiraHandler line, add:

```go
	reportGen := report.NewGenerator(db)
	reportHandler := &handlers.ReportHandler{DB: db, Generator: reportGen}
```

3. After the jira routes block, add:

```go
			reports := protected.Group("/reports")
			{
				reports.POST("/generate", reportHandler.Generate)
				reports.GET("", reportHandler.List)
				reports.GET("/:id", reportHandler.Get)
				reports.DELETE("/:id", reportHandler.Delete)

				templates := reports.Group("/templates")
				{
					templates.GET("", reportHandler.ListTemplates)
					templates.POST("", reportHandler.CreateTemplate)
					templates.PUT("/:id", reportHandler.UpdateTemplate)
					templates.DELETE("/:id", reportHandler.DeleteTemplate)
					templates.POST("/preview", reportHandler.PreviewTemplate)
				}
			}
```

**Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 3: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: wire report routes into server"
```

---

### Task 5: Add report config and worker auto-generation

**Files:**
- Modify: `backend/internal/config/config.go`
- Create: `backend/internal/worker/reports.go`
- Modify: `backend/internal/worker/scheduler.go`
- Modify: `backend/.env`
- Modify: `backend/.env.example`

**Step 1: Add report config fields**

In `backend/internal/config/config.go`, add to Config struct:

```go
	ReportAutoGenerate bool
	ReportAutoTime     string
```

In `Load()`, add before the validation block:

```go
	reportAutoGen := getEnv("REPORT_AUTO_GENERATE", "true")
	cfg.ReportAutoGenerate = reportAutoGen == "true" || reportAutoGen == "1"
	cfg.ReportAutoTime = getEnv("REPORT_AUTO_TIME", "23:00")
```

**Step 2: Add env vars to .env and .env.example**

Append to both:

```
# Reports
REPORT_AUTO_GENERATE=true
REPORT_AUTO_TIME=23:00
```

**Step 3: Create the worker report generation function**

Create `backend/internal/worker/reports.go`:

```go
package worker

import (
	"log"
	"time"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"gorm.io/gorm"
)

// AutoGenerateReports generates daily reports for all users who don't have one for today.
func AutoGenerateReports(db *gorm.DB) {
	today := time.Now().Format("2006-01-02")

	var users []models.User
	db.Find(&users)

	gen := report.NewGenerator(db)

	for _, user := range users {
		// Skip if report already exists for today
		var count int64
		db.Model(&models.Report{}).Where("user_id = ? AND date = ?", user.ID, today).Count(&count)
		if count > 0 {
			continue
		}

		date := time.Now()
		data, err := gen.BuildReportData(user.ID, date)
		if err != nil {
			log.Printf("[worker] report generation failed for user %d: %v", user.ID, err)
			continue
		}

		if data.Stats.TotalCommits == 0 {
			continue // Skip users with no commits today
		}

		templateContent, templateID := gen.GetTemplateContent(user.ID, nil)

		rendered, err := gen.Render(templateContent, data)
		if err != nil {
			log.Printf("[worker] report render failed for user %d: %v", user.ID, err)
			continue
		}

		rpt := models.Report{
			UserID:     user.ID,
			TemplateID: templateID,
			Date:       today,
			Title:      "Daily Report — " + date.Format("Monday, 02 January 2006"),
			Content:    rendered,
		}
		db.Create(&rpt)

		log.Printf("[worker] report generated for user %d: %d commits, %d cards",
			user.ID, data.Stats.TotalCommits, data.Stats.TotalCards)
	}
}
```

**Step 4: Add report generation to scheduler**

In `backend/internal/worker/scheduler.go`:

1. Add fields to the Scheduler struct:

```go
	ReportAutoGenerate bool
	ReportAutoTime     string
	reportRunning      atomic.Bool
	lastReportDate     string
```

2. Update `NewScheduler` to accept new params. Change the signature and body:

```go
func NewScheduler(db *gorm.DB, enc *crypto.Encryptor, commitInterval, jiraInterval time.Duration, reportAutoGen bool, reportAutoTime string) *Scheduler {
	return &Scheduler{
		DB:                 db,
		Encryptor:          enc,
		CommitInterval:     commitInterval,
		JiraInterval:       jiraInterval,
		Status:             NewSyncStatus(),
		ReportAutoGenerate: reportAutoGen,
		ReportAutoTime:     reportAutoTime,
	}
}
```

3. In `Start()`, add the report loop:

```go
func (s *Scheduler) Start(ctx context.Context) {
	log.Printf("[worker] starting scheduler: commits=%s, jira=%s, reports=%v at %s",
		s.CommitInterval, s.JiraInterval, s.ReportAutoGenerate, s.ReportAutoTime)

	go s.commitSyncLoop(ctx)
	go s.jiraSyncLoop(ctx)
	if s.ReportAutoGenerate {
		go s.reportLoop(ctx)
	}
}
```

4. Add the report loop method:

```go
func (s *Scheduler) reportLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] report loop stopped")
			return
		case <-ticker.C:
			s.checkAndGenerateReport()
		}
	}
}

func (s *Scheduler) checkAndGenerateReport() {
	now := time.Now()
	today := now.Format("2006-01-02")

	// Already generated today
	if s.lastReportDate == today {
		return
	}

	// Check if it's past the configured time
	currentTime := now.Format("15:04")
	if currentTime < s.ReportAutoTime {
		return
	}

	if !s.reportRunning.CompareAndSwap(false, true) {
		return
	}
	defer s.reportRunning.Store(false)

	log.Println("[worker] auto-generating daily reports")
	AutoGenerateReports(s.DB)
	s.lastReportDate = today
	log.Println("[worker] daily report generation completed")
}
```

**Step 5: Update main.go to pass report config to scheduler**

In `backend/cmd/server/main.go`, update the scheduler creation:

```go
	scheduler = worker.NewScheduler(db, encryptor, cfg.SyncIntervalCommits, cfg.SyncIntervalJira, cfg.ReportAutoGenerate, cfg.ReportAutoTime)
```

**Step 6: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 7: Commit**

```bash
git add backend/internal/config/config.go backend/internal/worker/reports.go backend/internal/worker/scheduler.go backend/cmd/server/main.go backend/.env backend/.env.example
git commit -m "feat: add daily report auto-generation worker"
```

---

### Task 6: Update SIT test router and add report SIT test

**Files:**
- Modify: `backend/tests/sit/gitlab_sit_test.go` (setupRouter)
- Create or modify: `backend/tests/sit/report_sit_test.go`

**Step 1: Update setupRouter to include report routes**

In `backend/tests/sit/gitlab_sit_test.go`, add import for report service and add the handler + routes:

1. Add import: `"github.com/cds-id/pdt/backend/internal/services/report"`

2. After the jiraHandler line, add:

```go
	reportGen := report.NewGenerator(db)
	reportHandler := &handlers.ReportHandler{DB: db, Generator: reportGen}
```

3. After the jira routes, add:

```go
	reports := protected.Group("/reports")
	reports.POST("/generate", reportHandler.Generate)
	reports.GET("", reportHandler.List)
	reports.GET("/:id", reportHandler.Get)
	reports.DELETE("/:id", reportHandler.Delete)
	reportTemplates := reports.Group("/templates")
	reportTemplates.GET("", reportHandler.ListTemplates)
	reportTemplates.POST("", reportHandler.CreateTemplate)
	reportTemplates.PUT("/:id", reportHandler.UpdateTemplate)
	reportTemplates.DELETE("/:id", reportHandler.DeleteTemplate)
	reportTemplates.POST("/preview", reportHandler.PreviewTemplate)
```

**Step 2: Create the report SIT test**

Create `backend/tests/sit/report_sit_test.go`:

```go
package sit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestFullReportFlow(t *testing.T) {
	skipIfNoGitLab(t)

	router, db, _, _ := setupRouter(t)

	t.Cleanup(func() {
		db.Exec("DELETE FROM reports WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-report@test.local")
		db.Exec("DELETE FROM report_templates WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-report@test.local")
		db.Exec("DELETE FROM commit_card_links WHERE commit_id IN (SELECT id FROM commits WHERE repo_id IN (SELECT id FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?)))", "sit-report@test.local")
		db.Exec("DELETE FROM commits WHERE repo_id IN (SELECT id FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?))", "sit-report@test.local")
		db.Exec("DELETE FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-report@test.local")
		db.Exec("DELETE FROM users WHERE email = ?", "sit-report@test.local")
	})

	var token string

	// --- Step 1: Register and setup ---
	t.Run("1_Setup", func(t *testing.T) {
		// Register
		body := jsonBody(map[string]string{"email": "sit-report@test.local", "password": "testpass123"})
		w := doRequest(router, "POST", "/api/auth/register", body, "")
		if w.Code != http.StatusCreated {
			t.Fatalf("register failed: %d — %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		token = resp["token"].(string)

		// Configure GitLab
		body = jsonBody(map[string]string{
			"gitlab_token": getEnv("SIT_GITLAB_TOKEN"),
			"gitlab_url":   getEnv("SIT_GITLAB_URL"),
		})
		w = doRequest(router, "PUT", "/api/user/profile", body, token)
		if w.Code != http.StatusOK {
			t.Fatalf("profile update failed: %d", w.Code)
		}

		// Add repo
		body = jsonBody(map[string]string{"url": getEnv("SIT_GITLAB_REPO")})
		w = doRequest(router, "POST", "/api/repos", body, token)
		if w.Code != http.StatusCreated {
			t.Fatalf("add repo failed: %d — %s", w.Code, w.Body.String())
		}

		// Sync commits
		w = doRequest(router, "POST", "/api/sync/commits", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("sync failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Setup complete: registered, configured, synced")
	})

	// --- Step 2: Generate report for today ---
	var reportID float64
	t.Run("2_GenerateReport", func(t *testing.T) {
		today := time.Now().Format("2006-01-02")
		body := jsonBody(map[string]string{"date": today})
		w := doRequest(router, "POST", "/api/reports/generate", body, token)
		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Fatalf("generate report failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		reportID = resp["id"].(float64)
		t.Logf("Generated report id=%v, title=%v", resp["id"], resp["title"])
		content := resp["content"].(string)
		if len(content) > 200 {
			t.Logf("Content preview: %s...", content[:200])
		} else {
			t.Logf("Content: %s", content)
		}
	})

	// --- Step 3: List reports ---
	t.Run("3_ListReports", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/reports", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("list reports failed: %d — %s", w.Code, w.Body.String())
		}

		var reports []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &reports)
		t.Logf("Total reports: %d", len(reports))
	})

	// --- Step 4: Get specific report ---
	t.Run("4_GetReport", func(t *testing.T) {
		url := fmt.Sprintf("/api/reports/%d", int(reportID))
		w := doRequest(router, "GET", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("get report failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Report retrieved successfully")
	})

	// --- Step 5: Create custom template ---
	var templateID float64
	t.Run("5_CreateTemplate", func(t *testing.T) {
		body := jsonBody(map[string]interface{}{
			"name":       "Custom Format",
			"content":    "# {{.DateFormatted}}\n\nTotal: {{.Stats.TotalCommits}} commits on {{.Stats.TotalCards}} cards\n\n{{range .Cards}}\n## {{.Key}}: {{.Summary}}\n{{range .Commits}}\n- {{.Message}}\n{{end}}\n{{end}}",
			"is_default": true,
		})
		w := doRequest(router, "POST", "/api/reports/templates", body, token)
		if w.Code != http.StatusCreated {
			t.Fatalf("create template failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		templateID = resp["id"].(float64)
		t.Logf("Created template id=%v, name=%v, is_default=%v", resp["id"], resp["name"], resp["is_default"])
	})

	// --- Step 6: Preview template ---
	t.Run("6_PreviewTemplate", func(t *testing.T) {
		body := jsonBody(map[string]string{
			"content": "Preview: {{.Stats.TotalCommits}} commits by {{.Author}}",
		})
		w := doRequest(router, "POST", "/api/reports/templates/preview", body, token)
		if w.Code != http.StatusOK {
			t.Fatalf("preview failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		t.Logf("Preview: %v", resp["rendered"])
	})

	// --- Step 7: Generate with custom template ---
	t.Run("7_GenerateWithCustom", func(t *testing.T) {
		today := time.Now().Format("2006-01-02")
		tid := uint(templateID)
		body := jsonBody(map[string]interface{}{
			"date":        today,
			"template_id": tid,
		})
		w := doRequest(router, "POST", "/api/reports/generate", body, token)
		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Fatalf("generate with custom failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		content := resp["content"].(string)
		t.Logf("Custom report content preview: %s", truncate(content, 200))
	})

	// --- Step 8: Delete template ---
	t.Run("8_DeleteTemplate", func(t *testing.T) {
		url := fmt.Sprintf("/api/reports/templates/%d", int(templateID))
		w := doRequest(router, "DELETE", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("delete template failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Template deleted")
	})

	// --- Step 9: Delete report ---
	t.Run("9_DeleteReport", func(t *testing.T) {
		url := fmt.Sprintf("/api/reports/%d", int(reportID))
		w := doRequest(router, "DELETE", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("delete report failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Report deleted")
	})
}
```

**Step 3: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 4: Run the report SIT test**

Run: `cd /home/nst/GolandProjects/pdt/backend && go test -v -run TestFullReportFlow ./tests/sit/ -timeout 120s`
Expected: All 9 sub-tests PASS.

**Step 5: Run all other SIT tests to verify no regressions**

Run: `cd /home/nst/GolandProjects/pdt/backend && go test -v -run TestFullGitHubFlow ./tests/sit/ -timeout 120s`
Run: `cd /home/nst/GolandProjects/pdt/backend && go test -v -run TestFullGitLabFlow ./tests/sit/ -timeout 120s`
Expected: All pass.

**Step 6: Commit**

```bash
git add backend/tests/sit/gitlab_sit_test.go backend/tests/sit/report_sit_test.go
git commit -m "test: add report SIT test with full template + generation flow"
```
