package models

import "time"

// ComposioConfig stores a user's Composio API key (encrypted).
type ComposioConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	APIKey    string    `gorm:"type:text;not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ComposioConnection tracks a user's connected Composio service.
type ComposioConnection struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	UserID        uint      `gorm:"not null;uniqueIndex:idx_user_toolkit" json:"user_id"`
	Toolkit       string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_user_toolkit" json:"toolkit"`
	AuthConfigID  string    `gorm:"type:varchar(255)" json:"auth_config_id"`
	IntegrationID string    `gorm:"type:varchar(255)" json:"integration_id"`
	AccountID     string    `gorm:"type:varchar(255)" json:"account_id"`
	Status        string    `gorm:"type:varchar(50);default:inactive" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
