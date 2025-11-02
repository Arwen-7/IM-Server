package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// IDGenerator ID生成器
type IDGenerator struct {
	mu       sync.Mutex
	sequence uint32
	lastTime int64
}

var defaultIDGen = &IDGenerator{}

// GenerateID 生成唯一ID（类似雪花算法的简化版）
func GenerateID() string {
	defaultIDGen.mu.Lock()
	defer defaultIDGen.mu.Unlock()

	now := time.Now().UnixMilli()
	if now == defaultIDGen.lastTime {
		defaultIDGen.sequence++
		if defaultIDGen.sequence > 999 {
			// 等待下一毫秒
			for now <= defaultIDGen.lastTime {
				now = time.Now().UnixMilli()
			}
			defaultIDGen.sequence = 0
		}
	} else {
		defaultIDGen.sequence = 0
		defaultIDGen.lastTime = now
	}

	return fmt.Sprintf("%d%03d", now, defaultIDGen.sequence)
}

// GenerateUUID 生成UUID
func GenerateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return GenerateID()
	}
	return hex.EncodeToString(b)
}

// GenerateMessageID 生成消息ID
func GenerateMessageID() string {
	return GenerateID()
}

// GenerateSessionID 生成会话ID
func GenerateSessionID() string {
	return GenerateUUID()
}

