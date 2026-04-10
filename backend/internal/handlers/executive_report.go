package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/executive"
)

// CorrelatorBuilder is the narrow interface the handler depends on.
// The production wiring passes *executive.Correlator; tests pass a fake.
type CorrelatorBuilder interface {
	Build(ctx context.Context, userID uint, workspaceID *uint, r executive.DateRange, staleDays int) (*executive.CorrelatedDataset, error)
}

type ExecutiveReportHandler struct {
	DB         *gorm.DB
	Correlator CorrelatorBuilder
	Agent      *agent.ExecutiveReportAgent
}

type generateRequest struct {
	RangeStart         time.Time `json:"range_start" binding:"required"`
	RangeEnd           time.Time `json:"range_end" binding:"required"`
	StaleThresholdDays int       `json:"stale_threshold_days"`
	WorkspaceID        *uint     `json:"workspace_id"`
}

func (h *ExecutiveReportHandler) Generate(c *gin.Context) {
	var req generateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.RangeEnd.Before(req.RangeStart) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "range_end before range_start"})
		return
	}
	if req.RangeEnd.Sub(req.RangeStart) > time.Duration(executive.MaxRangeDays)*24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("range exceeds max %d days", executive.MaxRangeDays)})
		return
	}
	if req.StaleThresholdDays == 0 {
		req.StaleThresholdDays = executive.StaleThresholdDefault
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	// SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	writeEvent := func(event string, payload any) bool {
		b, err := json.Marshal(payload)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(b)); err != nil {
			return false
		}
		c.Writer.Flush()
		return true
	}

	// Pre-insert a "generating" row so interruptions remain visible in history.
	row := models.ExecutiveReport{
		UserID:             userID,
		WorkspaceID:        req.WorkspaceID,
		RangeStart:         req.RangeStart,
		RangeEnd:           req.RangeEnd,
		StaleThresholdDays: req.StaleThresholdDays,
		Status:             "generating",
	}
	if err := h.DB.Create(&row).Error; err != nil {
		writeEvent("error", gin.H{"message": err.Error()})
		return
	}

	writeEvent("status", gin.H{"phase": "correlating"})
	ds, err := h.Correlator.Build(c.Request.Context(), userID, req.WorkspaceID,
		executive.DateRange{Start: req.RangeStart, End: req.RangeEnd}, req.StaleThresholdDays)
	if err != nil {
		h.markFailed(row.ID, err.Error())
		writeEvent("error", gin.H{"message": err.Error()})
		return
	}

	writeEvent("dataset", ds)
	writeEvent("status", gin.H{"phase": "thinking"})

	events := make(chan agent.ExecutiveEvent, 64)
	go h.Agent.Run(c.Request.Context(), ds, events)

	var narrative string
	var suggestions []executive.Suggestion
	var streamErr error

	for ev := range events {
		switch ev.Kind {
		case "delta":
			narrative += ev.Delta
			writeEvent("delta", gin.H{"text": ev.Delta})
		case "suggestion":
			if ev.Suggestion != nil {
				suggestions = append(suggestions, *ev.Suggestion)
				writeEvent("suggestion", ev.Suggestion)
			}
		case "error":
			if ev.Err != nil {
				streamErr = ev.Err
			}
		case "done":
			// loop exits when channel closes
		}
	}

	if streamErr != nil {
		h.markFailed(row.ID, streamErr.Error())
		writeEvent("error", gin.H{"message": streamErr.Error()})
		return
	}

	writeEvent("status", gin.H{"phase": "persisting"})

	dsBytes, _ := json.Marshal(ds)
	sugBytes, _ := json.Marshal(suggestions)
	now := time.Now()
	if err := h.DB.Model(&row).Updates(map[string]any{
		"narrative":    narrative,
		"suggestions":  datatypes.JSON(sugBytes),
		"dataset":      datatypes.JSON(dsBytes),
		"status":       "completed",
		"completed_at": &now,
	}).Error; err != nil {
		writeEvent("error", gin.H{"message": err.Error()})
		return
	}

	writeEvent("done", gin.H{"id": row.ID})
}

func (h *ExecutiveReportHandler) markFailed(id uint, msg string) {
	h.DB.Model(&models.ExecutiveReport{}).Where("id = ?", id).Updates(map[string]any{
		"status":        "failed",
		"error_message": msg,
	})
}

type executiveReportListItem struct {
	ID          uint       `json:"id"`
	RangeStart  time.Time  `json:"range_start"`
	RangeEnd    time.Time  `json:"range_end"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func (h *ExecutiveReportHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	var rows []models.ExecutiveReport
	if err := h.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(100).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]executiveReportListItem, len(rows))
	for i, r := range rows {
		out[i] = executiveReportListItem{
			ID: r.ID, RangeStart: r.RangeStart, RangeEnd: r.RangeEnd,
			Status: r.Status, CreatedAt: r.CreatedAt, CompletedAt: r.CompletedAt,
		}
	}
	c.JSON(http.StatusOK, out)
}

func (h *ExecutiveReportHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var row models.ExecutiveReport
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *ExecutiveReportHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	res := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.ExecutiveReport{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
