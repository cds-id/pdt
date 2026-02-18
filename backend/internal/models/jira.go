package models

import "time"

type SprintState string

const (
	SprintActive SprintState = "active"
	SprintClosed SprintState = "closed"
	SprintFuture SprintState = "future"
)

type Sprint struct {
	ID           uint        `gorm:"primarykey" json:"id"`
	UserID       uint        `gorm:"index;not null" json:"user_id"`
	JiraSprintID string      `gorm:"type:varchar(50);uniqueIndex;not null" json:"jira_sprint_id"`
	Name         string      `gorm:"type:varchar(255)" json:"name"`
	State        SprintState `gorm:"type:varchar(10)" json:"state"`
	StartDate    *time.Time  `json:"start_date"`
	EndDate      *time.Time  `json:"end_date"`
	CreatedAt    time.Time   `json:"created_at"`
	User         User        `gorm:"foreignKey:UserID" json:"-"`
	Cards        []JiraCard  `gorm:"foreignKey:SprintID" json:"cards,omitempty"`
}

type JiraCard struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	Key         string    `gorm:"column:card_key;type:varchar(50);index;not null" json:"key"`
	Summary     string    `gorm:"type:text" json:"summary"`
	Status      string    `gorm:"type:varchar(100)" json:"status"`
	Assignee    string    `gorm:"type:varchar(255)" json:"assignee"`
	SprintID    *uint     `gorm:"index" json:"sprint_id"`
	DetailsJSON string    `gorm:"type:text" json:"details_json,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	User        User      `gorm:"foreignKey:UserID" json:"-"`
	Sprint      *Sprint   `gorm:"foreignKey:SprintID" json:"-"`
}
