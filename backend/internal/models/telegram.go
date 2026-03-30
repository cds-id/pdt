package models

import "time"

type TelegramConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	BotToken  string    `gorm:"type:text;not null" json:"-"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
}

type TelegramWhitelist struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	UserID           uint           `gorm:"index;not null" json:"user_id"`
	TelegramConfigID uint           `gorm:"index;not null" json:"telegram_config_id"`
	TelegramUserID   int64          `gorm:"index;not null" json:"telegram_user_id"`
	DisplayName      string         `gorm:"type:varchar(200)" json:"display_name"`
	CreatedAt        time.Time      `json:"created_at"`
	User             User           `gorm:"foreignKey:UserID" json:"-"`
	TelegramConfig   TelegramConfig `gorm:"foreignKey:TelegramConfigID" json:"-"`
}
