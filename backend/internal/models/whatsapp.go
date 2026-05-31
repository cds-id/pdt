package models

import "time"

type WaNumber struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	PhoneNumber string     `gorm:"type:varchar(30);not null" json:"phone_number"`
	DisplayName string     `gorm:"type:varchar(100)" json:"display_name"`
	DeviceJID   string     `gorm:"column:device_jid;type:varchar(200)" json:"-"`
	Status      string     `gorm:"type:varchar(20);default:disconnected" json:"status"`
	PairedAt    *time.Time `json:"paired_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	User        User       `gorm:"foreignKey:UserID" json:"-"`
}

type WaListener struct {
	ID                 uint       `gorm:"primarykey" json:"id"`
	WaNumberID         uint       `gorm:"index;not null" json:"wa_number_id"`
	JID                string     `gorm:"column:jid;type:varchar(100);not null" json:"jid"`
	Name               string     `gorm:"type:varchar(200);not null" json:"name"`
	Type               string     `gorm:"type:varchar(20);not null" json:"type"`
	IsActive           bool       `gorm:"default:true" json:"is_active"`
	// Source distinguishes explicit monitor rows ("monitor") from auto-created
	// inbox conversations ("inbox").
	Source             string     `gorm:"type:varchar(20);default:monitor;index" json:"source"`
	LastMessageAt      *time.Time `gorm:"index" json:"last_message_at,omitempty"`
	LastMessagePreview string     `gorm:"type:varchar(300)" json:"last_message_preview"`
	UnreadCount        int        `gorm:"default:0" json:"unread_count"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	WaNumber           WaNumber   `gorm:"foreignKey:WaNumberID" json:"-"`
}

type WaMessage struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	WaListenerID uint       `gorm:"index;not null" json:"wa_listener_id"`
	MessageID    string     `gorm:"column:message_id;type:varchar(100);uniqueIndex;not null" json:"message_id"`
	SenderJID    string     `gorm:"column:sender_jid;type:varchar(100);not null" json:"sender_jid"`
	SenderName   string     `gorm:"type:varchar(200)" json:"sender_name"`
	Content      string     `gorm:"type:longtext" json:"content"`
	MessageType  string     `gorm:"type:varchar(20);default:text" json:"message_type"`
	HasMedia     bool       `gorm:"default:false" json:"has_media"`
	FromMe       bool       `gorm:"column:from_me;default:false" json:"from_me"`
	ChatJID      string     `gorm:"column:chat_jid;type:varchar(100);index" json:"chat_jid"`
	// Status tracks outgoing message delivery: sent | delivered | read.
	Status       string     `gorm:"type:varchar(20)" json:"status,omitempty"`
	Timestamp    time.Time  `gorm:"index;not null" json:"timestamp"`
	CreatedAt    time.Time  `json:"created_at"`
	WaListener   WaListener `gorm:"foreignKey:WaListenerID" json:"-"`
	Media        []WaMedia  `gorm:"foreignKey:WaMessageID" json:"media,omitempty"`
}

type WaMedia struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	WaMessageID uint      `gorm:"index;not null" json:"wa_message_id"`
	FileName    string    `gorm:"type:varchar(255)" json:"file_name"`
	MimeType    string    `gorm:"type:varchar(100)" json:"mime_type"`
	FileSize    int64     `json:"file_size"`
	R2Key       string    `gorm:"column:r2_key;type:varchar(500)" json:"r2_key"`
	FileURL     string    `gorm:"column:file_url;type:varchar(500)" json:"file_url"`
	CreatedAt   time.Time `json:"created_at"`
	WaMessage   WaMessage `gorm:"foreignKey:WaMessageID" json:"-"`
}

type WaOutbox struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	WaNumberID  uint       `gorm:"index;not null" json:"wa_number_id"`
	TargetJID   string     `gorm:"column:target_jid;type:varchar(100);not null" json:"target_jid"`
	TargetName  string     `gorm:"type:varchar(200)" json:"target_name"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	MediaURL    string     `gorm:"column:media_url;type:varchar(500)" json:"media_url,omitempty"`
	Status      string     `gorm:"type:varchar(20);default:pending;index" json:"status"`
	RequestedBy string     `gorm:"type:varchar(20);default:agent" json:"requested_by"`
	Context     string     `gorm:"type:text" json:"context"`
	WaMessageID string     `gorm:"column:wa_message_id;type:varchar(100)" json:"wa_message_id,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	WaNumber    WaNumber   `gorm:"foreignKey:WaNumberID" json:"-"`
}
