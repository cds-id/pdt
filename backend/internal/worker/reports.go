package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	"gorm.io/gorm"
)

// GenerateMonthlyReportForUser generates a monthly report for a single user for the given month/year.
func GenerateMonthlyReportForUser(db *gorm.DB, enc *crypto.Encryptor, r2 *storage.R2Client, userID uint, month, year int) error {
	generator := report.NewGenerator(db, enc)
	data, err := generator.BuildMonthlyReportData(userID, month, year)
	if err != nil {
		return fmt.Errorf("build monthly data: %w", err)
	}

	if data.TotalCommits == 0 {
		log.Printf("[report-worker] user=%d no activity in %d-%02d, skipping monthly report", userID, year, month)
		return nil
	}

	templateContent := generator.GetMonthlyTemplateContent(userID)
	rendered, err := generator.RenderMonthly(templateContent, data)
	if err != nil {
		return fmt.Errorf("render monthly: %w", err)
	}

	m := month
	y := year
	dateStr := fmt.Sprintf("%04d-%02d", year, month)
	rpt := models.Report{
		UserID:     userID,
		Date:       dateStr,
		Title:      fmt.Sprintf("Monthly Report — %s %d", time.Month(month).String(), year),
		Content:    rendered,
		ReportType: "monthly",
		Month:      &m,
		Year:       &y,
	}

	var existing models.Report
	if db.Where("user_id = ? AND report_type = ? AND month = ? AND year = ?", userID, "monthly", month, year).First(&existing).Error == nil {
		existing.Content = rendered
		existing.Title = rpt.Title
		db.Save(&existing)
	} else {
		db.Create(&rpt)
	}

	log.Printf("[report-worker] user=%d monthly report generated for %d-%02d", userID, year, month)
	return nil
}

// AutoGenerateReports generates daily reports for all users who don't have one for today.
func AutoGenerateReports(db *gorm.DB, enc *crypto.Encryptor, r2 *storage.R2Client) {
	today := time.Now().Format("2006-01-02")

	var users []models.User
	db.Find(&users)

	gen := report.NewGenerator(db, enc)

	for _, user := range users {
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
			continue
		}

		templateContent, templateID := gen.GetTemplateContent(user.ID, nil)

		rendered, err := gen.Render(templateContent, data)
		if err != nil {
			log.Printf("[worker] report render failed for user %d: %v", user.ID, err)
			continue
		}

		var fileURL string
		if r2 != nil {
			key := fmt.Sprintf("reports/%d/%s.md", user.ID, today)
			url, err := r2.Upload(context.Background(), key, []byte(rendered), "text/markdown; charset=utf-8")
			if err != nil {
				log.Printf("[worker] R2 upload failed for user %d: %v", user.ID, err)
			} else {
				fileURL = url
			}
		}

		rpt := models.Report{
			UserID:     user.ID,
			TemplateID: templateID,
			Date:       today,
			Title:      "Daily Report — " + date.Format("Monday, 02 January 2006"),
			Content:    rendered,
			FileURL:    fileURL,
		}
		db.Create(&rpt)

		log.Printf("[worker] report generated for user %d: %d commits, %d cards, url=%s",
			user.ID, data.Stats.TotalCommits, data.Stats.TotalCards, fileURL)
	}
}
