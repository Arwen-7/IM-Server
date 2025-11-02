package model

import (
	"time"
)

// Friend 好友关系模型
type Friend struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	UserID    string    `gorm:"index;size:64;not null" json:"user_id"`
	FriendID  string    `gorm:"index;size:64;not null" json:"friend_id"`
	Remark    string    `gorm:"size:128" json:"remark"` // 备注名
	Status    int       `gorm:"default:1" json:"status"` // 1: 正常, 2: 已删除, 3: 已拉黑
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (Friend) TableName() string {
	return "friends"
}

// FriendRequest 好友请求
type FriendRequest struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	FromUser  string    `gorm:"index;size:64;not null" json:"from_user"`
	ToUser    string    `gorm:"index;size:64;not null" json:"to_user"`
	Message   string    `gorm:"size:256" json:"message"`
	Status    int       `gorm:"default:0" json:"status"` // 0: 待处理, 1: 已同意, 2: 已拒绝
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (FriendRequest) TableName() string {
	return "friend_requests"
}

