# Agent Scheduling REST API & Integration Implementation Plan (Plan 2 of 3)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the scheduler engine into the application with REST API endpoints, Telegram notifications on run completion, and event emission from existing worker sync loops.

**Architecture:** New `ScheduleHandler` following existing Gin handler patterns. Engine and event bus initialized in `main.go` alongside existing worker scheduler. Worker sync loops emit events after completion. Executor sends Telegram notifications via the bot API.

**Tech Stack:** Go, Gin, GORM, existing `scheduler`, `eventbus`, `telegram` packages

**Spec:** `docs/superpowers/specs/2026-03-31-agent-scheduling-design.md`

**Depends on:** Plan 1 (scheduler engine) — already implemented.

---

## File Structure

```
backend/
├── internal/
│   ├── handlers/
│   │   └── schedule.go          # REST API handler (CRUD + toggle + run + history)
│   ├── scheduler/
│   │   └── notifier.go          # Telegram notification on run completion
│   └── worker/
│       └── scheduler.go         # Modify: add EventBus field, emit events after sync
├── cmd/server/
│   └── main.go                  # Modify: wire engine, event bus, schedule handler
```

---

### Task 1: Schedule REST API handler

**Files:**
- Create: `backend/internal/handlers/schedule.go`

- [ ] **Step 1: Create the handler struct and CRUD methods**

```go
// backend/internal/handlers/schedule.go
package handlers

import (
	"net/http"
	"time"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/scheduler"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ScheduleHandler struct {
	DB     *gorm.DB
	Engine *scheduler.Engine
}

func (h *ScheduleHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	var schedules []models.AgentSchedule
	h.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&schedules)
	c.JSON(http.StatusOK, schedules)
}

func (h *ScheduleHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Name            string          `json:"name" binding:"required"`
		AgentName       string          `json:"agent_name"`
		Prompt          string          `json:"prompt" binding:"required"`
		TriggerType     string          `json:"trigger_type" binding:"required,oneof=cron interval event"`
		CronExpr        string          `json:"cron_expr"`
		IntervalSeconds int             `json:"interval_seconds"`
		EventName       string          `json:"event_name"`
		ChainConfig     json.RawMessage `json:"chain_config"`
		Enabled         *bool           `json:"enabled"`
	}
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

	// Compute initial next_run_at
	if schedule.TriggerType != "event" {
		nextRun, err := scheduler.NextRunAt(schedule.TriggerType, schedule.CronExpr, schedule.IntervalSeconds, time.Now())
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule: " + err.Error()})
			return
		}
		schedule.NextRunAt = nextRun
	}

	if err := h.DB.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create schedule"})
		return
	}

	if h.Engine != nil {
		h.Engine.RefreshEventSubscriptions()
	}

	c.JSON(http.StatusCreated, schedule)
}

func (h *ScheduleHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var schedule models.AgentSchedule
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	var req struct {
		Name            *string          `json:"name"`
		AgentName       *string          `json:"agent_name"`
		Prompt          *string          `json:"prompt"`
		TriggerType     *string          `json:"trigger_type"`
		CronExpr        *string          `json:"cron_expr"`
		IntervalSeconds *int             `json:"interval_seconds"`
		EventName       *string          `json:"event_name"`
		ChainConfig     *json.RawMessage `json:"chain_config"`
		Enabled         *bool            `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.AgentName != nil {
		updates["agent_name"] = *req.AgentName
	}
	if req.Prompt != nil {
		updates["prompt"] = *req.Prompt
	}
	if req.TriggerType != nil {
		updates["trigger_type"] = *req.TriggerType
	}
	if req.CronExpr != nil {
		updates["cron_expr"] = *req.CronExpr
	}
	if req.IntervalSeconds != nil {
		updates["interval_seconds"] = *req.IntervalSeconds
	}
	if req.EventName != nil {
		updates["event_name"] = *req.EventName
	}
	if req.ChainConfig != nil {
		updates["chain_config"] = *req.ChainConfig
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	h.DB.Model(&schedule).Updates(updates)

	// Recompute next_run_at
	triggerType := schedule.TriggerType
	if req.TriggerType != nil {
		triggerType = *req.TriggerType
	}
	cronExpr := schedule.CronExpr
	if req.CronExpr != nil {
		cronExpr = *req.CronExpr
	}
	intervalSecs := schedule.IntervalSeconds
	if req.IntervalSeconds != nil {
		intervalSecs = *req.IntervalSeconds
	}

	if triggerType != "event" {
		nextRun, _ := scheduler.NextRunAt(triggerType, cronExpr, intervalSecs, time.Now())
		h.DB.Model(&schedule).Update("next_run_at", nextRun)
	}

	if h.Engine != nil {
		h.Engine.RefreshEventSubscriptions()
	}

	h.DB.Where("id = ?", id).First(&schedule)
	c.JSON(http.StatusOK, schedule)
}

func (h *ScheduleHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	result := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.AgentSchedule{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Clean up runs
	h.DB.Where("schedule_id = ?", id).Delete(&models.AgentScheduleRun{})

	if h.Engine != nil {
		h.Engine.RefreshEventSubscriptions()
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *ScheduleHandler) Toggle(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var schedule models.AgentSchedule
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	newEnabled := !schedule.Enabled
	h.DB.Model(&schedule).Update("enabled", newEnabled)

	if h.Engine != nil {
		h.Engine.RefreshEventSubscriptions()
	}

	c.JSON(http.StatusOK, gin.H{"enabled": newEnabled})
}

func (h *ScheduleHandler) RunNow(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var schedule models.AgentSchedule
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if h.Engine != nil {
		go h.Engine.RunScheduleNow(schedule)
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "run triggered"})
}

func (h *ScheduleHandler) ListRuns(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// Verify ownership
	var schedule models.AgentSchedule
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	var runs []models.AgentScheduleRun
	h.DB.Where("schedule_id = ?", id).Order("created_at desc").Limit(50).Find(&runs)
	c.JSON(http.StatusOK, runs)
}

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
```

Note: This file needs `"encoding/json"` in imports for `json.RawMessage`.

- [ ] **Step 2: Add RunScheduleNow to Engine**

In `backend/internal/scheduler/engine.go`, add this method:

```go
// RunScheduleNow triggers a manual run of a schedule.
func (e *Engine) RunScheduleNow(schedule models.AgentSchedule) {
	e.pool.Submit(func() {
		e.executeSchedule(context.Background(), schedule, "manual")
	})
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd backend && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handlers/schedule.go backend/internal/scheduler/engine.go
git commit -m "feat(scheduler): add REST API handler for schedule CRUD and run management"
```

---

### Task 2: Telegram notification on run completion

**Files:**
- Create: `backend/internal/scheduler/notifier.go`

- [ ] **Step 1: Implement the notifier**

```go
// backend/internal/scheduler/notifier.go
package scheduler

import (
	"fmt"
	"log"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/telegram/formatter"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

// Notifier sends Telegram notifications for scheduled run completions.
type Notifier struct {
	DB  *gorm.DB
	Bot *tgbotapi.BotAPI
}

// NotifyRunCompleted sends a Telegram message summarizing a completed run.
func (n *Notifier) NotifyRunCompleted(run *models.AgentScheduleRun, scheduleName string) {
	if n.Bot == nil {
		return
	}

	chatID := n.findTelegramChat(run.UserID)
	if chatID == 0 {
		return
	}

	var text string
	if run.Status == "completed" {
		summary := run.ResultSummary
		if len(summary) > 500 {
			summary = summary[:500] + "..."
		}
		text = fmt.Sprintf("📋 **Scheduled: %s**\n✅ Completed\n\n%s", scheduleName, summary)
	} else {
		errMsg := run.Error
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "..."
		}
		text = fmt.Sprintf("📋 **Scheduled: %s**\n❌ Failed: %s", scheduleName, errMsg)
	}

	htmlContent := formatter.ToTelegramHTML(text)
	msg := tgbotapi.NewMessage(chatID, htmlContent)
	msg.ParseMode = "HTML"
	if _, err := n.Bot.Send(msg); err != nil {
		// Fallback to plain text
		msg.ParseMode = ""
		msg.Text = text
		n.Bot.Send(msg)
	}
}

func (n *Notifier) findTelegramChat(userID uint) int64 {
	var whitelist models.TelegramWhitelist
	if err := n.DB.Where("user_id = ?", userID).First(&whitelist).Error; err != nil {
		return 0
	}

	// Find the most recent chat from this user
	var conv models.Conversation
	if err := n.DB.Where("user_id = ? AND telegram_chat_id != 0", userID).
		Order("updated_at desc").First(&conv).Error; err != nil {
		// Use the telegram user ID as chat ID (works for private chats)
		return whitelist.TelegramUserID
	}
	return conv.TelegramChatID
}
```

- [ ] **Step 2: Integrate notifier into executor**

Add a `Notifier` field to `Executor` in `backend/internal/scheduler/executor.go`:

```go
type Executor struct {
	DB       *gorm.DB
	Client   *minimax.Client
	Agents   map[string]agent.Agent
	Notifier *Notifier
}
```

At the end of the `Run` method, after updating the run status to "completed", add:

```go
	// Send Telegram notification
	if e.Notifier != nil {
		e.Notifier.NotifyRunCompleted(&run, schedule.Name)
	}
```

Also add notification for the failure path in `failRun`:

```go
func (e *Executor) failRun(run *models.AgentScheduleRun, scheduleName string, err error) {
	now := time.Now()
	e.DB.Model(run).Updates(map[string]any{
		"status":       "failed",
		"completed_at": &now,
		"error":        err.Error(),
	})
	run.Status = "failed"
	run.Error = err.Error()
	if e.Notifier != nil {
		e.Notifier.NotifyRunCompleted(run, scheduleName)
	}
}
```

Update all `failRun` call sites to pass `schedule.Name`:
- `e.failRun(&run, schedule.Name, fmt.Errorf("create conversation: %w", err))`
- `e.failRun(&run, schedule.Name, err)`

- [ ] **Step 3: Update engine to pass notifier to executor**

In `backend/internal/scheduler/engine.go`, add a `Notifier` field:

```go
type Engine struct {
	db       *gorm.DB
	client   *minimax.Client
	agents   map[string]agent.Agent
	pool     *Pool
	bus      *eventbus.Bus
	notifier *Notifier
	unsubs   []func()
	mu       sync.Mutex
}
```

Update `NewEngine` to accept a notifier:

```go
func NewEngine(db *gorm.DB, client *minimax.Client, bus *eventbus.Bus, notifier *Notifier, agents ...agent.Agent) *Engine {
```

Set it in the constructor: `notifier: notifier,`

Update `executeSchedule` to pass it to executor:

```go
executor := &Executor{
	DB:       e.db,
	Client:   e.client,
	Agents:   e.agents,
	Notifier: e.notifier,
}
```

- [ ] **Step 4: Verify compilation**

```bash
cd backend && go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/scheduler/notifier.go backend/internal/scheduler/executor.go backend/internal/scheduler/engine.go
git commit -m "feat(scheduler): add Telegram notification on run completion"
```

---

### Task 3: Worker event emission

**Files:**
- Modify: `backend/internal/worker/scheduler.go`

- [ ] **Step 1: Add EventBus field to worker Scheduler**

Add the event bus import and field:

```go
import (
	// ... existing imports ...
	"github.com/cds-id/pdt/backend/internal/scheduler/eventbus"
)

type Scheduler struct {
	// ... existing fields ...
	EventBus *eventbus.Bus
}
```

- [ ] **Step 2: Emit commit_synced event after sync**

In `runCommitSync()`, after the for loop over userIDs completes (before the final log line), add:

```go
	// Emit events for each synced user
	if s.EventBus != nil {
		for _, uid := range userIDs {
			s.EventBus.Publish("commit_synced", map[string]any{
				"user_id": uid,
			})
		}
	}
```

- [ ] **Step 3: Emit jira_synced event after sync**

In `runJiraSync()` (the equivalent jira sync function), after the sync loop, add:

```go
	if s.EventBus != nil {
		for _, uid := range userIDs {
			s.EventBus.Publish("jira_synced", map[string]any{
				"user_id": uid,
			})
		}
	}
```

- [ ] **Step 4: Verify compilation**

```bash
cd backend && go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/worker/scheduler.go
git commit -m "feat(scheduler): emit commit_synced and jira_synced events from worker"
```

---

### Task 4: Wire everything in main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Import scheduler and eventbus packages**

Add to imports:

```go
agentScheduler "github.com/cds-id/pdt/backend/internal/scheduler"
"github.com/cds-id/pdt/backend/internal/scheduler/eventbus"
```

- [ ] **Step 2: Create event bus and pass to worker scheduler**

After the worker scheduler creation block (`if cfg.SyncEnabled`), add the event bus creation BEFORE the worker scheduler, and pass it in:

```go
// Event bus (shared between worker and agent scheduler)
eventBus := eventbus.New()

// Worker scheduler
var syncStatus *worker.SyncStatus
if cfg.SyncEnabled {
	workerScheduler := worker.NewScheduler(db, encryptor, cfg.SyncIntervalCommits, cfg.SyncIntervalJira, cfg.ReportAutoGenerate, cfg.ReportAutoTime, cfg.ReportMonthlyAutoTime, r2Client, weaviateClient)
	workerScheduler.EventBus = eventBus
	workerScheduler.Start(ctx)
	syncStatus = workerScheduler.Status
} else {
	syncStatus = worker.NewSyncStatus()
}
```

- [ ] **Step 3: Create agent scheduler engine after MiniMax client and Telegram bot**

After the Telegram bot initialization block, add:

```go
// Agent scheduler engine
var scheduleEngine *agentScheduler.Engine
if miniMaxClient != nil {
	var notifier *agentScheduler.Notifier
	if tgBot != nil {
		notifier = &agentScheduler.Notifier{DB: db, Bot: tgBot.API()}
	}
	scheduleEngine = agentScheduler.NewEngine(db, miniMaxClient, eventBus, notifier,
		&agent.GitAgent{DB: db, UserID: 0, Encryptor: encryptor, Weaviate: weaviateClient},
		&agent.JiraAgent{DB: db, UserID: 0, Weaviate: weaviateClient},
		&agent.ReportAgent{DB: db, UserID: 0, Generator: reportGen, R2: r2Client},
		&agent.ProofAgent{DB: db, UserID: 0},
		&agent.BriefingAgent{DB: db, UserID: 0},
	)
	scheduleEngine.Start(ctx)
}
```

**IMPORTANT:** Agents are user-scoped (each has a `UserID` field). The engine stores shared dependencies (DB, Encryptor, etc.) and the executor builds fresh user-scoped agents per run using the schedule's `UserID`. This means the `Engine` and `Executor` need agent-building dependencies rather than pre-built agents. The executor's `Agents` map should be built per-run in `executeSchedule` using the schedule's `UserID`.

- [ ] **Step 4: Add Bot API accessor to telegram Bot**

In `backend/internal/services/telegram/bot.go`, add:

```go
// API returns the underlying BotAPI for sending messages.
func (b *Bot) API() *tgbotapi.BotAPI {
	return b.api
}
```

- [ ] **Step 5: Create schedule handler and register routes**

After the handler initialization block:

```go
scheduleHandler := &handlers.ScheduleHandler{
	DB:     db,
	Engine: scheduleEngine,
}
```

In the route registration, add after the conversations routes:

```go
schedules := protected.Group("/schedules")
{
	schedules.GET("", scheduleHandler.List)
	schedules.POST("", scheduleHandler.Create)
	schedules.PUT("/:id", scheduleHandler.Update)
	schedules.DELETE("/:id", scheduleHandler.Delete)
	schedules.POST("/:id/toggle", scheduleHandler.Toggle)
	schedules.POST("/:id/run", scheduleHandler.RunNow)
	schedules.GET("/:id/runs", scheduleHandler.ListRuns)
	schedules.GET("/runs/:runId", scheduleHandler.GetRun)
}
```

- [ ] **Step 6: Verify compilation**

```bash
cd backend && go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add backend/cmd/server/main.go backend/internal/services/telegram/bot.go
git commit -m "feat(scheduler): wire engine, event bus, and REST API routes in main.go"
```

---

### Task 5: Build verification and all tests

**Files:** None (verification only)

- [ ] **Step 1: Full build**

```bash
cd backend && go build ./...
```

- [ ] **Step 2: Run all scheduler tests**

```bash
cd backend && go test ./internal/scheduler/... -v
```

- [ ] **Step 3: Run go vet**

```bash
cd backend && go vet ./...
```

- [ ] **Step 4: Run all tests**

```bash
cd backend && go test ./... 2>&1 | tail -20
```
