package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arwen/im-server/internal/repository"
)

// SessionInfo 会话信息
type SessionInfo struct {
	UserID     string    `json:"user_id"`
	Platform   string    `json:"platform"`
	DeviceInfo string    `json:"device_info"`
	ConnID     string    `json:"conn_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// SaveSession 保存会话
func SaveSession(sessionID string, info *SessionInfo, expire time.Duration) error {
	ctx := context.Background()
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("session:%s", sessionID)
	return repository.RedisClient.Set(ctx, key, data, expire).Err()
}

// GetSession 获取会话
func GetSession(sessionID string) (*SessionInfo, error) {
	ctx := context.Background()
	key := fmt.Sprintf("session:%s", sessionID)

	data, err := repository.RedisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var info SessionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// DeleteSession 删除会话
func DeleteSession(sessionID string) error {
	ctx := context.Background()
	key := fmt.Sprintf("session:%s", sessionID)
	return repository.RedisClient.Del(ctx, key).Err()
}

// SaveUserConnection 保存用户连接映射
func SaveUserConnection(userID, connID string) error {
	ctx := context.Background()
	key := fmt.Sprintf("user_conn:%s", userID)
	return repository.RedisClient.Set(ctx, key, connID, 0).Err()
}

// GetUserConnection 获取用户连接
func GetUserConnection(userID string) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("user_conn:%s", userID)
	return repository.RedisClient.Get(ctx, key).Result()
}

// DeleteUserConnection 删除用户连接
func DeleteUserConnection(userID string) error {
	ctx := context.Background()
	key := fmt.Sprintf("user_conn:%s", userID)
	return repository.RedisClient.Del(ctx, key).Err()
}

