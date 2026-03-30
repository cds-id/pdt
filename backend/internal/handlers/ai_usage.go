package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
)

type AIUsageHandler struct {
	DB *gorm.DB
}

// GetUsageSummary returns AI usage statistics for the current user.
func (h *AIUsageHandler) GetUsageSummary(c *gin.Context) {
	userID := c.GetUint("user_id")

	now := time.Now()
	todayStart := now.Truncate(24 * time.Hour)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Today's usage by provider
	type providerSummary struct {
		Provider         string `json:"provider"`
		Model            string `json:"model"`
		Feature          string `json:"feature"`
		Calls            int64  `json:"calls"`
		PromptTokens     int64  `json:"prompt_tokens"`
		CompletionTokens int64  `json:"completion_tokens"`
	}

	var todayByProvider []providerSummary
	h.DB.Model(&models.AIUsage{}).
		Select("provider, model, feature, COUNT(*) as calls, COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, COALESCE(SUM(completion_tokens), 0) as completion_tokens").
		Where("user_id = ? AND created_at >= ?", userID, todayStart).
		Group("provider, model, feature").
		Scan(&todayByProvider)

	// Monthly usage by provider
	var monthByProvider []providerSummary
	h.DB.Model(&models.AIUsage{}).
		Select("provider, model, feature, COUNT(*) as calls, COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, COALESCE(SUM(completion_tokens), 0) as completion_tokens").
		Where("user_id = ? AND created_at >= ?", userID, monthStart).
		Group("provider, model, feature").
		Scan(&monthByProvider)

	// Daily usage for the last 30 days (for chart)
	type dailyUsage struct {
		Date             string `json:"date"`
		Provider         string `json:"provider"`
		Calls            int64  `json:"calls"`
		PromptTokens     int64  `json:"prompt_tokens"`
		CompletionTokens int64  `json:"completion_tokens"`
	}

	thirtyDaysAgo := now.AddDate(0, 0, -30)
	var dailyData []dailyUsage
	h.DB.Model(&models.AIUsage{}).
		Select("DATE(created_at) as date, provider, COUNT(*) as calls, COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, COALESCE(SUM(completion_tokens), 0) as completion_tokens").
		Where("user_id = ? AND created_at >= ?", userID, thirtyDaysAgo).
		Group("DATE(created_at), provider").
		Order("date").
		Scan(&dailyData)

	// Totals
	var totalCalls int64
	var totalPrompt, totalCompletion int64
	h.DB.Model(&models.AIUsage{}).
		Where("user_id = ?", userID).
		Count(&totalCalls)
	h.DB.Model(&models.AIUsage{}).
		Select("COALESCE(SUM(prompt_tokens), 0) as prompt, COALESCE(SUM(completion_tokens), 0) as completion").
		Where("user_id = ?", userID).
		Row().Scan(&totalPrompt, &totalCompletion)

	// Vision rate limit status
	var visionTodayCount int64
	h.DB.Model(&models.AIUsage{}).
		Where("provider = ? AND feature = ? AND created_at >= ?", "mistral", "wa_vision", todayStart).
		Count(&visionTodayCount)

	c.JSON(http.StatusOK, gin.H{
		"today":    todayByProvider,
		"month":    monthByProvider,
		"daily":    dailyData,
		"totals": gin.H{
			"calls":             totalCalls,
			"prompt_tokens":     totalPrompt,
			"completion_tokens": totalCompletion,
			"total_tokens":      totalPrompt + totalCompletion,
		},
		"rate_limits": gin.H{
			"vision": gin.H{
				"used":  visionTodayCount,
				"limit": 20,
			},
		},
	})
}
