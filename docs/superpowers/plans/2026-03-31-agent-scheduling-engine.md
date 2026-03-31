# Agent Scheduling Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the backend scheduling engine that runs agents on cron/interval/event triggers with chaining support and execution tracking.

**Architecture:** DB-polled scheduler (30s) with bounded goroutine pool, in-process event bus, and per-run executor that creates conversations and persists run history. Agents can chain via explicit config or dynamic `trigger_agent` tool calls.

**Tech Stack:** Go, GORM (MySQL), `github.com/robfig/cron/v3` (parser only), existing `minimax` + `agent` packages

**Spec:** `docs/superpowers/specs/2026-03-31-agent-scheduling-design.md`

**Scope:** This plan covers the engine only (models, event bus, cron, pool, executor, chaining). REST API, Telegram notifications, and frontend are separate plans.

---

## File Structure

```
backend/internal/
├── models/
│   └── schedule.go              # 3 new models: AgentSchedule, AgentScheduleRun, AgentScheduleRunStep
├── scheduler/
│   ├── eventbus/
│   │   ├── bus.go               # In-process pub/sub event bus
│   │   └── bus_test.go
│   ├── cron.go                  # Cron expression parsing + next-run computation
│   ├── cron_test.go
│   ├── pool.go                  # Bounded goroutine pool
│   ├── pool_test.go
│   ├── executor.go              # Single-run executor (agent invocation, chaining, persistence)
│   ├── executor_test.go
│   ├── engine.go                # Main scheduler loop (poll + dispatch)
│   └── engine_test.go
├── ai/agent/
│   └── trigger_tool.go          # trigger_agent tool injected during scheduled runs
```

---

### Task 1: Add models

**Files:**
- Create: `backend/internal/models/schedule.go`

- [ ] **Step 1: Create the schedule models**

```go
// backend/internal/models/schedule.go
package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgentSchedule struct {
	ID              string          `gorm:"type:varchar(36);primarykey" json:"id"`
	UserID          uint            `gorm:"index;not null" json:"user_id"`
	Name            string          `gorm:"type:varchar(255);not null" json:"name"`
	AgentName       string          `gorm:"type:varchar(50)" json:"agent_name"`
	Prompt          string          `gorm:"type:text;not null" json:"prompt"`
	TriggerType     string          `gorm:"type:varchar(20);not null" json:"trigger_type"` // cron, interval, event
	CronExpr        string          `gorm:"type:varchar(100)" json:"cron_expr,omitempty"`
	IntervalSeconds int             `gorm:"default:0" json:"interval_seconds,omitempty"`
	EventName       string          `gorm:"type:varchar(100)" json:"event_name,omitempty"`
	ChainConfig     json.RawMessage `gorm:"type:json" json:"chain_config,omitempty"`
	Enabled         bool            `gorm:"default:true" json:"enabled"`
	NextRunAt       *time.Time      `gorm:"index" json:"next_run_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	User            User            `gorm:"foreignKey:UserID" json:"-"`
}

func (s *AgentSchedule) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

type ChainStep struct {
	Agent     string `json:"agent"`
	Prompt    string `json:"prompt"`
	Condition string `json:"condition"` // "always", "contains:<keyword>", "status:completed", "status:failed"
}

type AgentScheduleRun struct {
	ID             string     `gorm:"type:varchar(36);primarykey" json:"id"`
	ScheduleID     string     `gorm:"type:varchar(36);index;not null" json:"schedule_id"`
	UserID         uint       `gorm:"index;not null" json:"user_id"`
	ConversationID string     `gorm:"type:varchar(36)" json:"conversation_id,omitempty"`
	Status         string     `gorm:"type:varchar(20);not null;default:pending" json:"status"` // pending, running, completed, failed
	TriggerType    string     `gorm:"type:varchar(20);not null" json:"trigger_type"`            // cron, interval, event, manual, chain
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ResultSummary  string     `gorm:"type:text" json:"result_summary,omitempty"`
	Error          string     `gorm:"type:text" json:"error,omitempty"`
	TokenUsage     string     `gorm:"type:json" json:"token_usage,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`

	Schedule AgentSchedule `gorm:"foreignKey:ScheduleID" json:"-"`
	User     User          `gorm:"foreignKey:UserID" json:"-"`
}

func (r *AgentScheduleRun) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

type AgentScheduleRunStep struct {
	ID         string    `gorm:"type:varchar(36);primarykey" json:"id"`
	RunID      string    `gorm:"type:varchar(36);index;not null" json:"run_id"`
	AgentName  string    `gorm:"type:varchar(50);not null" json:"agent_name"`
	Prompt     string    `gorm:"type:text;not null" json:"prompt"`
	Response   string    `gorm:"type:text" json:"response"`
	Status     string    `gorm:"type:varchar(20);not null" json:"status"` // completed, failed
	DurationMs int       `json:"duration_ms"`
	CreatedAt  time.Time `json:"created_at"`

	Run AgentScheduleRun `gorm:"foreignKey:RunID" json:"-"`
}

func (s *AgentScheduleRunStep) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}
```

- [ ] **Step 2: Add AutoMigrate in main.go**

In `backend/cmd/server/main.go`, find the existing `db.AutoMigrate(...)` call and append the 3 new models:

```go
&models.AgentSchedule{},
&models.AgentScheduleRun{},
&models.AgentScheduleRunStep{},
```

- [ ] **Step 3: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/models/schedule.go backend/cmd/server/main.go
git commit -m "feat(scheduler): add schedule, run, and run step models"
```

---

### Task 2: Add robfig/cron dependency and cron parser

**Files:**
- Create: `backend/internal/scheduler/cron.go`
- Create: `backend/internal/scheduler/cron_test.go`

- [ ] **Step 1: Install robfig/cron**

```bash
cd backend && go get github.com/robfig/cron/v3
```

- [ ] **Step 2: Write failing tests for cron parsing**

```go
// backend/internal/scheduler/cron_test.go
package scheduler

import (
	"testing"
	"time"
)

func TestNextCronRun(t *testing.T) {
	// Fixed reference time: Monday 2026-03-30 10:00:00 UTC
	ref := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		expr     string
		after    time.Time
		expected time.Time
		wantErr  bool
	}{
		{
			name:     "every weekday at 8am",
			expr:     "0 8 * * 1-5",
			after:    ref,
			expected: time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC), // Tuesday
		},
		{
			name:     "every hour",
			expr:     "0 * * * *",
			after:    ref,
			expected: time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC),
		},
		{
			name:    "invalid expression",
			expr:    "not a cron",
			after:   ref,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextCronRun(tt.expr, tt.after)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.expected) {
				t.Errorf("NextCronRun(%q, %v)\n  got:  %v\n  want: %v", tt.expr, tt.after, got, tt.expected)
			}
		})
	}
}

func TestNextRunAt(t *testing.T) {
	ref := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		triggerType     string
		cronExpr        string
		intervalSeconds int
		after           time.Time
		wantErr         bool
	}{
		{
			name:        "cron trigger",
			triggerType: "cron",
			cronExpr:    "0 8 * * 1-5",
			after:       ref,
		},
		{
			name:            "interval trigger",
			triggerType:     "interval",
			intervalSeconds: 900,
			after:           ref,
		},
		{
			name:        "event trigger returns nil",
			triggerType: "event",
			after:       ref,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextRunAt(tt.triggerType, tt.cronExpr, tt.intervalSeconds, tt.after)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			switch tt.triggerType {
			case "cron":
				if got == nil {
					t.Fatal("expected non-nil time for cron")
				}
			case "interval":
				if got == nil {
					t.Fatal("expected non-nil time for interval")
				}
				expected := ref.Add(900 * time.Second)
				if !got.Equal(expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			case "event":
				if got != nil {
					t.Errorf("expected nil for event trigger, got %v", got)
				}
			}
		})
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
cd backend && go test ./internal/scheduler/ -v
```

Expected: compilation error — package doesn't exist.

- [ ] **Step 4: Implement cron.go**

```go
// backend/internal/scheduler/cron.go
package scheduler

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// NextCronRun parses a 5-field cron expression and returns the next run time after the given time.
func NextCronRun(expr string, after time.Time) (time.Time, error) {
	schedule, err := cronParser.Parse(expr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return schedule.Next(after), nil
}

// NextRunAt computes the next run time for a schedule based on its trigger type.
// Returns nil for event-triggered schedules (they run on event, not on time).
func NextRunAt(triggerType, cronExpr string, intervalSeconds int, after time.Time) (*time.Time, error) {
	switch triggerType {
	case "cron":
		next, err := NextCronRun(cronExpr, after)
		if err != nil {
			return nil, err
		}
		return &next, nil
	case "interval":
		next := after.Add(time.Duration(intervalSeconds) * time.Second)
		return &next, nil
	case "event":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown trigger type: %s", triggerType)
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd backend && go test ./internal/scheduler/ -v
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/scheduler/ backend/go.mod backend/go.sum
git commit -m "feat(scheduler): add cron expression parser and next-run computation"
```

---

### Task 3: Event bus

**Files:**
- Create: `backend/internal/scheduler/eventbus/bus.go`
- Create: `backend/internal/scheduler/eventbus/bus_test.go`

- [ ] **Step 1: Write failing tests**

```go
// backend/internal/scheduler/eventbus/bus_test.go
package eventbus

import (
	"sync"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	bus := New()
	defer bus.Close()

	var received map[string]any
	var mu sync.Mutex
	done := make(chan struct{})

	bus.Subscribe("test_event", func(payload map[string]any) {
		mu.Lock()
		received = payload
		mu.Unlock()
		close(done)
	})

	bus.Publish("test_event", map[string]any{"key": "value"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	mu.Lock()
	defer mu.Unlock()
	if received["key"] != "value" {
		t.Errorf("got %v, want key=value", received)
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := New()
	defer bus.Close()

	called := false
	unsub := bus.Subscribe("test_event", func(payload map[string]any) {
		called = true
	})

	unsub()
	bus.Publish("test_event", map[string]any{})
	time.Sleep(50 * time.Millisecond)

	if called {
		t.Error("handler should not have been called after unsubscribe")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := New()
	defer bus.Close()

	var count int
	var mu sync.Mutex
	done := make(chan struct{})

	for i := 0; i < 3; i++ {
		bus.Subscribe("test_event", func(payload map[string]any) {
			mu.Lock()
			count++
			if count == 3 {
				close(done)
			}
			mu.Unlock()
		})
	}

	bus.Publish("test_event", map[string]any{})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for all subscribers")
	}

	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Errorf("expected 3 calls, got %d", count)
	}
}

func TestNoSubscribers(t *testing.T) {
	bus := New()
	defer bus.Close()
	// Should not panic
	bus.Publish("nonexistent", map[string]any{})
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend && go test ./internal/scheduler/eventbus/ -v
```

Expected: compilation error.

- [ ] **Step 3: Implement event bus**

```go
// backend/internal/scheduler/eventbus/bus.go
package eventbus

import (
	"sync"
)

type handler struct {
	id int
	fn func(payload map[string]any)
}

// Bus is an in-process pub/sub event bus.
type Bus struct {
	mu       sync.RWMutex
	subs     map[string][]handler
	nextID   int
	closed   bool
}

// New creates a new event bus.
func New() *Bus {
	return &Bus{
		subs: make(map[string][]handler),
	}
}

// Subscribe registers a handler for an event. Returns an unsubscribe function.
func (b *Bus) Subscribe(event string, fn func(payload map[string]any)) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++
	b.subs[event] = append(b.subs[event], handler{id: id, fn: fn})

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		handlers := b.subs[event]
		for i, h := range handlers {
			if h.id == id {
				b.subs[event] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}

// Publish sends an event to all subscribers. Handlers are called in goroutines.
func (b *Bus) Publish(event string, payload map[string]any) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	for _, h := range b.subs[event] {
		go h.fn(payload)
	}
}

// Close prevents further event delivery.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./internal/scheduler/eventbus/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/scheduler/eventbus/
git commit -m "feat(scheduler): add in-process event bus"
```

---

### Task 4: Goroutine pool

**Files:**
- Create: `backend/internal/scheduler/pool.go`
- Create: `backend/internal/scheduler/pool_test.go`

- [ ] **Step 1: Write failing tests**

```go
// backend/internal/scheduler/pool_test.go
package scheduler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolConcurrencyLimit(t *testing.T) {
	pool := NewPool(2)
	defer pool.Stop()

	var maxConcurrent int32
	var current int32
	var mu sync.Mutex
	done := make(chan struct{})
	var completed int32

	for i := 0; i < 5; i++ {
		pool.Submit(func() {
			c := atomic.AddInt32(&current, 1)
			mu.Lock()
			if c > maxConcurrent {
				maxConcurrent = c
			}
			mu.Unlock()

			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&current, -1)

			if atomic.AddInt32(&completed, 1) == 5 {
				close(done)
			}
		})
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}

	mu.Lock()
	defer mu.Unlock()
	if maxConcurrent > 2 {
		t.Errorf("max concurrent was %d, expected <= 2", maxConcurrent)
	}
	if atomic.LoadInt32(&completed) != 5 {
		t.Errorf("expected 5 completed, got %d", completed)
	}
}

func TestPoolStop(t *testing.T) {
	pool := NewPool(1)
	var ran atomic.Bool

	pool.Stop()

	pool.Submit(func() {
		ran.Store(true)
	})

	time.Sleep(50 * time.Millisecond)
	if ran.Load() {
		t.Error("job should not run after Stop")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend && go test ./internal/scheduler/ -v -run TestPool
```

Expected: compilation error.

- [ ] **Step 3: Implement pool**

```go
// backend/internal/scheduler/pool.go
package scheduler

// Pool is a bounded goroutine pool for executing scheduled jobs.
type Pool struct {
	jobs    chan func()
	stop    chan struct{}
	stopped bool
}

// NewPool creates a pool with the given max concurrency.
func NewPool(maxWorkers int) *Pool {
	p := &Pool{
		jobs: make(chan func(), 100),
		stop: make(chan struct{}),
	}
	for i := 0; i < maxWorkers; i++ {
		go p.worker()
	}
	return p
}

func (p *Pool) worker() {
	for {
		select {
		case <-p.stop:
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			job()
		}
	}
}

// Submit adds a job to the pool. If the pool is stopped, the job is dropped.
func (p *Pool) Submit(job func()) {
	if p.stopped {
		return
	}
	select {
	case p.jobs <- job:
	case <-p.stop:
	}
}

// Stop shuts down the pool. Workers finish current jobs but accept no new ones.
func (p *Pool) Stop() {
	if p.stopped {
		return
	}
	p.stopped = true
	close(p.stop)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./internal/scheduler/ -v -run TestPool
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/scheduler/pool.go backend/internal/scheduler/pool_test.go
git commit -m "feat(scheduler): add bounded goroutine pool"
```

---

### Task 5: Executor — core run logic

**Files:**
- Create: `backend/internal/scheduler/executor.go`
- Create: `backend/internal/scheduler/executor_test.go`

- [ ] **Step 1: Write the executor with conversation creation and agent invocation**

```go
// backend/internal/scheduler/executor.go
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/gorm"
)

// Executor runs a single scheduled agent invocation.
type Executor struct {
	DB     *gorm.DB
	Client *minimax.Client
	Agents map[string]agent.Agent
}

// nopWriter is a StreamWriter that discards all output (scheduled runs don't stream).
type nopWriter struct{}

func (nopWriter) WriteContent(string) error            { return nil }
func (nopWriter) WriteThinking(string) error            { return nil }
func (nopWriter) WriteToolStatus(string, string) error  { return nil }
func (nopWriter) WriteDone() error                      { return nil }
func (nopWriter) WriteError(string) error               { return nil }

// Run executes a schedule: creates a conversation, runs the agent, persists the result.
func (e *Executor) Run(ctx context.Context, schedule models.AgentSchedule, triggerType string) (*models.AgentScheduleRun, error) {
	now := time.Now()
	run := models.AgentScheduleRun{
		ScheduleID:  schedule.ID,
		UserID:      schedule.UserID,
		Status:      "running",
		TriggerType: triggerType,
		StartedAt:   &now,
	}
	if err := e.DB.Create(&run).Error; err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	// Create conversation
	conv := models.Conversation{
		UserID: schedule.UserID,
		Title:  fmt.Sprintf("Scheduled: %s — %s", schedule.Name, now.Format("2006-01-02")),
	}
	if err := e.DB.Create(&conv).Error; err != nil {
		e.failRun(&run, fmt.Errorf("create conversation: %w", err))
		return &run, nil
	}
	run.ConversationID = conv.ID

	// Save user message (the prompt)
	userMsg := models.ChatMessage{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        schedule.Prompt,
	}
	e.DB.Create(&userMsg)

	// Run the agent
	messages := []minimax.Message{{Role: "user", Content: schedule.Prompt}}
	result, err := e.runAgent(ctx, schedule.AgentName, messages, &run)
	if err != nil {
		e.failRun(&run, err)
		return &run, nil
	}

	// Save assistant response
	if result.FullResponse != "" {
		assistantMsg := models.ChatMessage{
			ConversationID: conv.ID,
			Role:           "assistant",
			Content:        result.FullResponse,
		}
		e.DB.Create(&assistantMsg)
	}

	// Evaluate explicit chain
	if len(schedule.ChainConfig) > 0 {
		e.runChain(ctx, schedule.ChainConfig, result.FullResponse, &run)
	}

	// Complete the run
	completedAt := time.Now()
	usageJSON, _ := json.Marshal(map[string]int{
		"prompt_tokens":     result.Usage.PromptTokens,
		"completion_tokens": result.Usage.CompletionTokens,
	})
	e.DB.Model(&run).Updates(map[string]any{
		"status":          "completed",
		"completed_at":    &completedAt,
		"conversation_id": conv.ID,
		"result_summary":  summarize(result.FullResponse, 500),
		"token_usage":     string(usageJSON),
	})
	run.Status = "completed"

	return &run, nil
}

func (e *Executor) runAgent(ctx context.Context, agentName string, messages []minimax.Message, run *models.AgentScheduleRun) (*agent.LoopResult, error) {
	start := time.Now()

	var result *agent.LoopResult
	var err error

	if agentName == "" {
		// Use orchestrator routing
		orchestrator := agent.NewOrchestrator(e.Client, e.agentSlice()...)
		result, err = orchestrator.HandleMessage(ctx, messages, nopWriter{})
	} else {
		a, ok := e.Agents[agentName]
		if !ok {
			return nil, fmt.Errorf("unknown agent: %s", agentName)
		}
		result, err = agent.RunLoop(ctx, e.Client, a, messages, nopWriter{})
	}

	status := "completed"
	response := ""
	if err != nil {
		status = "failed"
		response = err.Error()
	} else {
		response = result.FullResponse
	}

	// Record step
	step := models.AgentScheduleRunStep{
		RunID:      run.ID,
		AgentName:  agentName,
		Prompt:     messages[len(messages)-1].Content,
		Response:   response,
		Status:     status,
		DurationMs: int(time.Since(start).Milliseconds()),
	}
	e.DB.Create(&step)

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e *Executor) runChain(ctx context.Context, chainConfigJSON json.RawMessage, previousResponse string, run *models.AgentScheduleRun) {
	var steps []models.ChainStep
	if err := json.Unmarshal(chainConfigJSON, &steps); err != nil {
		log.Printf("[scheduler] invalid chain config: %v", err)
		return
	}

	for _, step := range steps {
		if !evaluateCondition(step.Condition, previousResponse, "completed") {
			continue
		}
		messages := []minimax.Message{{Role: "user", Content: step.Prompt}}
		result, err := e.runAgent(ctx, step.Agent, messages, run)
		if err != nil {
			log.Printf("[scheduler] chain step %s failed: %v", step.Agent, err)
			continue
		}
		previousResponse = result.FullResponse
	}
}

func (e *Executor) failRun(run *models.AgentScheduleRun, err error) {
	now := time.Now()
	e.DB.Model(run).Updates(map[string]any{
		"status":       "failed",
		"completed_at": &now,
		"error":        err.Error(),
	})
	run.Status = "failed"
	run.Error = err.Error()
}

func (e *Executor) agentSlice() []agent.Agent {
	agents := make([]agent.Agent, 0, len(e.Agents))
	for _, a := range e.Agents {
		agents = append(agents, a)
	}
	return agents
}

func evaluateCondition(condition, response, status string) bool {
	switch {
	case condition == "always" || condition == "":
		return true
	case condition == "status:completed":
		return status == "completed"
	case condition == "status:failed":
		return status == "failed"
	case strings.HasPrefix(condition, "contains:"):
		keyword := strings.TrimPrefix(condition, "contains:")
		return strings.Contains(strings.ToLower(response), strings.ToLower(keyword))
	default:
		return false
	}
}

func summarize(text string, maxLen int) string {
	// Use first heading if available
	lines := strings.SplitN(text, "\n", 5)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimLeft(trimmed, "# ")
			if len(heading) > 0 && len(heading) <= maxLen {
				return heading
			}
		}
	}
	// Fall back to first N chars
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen])
}
```

- [ ] **Step 2: Write tests for evaluateCondition and summarize**

```go
// backend/internal/scheduler/executor_test.go
package scheduler

import (
	"testing"
)

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		response  string
		status    string
		want      bool
	}{
		{"always", "always", "anything", "completed", true},
		{"empty condition", "", "anything", "completed", true},
		{"status completed match", "status:completed", "", "completed", true},
		{"status completed no match", "status:completed", "", "failed", false},
		{"status failed match", "status:failed", "", "failed", true},
		{"contains match", "contains:blocker", "There is a BLOCKER in sprint", "completed", true},
		{"contains no match", "contains:blocker", "All clear", "completed", false},
		{"unknown condition", "unknown:foo", "", "completed", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateCondition(tt.condition, tt.response, tt.status)
			if got != tt.want {
				t.Errorf("evaluateCondition(%q, %q, %q) = %v, want %v",
					tt.condition, tt.response, tt.status, got, tt.want)
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short text",
			input:  "Hello world",
			maxLen: 500,
			want:   "Hello world",
		},
		{
			name:   "with heading",
			input:  "# Daily Report\n\nSome content here",
			maxLen: 500,
			want:   "Daily Report",
		},
		{
			name:   "long text truncated",
			input:  "This is a very long text that should be truncated",
			maxLen: 20,
			want:   "This is a very long ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarize(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("summarize(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 3: Run tests**

```bash
cd backend && go test ./internal/scheduler/ -v -run "TestEvaluateCondition|TestSummarize"
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/scheduler/executor.go backend/internal/scheduler/executor_test.go
git commit -m "feat(scheduler): add executor with conversation creation and chain evaluation"
```

---

### Task 6: trigger_agent tool for dynamic chaining

**Files:**
- Create: `backend/internal/ai/agent/trigger_tool.go`

- [ ] **Step 1: Implement the trigger_agent tool definition**

```go
// backend/internal/ai/agent/trigger_tool.go
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

// TriggerAgentTool is a tool injected during scheduled runs that allows agents
// to dynamically trigger other agents. Not available in interactive chat.
type TriggerAgentTool struct {
	Agents   map[string]Agent
	Client   *minimax.Client
	MaxDepth int
	Depth    int
}

type triggerArgs struct {
	Agent  string `json:"agent"`
	Prompt string `json:"prompt"`
}

// Definition returns the tool schema for trigger_agent.
func (t *TriggerAgentTool) Definition() minimax.Tool {
	return minimax.Tool{
		Type: "function",
		Function: minimax.Function{
			Name:        "trigger_agent",
			Description: "Trigger another agent with a specific prompt. Use this to delegate tasks to specialist agents.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"agent": {
						"type": "string",
						"enum": ["git", "jira", "report", "proof", "briefing", "whatsapp"],
						"description": "The agent to trigger"
					},
					"prompt": {
						"type": "string",
						"description": "The message to send to the agent"
					}
				},
				"required": ["agent", "prompt"]
			}`),
		},
	}
}

// Execute runs the target agent and returns its response.
func (t *TriggerAgentTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if t.Depth >= t.MaxDepth {
		return map[string]string{"error": "maximum chain depth reached"}, nil
	}

	var a triggerArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("parse trigger_agent args: %w", err)
	}

	target, ok := t.Agents[a.Agent]
	if !ok {
		return map[string]string{"error": fmt.Sprintf("unknown agent: %s", a.Agent)}, nil
	}

	messages := []minimax.Message{{Role: "user", Content: a.Prompt}}
	result, err := RunLoop(ctx, t.Client, target, messages, nopStreamWriter{})
	if err != nil {
		return map[string]string{"error": err.Error()}, nil
	}

	return map[string]string{"response": result.FullResponse}, nil
}

// nopStreamWriter discards all streaming output.
type nopStreamWriter struct{}

func (nopStreamWriter) WriteContent(string) error            { return nil }
func (nopStreamWriter) WriteThinking(string) error            { return nil }
func (nopStreamWriter) WriteToolStatus(string, string) error  { return nil }
func (nopStreamWriter) WriteDone() error                      { return nil }
func (nopStreamWriter) WriteError(string) error               { return nil }
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: compiles.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/ai/agent/trigger_tool.go
git commit -m "feat(scheduler): add trigger_agent tool for dynamic agent chaining"
```

---

### Task 7: Engine — main scheduler loop

**Files:**
- Create: `backend/internal/scheduler/engine.go`
- Create: `backend/internal/scheduler/engine_test.go`

- [ ] **Step 1: Implement the engine**

```go
// backend/internal/scheduler/engine.go
package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/scheduler/eventbus"
	"gorm.io/gorm"
)

const (
	defaultPollInterval = 30 * time.Second
	defaultMaxWorkers   = 3
)

// Engine polls the database for due schedules and dispatches them to a worker pool.
type Engine struct {
	db       *gorm.DB
	client   *minimax.Client
	agents   map[string]agent.Agent
	pool     *Pool
	bus      *eventbus.Bus
	unsubs   []func()
	mu       sync.Mutex
}

// NewEngine creates a new scheduler engine.
func NewEngine(db *gorm.DB, client *minimax.Client, bus *eventbus.Bus, agents ...agent.Agent) *Engine {
	agentMap := make(map[string]agent.Agent)
	for _, a := range agents {
		agentMap[a.Name()] = a
	}
	return &Engine{
		db:     db,
		client: client,
		agents: agentMap,
		pool:   NewPool(defaultMaxWorkers),
		bus:    bus,
	}
}

// Start begins the polling loop and event subscriptions.
func (e *Engine) Start(ctx context.Context) {
	e.subscribeEvents()
	go e.pollLoop(ctx)
	log.Println("[scheduler] engine started")
}

// Stop shuts down the engine.
func (e *Engine) Stop() {
	e.mu.Lock()
	for _, unsub := range e.unsubs {
		unsub()
	}
	e.unsubs = nil
	e.mu.Unlock()
	e.pool.Stop()
	log.Println("[scheduler] engine stopped")
}

func (e *Engine) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(defaultPollInterval)
	defer ticker.Stop()

	// Run immediately on start
	e.pollAndDispatch(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.pollAndDispatch(ctx)
		}
	}
}

func (e *Engine) pollAndDispatch(ctx context.Context) {
	now := time.Now()
	var schedules []models.AgentSchedule
	e.db.Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, now).
		Find(&schedules)

	for _, s := range schedules {
		schedule := s // capture for closure
		e.pool.Submit(func() {
			e.executeSchedule(ctx, schedule, schedule.TriggerType)
		})
	}
}

func (e *Engine) executeSchedule(ctx context.Context, schedule models.AgentSchedule, triggerType string) {
	log.Printf("[scheduler] executing schedule %q (id=%s, trigger=%s)", schedule.Name, schedule.ID, triggerType)

	executor := &Executor{
		DB:     e.db,
		Client: e.client,
		Agents: e.agents,
	}

	run, err := executor.Run(ctx, schedule, triggerType)
	if err != nil {
		log.Printf("[scheduler] schedule %q execution error: %v", schedule.Name, err)
		return
	}

	// Update next_run_at
	nextRun, err := NextRunAt(schedule.TriggerType, schedule.CronExpr, schedule.IntervalSeconds, time.Now())
	if err != nil {
		log.Printf("[scheduler] failed to compute next run for %q: %v", schedule.Name, err)
	}
	e.db.Model(&schedule).Update("next_run_at", nextRun)

	// Emit completion event
	if e.bus != nil {
		e.bus.Publish("schedule_completed", map[string]any{
			"user_id":       schedule.UserID,
			"schedule_id":   schedule.ID,
			"schedule_name": schedule.Name,
			"status":        run.Status,
		})
	}

	log.Printf("[scheduler] schedule %q completed with status %s", schedule.Name, run.Status)
}

func (e *Engine) subscribeEvents() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Unsubscribe existing
	for _, unsub := range e.unsubs {
		unsub()
	}
	e.unsubs = nil

	// Find all enabled event-triggered schedules
	var schedules []models.AgentSchedule
	e.db.Where("enabled = ? AND trigger_type = ?", true, "event").Find(&schedules)

	for _, s := range schedules {
		schedule := s
		unsub := e.bus.Subscribe(schedule.EventName, func(payload map[string]any) {
			// Filter by user_id
			if uid, ok := payload["user_id"]; ok {
				if userID, ok := uid.(uint); ok && userID != schedule.UserID {
					return
				}
			}
			e.pool.Submit(func() {
				e.executeSchedule(context.Background(), schedule, "event")
			})
		})
		e.unsubs = append(e.unsubs, unsub)
	}

	log.Printf("[scheduler] subscribed to events for %d schedules", len(schedules))
}

// RefreshEventSubscriptions re-subscribes event-triggered schedules.
// Call this after creating/updating/deleting schedules.
func (e *Engine) RefreshEventSubscriptions() {
	e.subscribeEvents()
}
```

- [ ] **Step 2: Write engine test for polling logic**

```go
// backend/internal/scheduler/engine_test.go
package scheduler

import (
	"testing"
)

func TestEngineCreation(t *testing.T) {
	// Verify engine can be constructed without panic
	e := NewEngine(nil, nil, nil)
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if e.pool == nil {
		t.Error("expected pool to be initialized")
	}
	if e.agents == nil {
		t.Error("expected agents map to be initialized")
	}
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: compiles.

- [ ] **Step 4: Run all scheduler tests**

```bash
cd backend && go test ./internal/scheduler/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/scheduler/engine.go backend/internal/scheduler/engine_test.go
git commit -m "feat(scheduler): add engine with DB polling, event subscriptions, and pool dispatch"
```

---

### Task 8: Full build verification

**Files:** None (verification only)

- [ ] **Step 1: Full build**

```bash
cd backend && go build ./...
```

Expected: compiles.

- [ ] **Step 2: Run all tests**

```bash
cd backend && go test ./internal/scheduler/... -v
```

Expected: all PASS.

- [ ] **Step 3: Run go vet**

```bash
cd backend && go vet ./internal/scheduler/...
```

Expected: no issues.

- [ ] **Step 4: Run all backend tests**

```bash
cd backend && go test ./... 2>&1 | tail -20
```

Expected: no regressions.
