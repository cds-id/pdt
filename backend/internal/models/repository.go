package models

import "time"

type Provider string

const (
	ProviderGitHub Provider = "github"
	ProviderGitLab Provider = "gitlab"
)

type Repository struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	Name         string    `gorm:"type:varchar(500);not null" json:"name"`
	Owner        string    `gorm:"type:varchar(255);not null" json:"owner"`
	Provider     Provider  `gorm:"type:varchar(10);not null" json:"provider"`
	URL          string    `gorm:"type:varchar(1000);not null" json:"url"`
	IsValid      bool      `gorm:"default:true" json:"is_valid"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
	CreatedAt    time.Time `json:"created_at"`
	User         User      `gorm:"foreignKey:UserID" json:"-"`
}
