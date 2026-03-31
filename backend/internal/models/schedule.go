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
	TriggerType     string          `gorm:"type:varchar(20);not null" json:"trigger_type"`
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
	Condition string `json:"condition"`
}

type AgentScheduleRun struct {
	ID             string        `gorm:"type:varchar(36);primarykey" json:"id"`
	ScheduleID     string        `gorm:"type:varchar(36);index;not null" json:"schedule_id"`
	UserID         uint          `gorm:"index;not null" json:"user_id"`
	ConversationID string        `gorm:"type:varchar(36)" json:"conversation_id,omitempty"`
	Status         string        `gorm:"type:varchar(20);not null;default:pending" json:"status"`
	TriggerType    string        `gorm:"type:varchar(20);not null" json:"trigger_type"`
	StartedAt      *time.Time    `json:"started_at,omitempty"`
	CompletedAt    *time.Time    `json:"completed_at,omitempty"`
	ResultSummary  string        `gorm:"type:text" json:"result_summary,omitempty"`
	Error          string        `gorm:"type:text" json:"error,omitempty"`
	TokenUsage     *string       `gorm:"type:json" json:"token_usage,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	Schedule       AgentSchedule `gorm:"foreignKey:ScheduleID" json:"-"`
	User           User          `gorm:"foreignKey:UserID" json:"-"`
}

func (r *AgentScheduleRun) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

type AgentScheduleRunStep struct {
	ID        string           `gorm:"type:varchar(36);primarykey" json:"id"`
	RunID     string           `gorm:"type:varchar(36);index;not null" json:"run_id"`
	AgentName string           `gorm:"type:varchar(50);not null" json:"agent_name"`
	Prompt    string           `gorm:"type:text;not null" json:"prompt"`
	Response  string           `gorm:"type:text" json:"response"`
	Status    string           `gorm:"type:varchar(20);not null" json:"status"`
	DurationMs int             `json:"duration_ms"`
	CreatedAt time.Time        `json:"created_at"`
	Run       AgentScheduleRun `gorm:"foreignKey:RunID" json:"-"`
}

func (s *AgentScheduleRunStep) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}
