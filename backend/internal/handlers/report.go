package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReportHandler struct {
	DB        *gorm.DB
	Generator *report.Generator
	R2        *storage.R2Client // nil if R2 not configured
}

// uploadToR2 uploads the report markdown to R2 and returns the URL.
// Returns empty string if R2 is not configured (non-fatal).
func (h *ReportHandler) uploadToR2(userID uint, date string, content string) string {
	if h.R2 == nil {
		return ""
	}

	key := fmt.Sprintf("reports/%d/%s.md", userID, date)
	url, err := h.R2.Upload(context.Background(), key, []byte(content), "text/markdown; charset=utf-8")
	if err != nil {
		log.Printf("[report] R2 upload failed: %v", err)
		return ""
	}
	return url
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

	fileURL := h.uploadToR2(userID, req.Date, rendered)

	rpt := models.Report{
		UserID:     userID,
		TemplateID: templateID,
		Date:       req.Date,
		Title:      "Daily Report — " + date.Format("Monday, 02 January 2006"),
		Content:    rendered,
		FileURL:    fileURL,
	}

	// Upsert: if report for this date already exists, update it
	var existing models.Report
	if err := h.DB.Where("user_id = ? AND date = ?", userID, req.Date).First(&existing).Error; err == nil {
		existing.Content = rendered
		existing.Title = rpt.Title
		existing.TemplateID = templateID
		existing.FileURL = fileURL
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
