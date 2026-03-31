package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/scheduler"
)

type ScheduleHandler struct {
	DB     *gorm.DB
	Engine *scheduler.Engine
}

// List GET /api/schedules
func (h *ScheduleHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")

	var schedules []models.AgentSchedule
	if err := h.DB.Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedules)
}

type createScheduleRequest struct {
	Name            string          `json:"name" binding:"required"`
	AgentName       string          `json:"agent_name"`
	Prompt          string          `json:"prompt" binding:"required"`
	TriggerType     string          `json:"trigger_type" binding:"required"`
	CronExpr        string          `json:"cron_expr"`
	IntervalSeconds int             `json:"interval_seconds"`
	EventName       string          `json:"event_name"`
	ChainConfig     json.RawMessage `json:"chain_config"`
	Enabled         *bool           `json:"enabled"`
}

// Create POST /api/schedules
func (h *ScheduleHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req createScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	schedule := models.AgentSchedule{
		UserID:          userID,
		Name:            req.Name,
		AgentName:       req.AgentName,
		Prompt:          req.Prompt,
		TriggerType:     req.TriggerType,
		CronExpr:        req.CronExpr,
		IntervalSeconds: req.IntervalSeconds,
		EventName:       req.EventName,
		ChainConfig:     req.ChainConfig,
		Enabled:         enabled,
	}

	nextRun, err := scheduler.NextRunAt(req.TriggerType, req.CronExpr, req.IntervalSeconds, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trigger configuration: " + err.Error()})
		return
	}
	schedule.NextRunAt = nextRun

	if err := h.DB.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.Engine.RefreshEventSubscriptions()

	c.JSON(http.StatusCreated, schedule)
}

type updateScheduleRequest struct {
	Name            *string         `json:"name"`
	AgentName       *string         `json:"agent_name"`
	Prompt          *string         `json:"prompt"`
	TriggerType     *string         `json:"trigger_type"`
	CronExpr        *string         `json:"cron_expr"`
	IntervalSeconds *int            `json:"interval_seconds"`
	EventName       *string         `json:"event_name"`
	ChainConfig     json.RawMessage `json:"chain_config"`
	Enabled         *bool           `json:"enabled"`
}

// Update PUT /api/schedules/:id
func (h *ScheduleHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var schedule models.AgentSchedule
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	var req updateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		schedule.Name = *req.Name
	}
	if req.AgentName != nil {
		schedule.AgentName = *req.AgentName
	}
	if req.Prompt != nil {
		schedule.Prompt = *req.Prompt
	}
	if req.TriggerType != nil {
		schedule.TriggerType = *req.TriggerType
	}
	if req.CronExpr != nil {
		schedule.CronExpr = *req.CronExpr
	}
	if req.IntervalSeconds != nil {
		schedule.IntervalSeconds = *req.IntervalSeconds
	}
	if req.EventName != nil {
		schedule.EventName = *req.EventName
	}
	if req.ChainConfig != nil {
		schedule.ChainConfig = req.ChainConfig
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}

	nextRun, err := scheduler.NextRunAt(schedule.TriggerType, schedule.CronExpr, schedule.IntervalSeconds, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trigger configuration: " + err.Error()})
		return
	}
	schedule.NextRunAt = nextRun

	if err := h.DB.Save(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.Engine.RefreshEventSubscriptions()

	c.JSON(http.StatusOK, schedule)
}

// Delete DELETE /api/schedules/:id
func (h *ScheduleHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var schedule models.AgentSchedule
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Delete in order: steps → runs → schedule (FK constraints)
	var runIDs []string
	h.DB.Model(&models.AgentScheduleRun{}).Where("schedule_id = ?", id).Pluck("id", &runIDs)
	if len(runIDs) > 0 {
		h.DB.Where("run_id IN ?", runIDs).Delete(&models.AgentScheduleRunStep{})
	}
	h.DB.Where("schedule_id = ?", id).Delete(&models.AgentScheduleRun{})
	h.DB.Delete(&schedule)

	h.Engine.RefreshEventSubscriptions()

	c.JSON(http.StatusOK, gin.H{"message": "schedule deleted"})
}

// Toggle POST /api/schedules/:id/toggle
func (h *ScheduleHandler) Toggle(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var schedule models.AgentSchedule
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	schedule.Enabled = !schedule.Enabled

	if schedule.Enabled {
		nextRun, err := scheduler.NextRunAt(schedule.TriggerType, schedule.CronExpr, schedule.IntervalSeconds, time.Now())
		if err == nil {
			schedule.NextRunAt = nextRun
		}
	}

	if err := h.DB.Save(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.Engine.RefreshEventSubscriptions()

	c.JSON(http.StatusOK, schedule)
}

// RunNow POST /api/schedules/:id/run
func (h *ScheduleHandler) RunNow(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var schedule models.AgentSchedule
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	go h.Engine.RunScheduleNow(schedule)

	c.JSON(http.StatusAccepted, gin.H{"message": "schedule run triggered"})
}

// ListRuns GET /api/schedules/:id/runs
func (h *ScheduleHandler) ListRuns(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var schedule models.AgentSchedule
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	var runs []models.AgentScheduleRun
	if err := h.DB.Where("schedule_id = ?", id).
		Order("created_at desc").
		Limit(50).
		Find(&runs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, runs)
}

// GetRun GET /api/schedules/runs/:runId
func (h *ScheduleHandler) GetRun(c *gin.Context) {
	userID := c.GetUint("user_id")
	runID := c.Param("runId")

	var run models.AgentScheduleRun
	if err := h.DB.Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	var steps []models.AgentScheduleRunStep
	h.DB.Where("run_id = ?", runID).Order("created_at asc").Find(&steps)

	c.JSON(http.StatusOK, gin.H{
		"run":   run,
		"steps": steps,
	})
}
