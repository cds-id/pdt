package models

import (
	"time"

	"gorm.io/datatypes"
)

type ExecutiveReport struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	UserID             uint           `gorm:"index;not null" json:"user_id"`
	WorkspaceID        *uint          `gorm:"index" json:"workspace_id,omitempty"`
	RangeStart         time.Time      `json:"range_start"`
	RangeEnd           time.Time      `json:"range_end"`
	StaleThresholdDays int            `json:"stale_threshold_days"`
	Narrative          string         `gorm:"type:text" json:"narrative"`
	Suggestions        datatypes.JSON `json:"suggestions"`
	Dataset            datatypes.JSON `json:"dataset"`
	Status             string         `gorm:"type:varchar(16);not null;default:'generating'" json:"status"`
	ErrorMessage       string         `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	CompletedAt        *time.Time     `json:"completed_at,omitempty"`
}

func (ExecutiveReport) TableName() string { return "executive_reports" }
