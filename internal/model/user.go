package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Nickname  string    `gorm:"size:128" json:"nickname"`
	Avatar    string    `gorm:"size:512" json:"avatar"`
	Email     string    `gorm:"size:128" json:"email"`
	Phone     string    `gorm:"size:32" json:"phone"`
	Password  string    `gorm:"size:256;not null" json:"-"`
	Status    int       `gorm:"default:1" json:"status"` // 1: 正常, 2: 禁用
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}

// UserSession 用户会话
type UserSession struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	UserID     string    `gorm:"index;size:64;not null" json:"user_id"`
	Platform   string    `gorm:"size:32" json:"platform"`
	DeviceInfo string    `gorm:"size:512" json:"device_info"`
	Token      string    `gorm:"size:512" json:"token"`
	ExpireAt   time.Time `json:"expire_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 表名
func (UserSession) TableName() string {
	return "user_sessions"
}

// OnlineStatus 在线状态
type OnlineStatus struct {
	UserID     string    `gorm:"primaryKey;size:64" json:"user_id"`
	Status     int       `gorm:"default:0" json:"status"` // 0: 离线, 1: 在线
	Platform   string    `gorm:"size:32" json:"platform"`
	LastOnline time.Time `json:"last_online"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 表名
func (OnlineStatus) TableName() string {
	return "online_status"
}

