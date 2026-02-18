package models

import "time"

type ReportTemplate struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	IsDefault bool      `gorm:"default:false" json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
}

type Report struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	TemplateID *uint     `gorm:"index" json:"template_id"`
	Date       string    `gorm:"type:varchar(10);index;not null" json:"date"`
	Title      string    `gorm:"type:varchar(500)" json:"title"`
	Content    string    `gorm:"type:text" json:"content"`
	FileURL    string    `gorm:"type:varchar(500)" json:"file_url"`
	CreatedAt  time.Time `json:"created_at"`
	User       User      `gorm:"foreignKey:UserID" json:"-"`
}
