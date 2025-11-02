package model

import (
	"time"
)

// Group 群组模型
type Group struct {
	ID          string    `gorm:"primaryKey;size:64" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Avatar      string    `gorm:"size:512" json:"avatar"`
	Description string    `gorm:"size:512" json:"description"`
	OwnerID     string    `gorm:"index;size:64;not null" json:"owner_id"`
	MaxMembers  int       `gorm:"default:500" json:"max_members"`
	Status      int       `gorm:"default:1" json:"status"` // 1: 正常, 2: 已解散
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 表名
func (Group) TableName() string {
	return "groups"
}

// GroupMember 群成员
type GroupMember struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	GroupID   string    `gorm:"index;size:64;not null" json:"group_id"`
	UserID    string    `gorm:"index;size:64;not null" json:"user_id"`
	Nickname  string    `gorm:"size:128" json:"nickname"` // 群昵称
	Role      int       `gorm:"default:3" json:"role"`    // 1: 群主, 2: 管理员, 3: 普通成员
	Status    int       `gorm:"default:1" json:"status"`  // 1: 正常, 2: 已退出, 3: 已被踢出
	JoinedAt  time.Time `json:"joined_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (GroupMember) TableName() string {
	return "group_members"
}

