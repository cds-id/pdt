package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Conversation struct {
	ID        string    `gorm:"type:varchar(36);primarykey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Title     string    `gorm:"type:varchar(255)" json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
	Messages  []ChatMessage `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

type ChatMessage struct {
	ID             string    `gorm:"type:varchar(36);primarykey" json:"id"`
	ConversationID string    `gorm:"type:varchar(36);index;not null" json:"conversation_id"`
	Role           string    `gorm:"type:varchar(20);not null" json:"role"`
	Content        string    `gorm:"type:text" json:"content"`
	ToolCalls      string    `gorm:"type:text" json:"tool_calls,omitempty"`
	ToolName       string    `gorm:"type:varchar(100)" json:"tool_name,omitempty"`
	ToolCallID     string    `gorm:"type:varchar(100)" json:"tool_call_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	Conversation   Conversation `gorm:"foreignKey:ConversationID" json:"-"`
}

func (m *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type AIUsage struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	ConversationID   string    `gorm:"type:varchar(36);index" json:"conversation_id"`
	Provider         string    `gorm:"type:varchar(30);index;default:minimax" json:"provider"`
	Model            string    `gorm:"type:varchar(60)" json:"model"`
	Feature          string    `gorm:"type:varchar(40);index" json:"feature"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CreatedAt        time.Time `json:"created_at"`
	User             User      `gorm:"foreignKey:UserID" json:"-"`
}
