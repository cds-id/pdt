package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	"gorm.io/gorm"
)

type ReportAgent struct {
	DB        *gorm.DB
	UserID    uint
	Generator *report.Generator
	R2        *storage.R2Client
}

func (a *ReportAgent) Name() string { return "report" }

func (a *ReportAgent) SystemPrompt() string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`You are a Report assistant for PDT. Today is %s. You help users generate daily and monthly reports, view existing reports, and manage report templates. Use the available tools to fetch and generate reports. When generating reports, confirm the date/month with the user first.`, today)
}

func (a *ReportAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Name:        "generate_daily_report",
			Description: "Generate a daily development report for a specific date",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"date": {"type": "string", "description": "Date in YYYY-MM-DD format (default: today)"}
				}
			}`),
		},
		{
			Name:        "generate_monthly_report",
			Description: "Generate a monthly report with aggregated stats and AI narrative",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"month": {"type": "integer", "description": "Month number (1-12)"},
					"year": {"type": "integer", "description": "Year (e.g., 2026)"}
				},
				"required": ["month", "year"]
			}`),
		},
		{
			Name:        "list_reports",
			Description: "List existing reports, optionally filtered by type",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"report_type": {"type": "string", "enum": ["daily", "monthly"], "description": "Filter by report type"},
					"limit": {"type": "integer", "description": "Max results (default 20)"}
				}
			}`),
		},
		{
			Name:        "get_report",
			Description: "Get the full content of a specific report by ID",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {"type": "integer", "description": "Report ID"}
				},
				"required": ["id"]
			}`),
		},
		{
			Name:        "preview_template",
			Description: "Preview a report template rendered with today's data",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"template_id": {"type": "integer", "description": "Template ID to preview"}
				},
				"required": ["template_id"]
			}`),
		},
	}
}

func (a *ReportAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "generate_daily_report":
		return a.generateDaily(args)
	case "generate_monthly_report":
		return a.generateMonthly(args)
	case "list_reports":
		return a.listReports(args)
	case "get_report":
		return a.getReport(args)
	case "preview_template":
		return a.previewTemplate(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (a *ReportAgent) generateDaily(args json.RawMessage) (any, error) {
	var params struct {
		Date string `json:"date"`
	}
	json.Unmarshal(args, &params)
	if params.Date == "" {
		params.Date = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", params.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %s", params.Date)
	}

	data, err := a.Generator.BuildReportData(a.UserID, date)
	if err != nil {
		return nil, fmt.Errorf("build report data: %w", err)
	}

	templateContent, templateID := a.Generator.GetTemplateContent(a.UserID, nil)
	rendered, err := a.Generator.Render(templateContent, data)
	if err != nil {
		return nil, fmt.Errorf("render report: %w", err)
	}

	// Upsert report
	var existing models.Report
	rpt := models.Report{
		UserID:     a.UserID,
		TemplateID: templateID,
		Date:       params.Date,
		Title:      fmt.Sprintf("Daily Report — %s", data.DateFormatted),
		Content:    rendered,
		ReportType: "daily",
	}

	if a.DB.Where("user_id = ? AND date = ? AND report_type = ?", a.UserID, params.Date, "daily").First(&existing).Error == nil {
		existing.Content = rendered
		existing.Title = rpt.Title
		a.DB.Save(&existing)
		rpt.ID = existing.ID
	} else {
		a.DB.Create(&rpt)
	}

	return map[string]any{
		"id":    rpt.ID,
		"title": rpt.Title,
		"date":  params.Date,
		"content": rendered,
		"stats": map[string]any{
			"total_commits": data.Stats.TotalCommits,
			"total_cards":   data.Stats.TotalCards,
		},
	}, nil
}

func (a *ReportAgent) generateMonthly(args json.RawMessage) (any, error) {
	var params struct {
		Month int `json:"month"`
		Year  int `json:"year"`
	}
	json.Unmarshal(args, &params)

	data, err := a.Generator.BuildMonthlyReportData(a.UserID, params.Month, params.Year)
	if err != nil {
		return nil, fmt.Errorf("build monthly report data: %w", err)
	}

	templateContent := a.Generator.GetMonthlyTemplateContent(a.UserID)
	rendered, err := a.Generator.RenderMonthly(templateContent, data)
	if err != nil {
		return nil, fmt.Errorf("render monthly report: %w", err)
	}

	month := params.Month
	year := params.Year
	dateStr := fmt.Sprintf("%04d-%02d", year, month)
	rpt := models.Report{
		UserID:     a.UserID,
		Date:       dateStr,
		Title:      fmt.Sprintf("Monthly Report — %s %d", time.Month(month).String(), year),
		Content:    rendered,
		ReportType: "monthly",
		Month:      &month,
		Year:       &year,
	}

	var existing models.Report
	if a.DB.Where("user_id = ? AND report_type = ? AND month = ? AND year = ?", a.UserID, "monthly", month, year).First(&existing).Error == nil {
		existing.Content = rendered
		existing.Title = rpt.Title
		a.DB.Save(&existing)
		rpt.ID = existing.ID
	} else {
		a.DB.Create(&rpt)
	}

	return map[string]any{
		"id":      rpt.ID,
		"title":   rpt.Title,
		"month":   month,
		"year":    year,
		"content": rendered,
	}, nil
}

func (a *ReportAgent) listReports(args json.RawMessage) (any, error) {
	var params struct {
		ReportType string `json:"report_type"`
		Limit      int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 20
	}

	query := a.DB.Where("user_id = ?", a.UserID)
	if params.ReportType != "" {
		query = query.Where("report_type = ?", params.ReportType)
	}

	var reports []models.Report
	query.Order("created_at desc").Limit(params.Limit).Find(&reports)

	type result struct {
		ID         uint   `json:"id"`
		Title      string `json:"title"`
		Date       string `json:"date"`
		ReportType string `json:"report_type"`
	}
	var results []result
	for _, r := range reports {
		results = append(results, result{
			ID:         r.ID,
			Title:      r.Title,
			Date:       r.Date,
			ReportType: r.ReportType,
		})
	}
	return results, nil
}

func (a *ReportAgent) getReport(args json.RawMessage) (any, error) {
	var params struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(args, &params)

	var rpt models.Report
	if err := a.DB.Where("id = ? AND user_id = ?", params.ID, a.UserID).First(&rpt).Error; err != nil {
		return nil, fmt.Errorf("report not found: %d", params.ID)
	}
	return map[string]any{
		"id":          rpt.ID,
		"title":       rpt.Title,
		"date":        rpt.Date,
		"report_type": rpt.ReportType,
		"content":     rpt.Content,
	}, nil
}

func (a *ReportAgent) previewTemplate(args json.RawMessage) (any, error) {
	var params struct {
		TemplateID uint `json:"template_id"`
	}
	json.Unmarshal(args, &params)

	var tmpl models.ReportTemplate
	if err := a.DB.Where("id = ? AND user_id = ?", params.TemplateID, a.UserID).First(&tmpl).Error; err != nil {
		return nil, fmt.Errorf("template not found: %d", params.TemplateID)
	}

	data, err := a.Generator.BuildReportData(a.UserID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("build preview data: %w", err)
	}

	rendered, err := a.Generator.Render(tmpl.Content, data)
	if err != nil {
		return nil, fmt.Errorf("render preview: %w", err)
	}

	return map[string]any{
		"template_name": tmpl.Name,
		"preview":       rendered,
	}, nil
}
