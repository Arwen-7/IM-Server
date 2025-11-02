package model

import (
	"time"
)

// Conversation 会话模型
type Conversation struct {
	ID             string    `gorm:"primaryKey;size:64" json:"id"`
	Type           int       `gorm:"not null" json:"type"` // 1: 单聊, 2: 群聊
	UserID         string    `gorm:"index;size:64" json:"user_id"`
	TargetID       string    `gorm:"index;size:64" json:"target_id"` // 对方用户ID或群组ID
	LastMessageID  string    `gorm:"size:64" json:"last_message_id"`
	LastMessage    string    `gorm:"size:512" json:"last_message"`
	LastMessageAt  int64     `json:"last_message_at"`
	UnreadCount    int       `gorm:"default:0" json:"unread_count"`
	Status         int       `gorm:"default:1" json:"status"` // 1: 正常, 2: 已删除
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 表名
func (Conversation) TableName() string {
	return "conversations"
}

