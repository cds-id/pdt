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

func NewEngine(db *gorm.DB, client *minimax.Client, bus *eventbus.Bus, notifier *Notifier, agents ...agent.Agent) *Engine {
	agentMap := make(map[string]agent.Agent)
	for _, a := range agents {
		agentMap[a.Name()] = a
	}
	return &Engine{
		db:       db,
		client:   client,
		agents:   agentMap,
		pool:     NewPool(defaultMaxWorkers),
		bus:      bus,
		notifier: notifier,
	}
}

func (e *Engine) Start(ctx context.Context) {
	e.subscribeEvents()
	go e.pollLoop(ctx)
	log.Println("[scheduler] engine started")
}

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
	e.db.Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, now).Find(&schedules)

	for _, s := range schedules {
		schedule := s
		e.pool.Submit(func() {
			e.executeSchedule(ctx, schedule, schedule.TriggerType)
		})
	}
}

func (e *Engine) executeSchedule(ctx context.Context, schedule models.AgentSchedule, triggerType string) {
	log.Printf("[scheduler] executing schedule %q (id=%s, trigger=%s)", schedule.Name, schedule.ID, triggerType)

	executor := &Executor{
		DB:       e.db,
		Client:   e.client,
		Agents:   e.agents,
		Notifier: e.notifier,
	}

	run, err := executor.Run(ctx, schedule, triggerType)
	if err != nil {
		log.Printf("[scheduler] schedule %q execution error: %v", schedule.Name, err)
		return
	}

	nextRun, err := NextRunAt(schedule.TriggerType, schedule.CronExpr, schedule.IntervalSeconds, time.Now())
	if err != nil {
		log.Printf("[scheduler] failed to compute next run for %q: %v", schedule.Name, err)
	}
	e.db.Model(&schedule).Update("next_run_at", nextRun)

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

	for _, unsub := range e.unsubs {
		unsub()
	}
	e.unsubs = nil

	var schedules []models.AgentSchedule
	e.db.Where("enabled = ? AND trigger_type = ?", true, "event").Find(&schedules)

	for _, s := range schedules {
		schedule := s
		unsub := e.bus.Subscribe(schedule.EventName, func(payload map[string]any) {
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

func (e *Engine) RefreshEventSubscriptions() {
	e.subscribeEvents()
}

func (e *Engine) RunScheduleNow(schedule models.AgentSchedule) {
	e.pool.Submit(func() {
		e.executeSchedule(context.Background(), schedule, "manual")
	})
}
