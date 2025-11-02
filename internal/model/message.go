package model

import (
	"time"
)

// Message 消息模型
type Message struct {
	ID             string    `gorm:"primaryKey;size:64" json:"id"`
	ClientMsgID    string    `gorm:"index;size:64" json:"client_msg_id"`
	ConversationID string    `gorm:"index;size:64;not null" json:"conversation_id"`
	SenderID       string    `gorm:"index;size:64;not null" json:"sender_id"`
	ReceiverID     string    `gorm:"index;size:64" json:"receiver_id"`
	GroupID        string    `gorm:"index;size:64" json:"group_id"`
	MessageType    int       `gorm:"not null" json:"message_type"` // 1: 文本, 2: 图片, 3: 语音, 4: 视频, 5: 文件
	Content        string    `gorm:"type:text" json:"content"`
	Seq            int64     `gorm:"index;not null" json:"seq"`
	Status         int       `gorm:"default:0" json:"status"` // 0: 未发送, 1: 已发送, 2: 已送达, 3: 已读, 4: 已撤回
	SendTime       int64     `json:"send_time"`
	ServerTime     int64     `json:"server_time"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 表名
func (Message) TableName() string {
	return "messages"
}

// MessageSequence 消息序列号
type MessageSequence struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	UserID    string    `gorm:"uniqueIndex;size:64;not null" json:"user_id"`
	MaxSeq    int64     `gorm:"default:0" json:"max_seq"`
	UpdatedAt time.Time `json:"updated_at"`
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

