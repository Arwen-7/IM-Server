package model

import (
	"time"
)

// Message 消息模型
type Message struct {
	ConversationID string    `gorm:"primaryKey;size:64;not null" json:"conversation_id"` // 会话ID（复合主键1）
	Seq            int64     `gorm:"primaryKey;autoIncrement:false" json:"seq"`           // 消息序列号（复合主键2，会话内递增）
	ServerMsgID    string    `gorm:"uniqueIndex;size:64;not null" json:"server_msg_id"`  // 服务端消息ID（服务端生成，全局唯一）
	ClientMsgID    string    `gorm:"size:64;not null" json:"client_msg_id"`              // 客户端消息ID（会话内幂等检查）
	SenderID       string    `gorm:"index;size:64;not null" json:"sender_id"`
	ReceiverID     string    `gorm:"index;size:64" json:"receiver_id"`
	GroupID        string    `gorm:"index;size:64" json:"group_id"`
	MessageType    int       `gorm:"not null" json:"message_type"` // 1: 文本, 2: 图片, 3: 语音, 4: 视频, 5: 文件
	Content        string    `gorm:"type:text" json:"content"`
	Status         int       `gorm:"default:1" json:"status"` // 1: 已发送, 2: 已送达, 3: 已读, 4: 已撤回
	SendTime       int64     `json:"send_time"`
	ServerTime     int64     `json:"server_time"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 表名
func (Message) TableName() string {
	return "messages"
}

// MessageSequence 消息序列号（每个会话维护独立的序列）
type MessageSequence struct {
	ID             string    `gorm:"primaryKey;size:64" json:"id"`
	ConversationID string    `gorm:"uniqueIndex;size:64;not null" json:"conversation_id"` // 会话ID
	MaxSeq         int64     `gorm:"default:0" json:"max_seq"`                             // 当前最大序列号（会话内）
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 表名
func (MessageSequence) TableName() string {
	return "message_sequences"
}

// MessageReadReceipt 消息已读回执
type MessageReadReceipt struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	MessageID      string    `gorm:"index;size:64;not null" json:"message_id"`
	ConversationID string    `gorm:"index;size:64;not null" json:"conversation_id"`
	UserID         string    `gorm:"index;size:64;not null" json:"user_id"`
	ReadTime       int64     `json:"read_time"`
	CreatedAt      time.Time `json:"created_at"`
}

// TableName 表名
func (MessageReadReceipt) TableName() string {
	return "message_read_receipts"
}

