package models

import "time"

type JiraComment struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	CardKey     string    `gorm:"type:varchar(50);index;not null" json:"card_key"`
	CommentID   string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"comment_id"`
	Author      string    `gorm:"type:varchar(255)" json:"author"`
	AuthorEmail string    `gorm:"type:varchar(255)" json:"author_email"`
	Body        string    `gorm:"type:text" json:"body"`
	CommentedAt time.Time `gorm:"index" json:"commented_at"`
	CreatedAt   time.Time `json:"created_at"`
	User        User      `gorm:"foreignKey:UserID" json:"-"`
}
