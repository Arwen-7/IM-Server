package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"sort"
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

// GenerateMessageID 生成消息ID（服务端 serverMsgID，参考 OpenIM 实现）
func GenerateMessageID(senderID string) string {
	// 格式化时间戳：年月日时分秒
	t := time.Now().Format("20060102150405")
	// MD5(时间戳 + 发送者ID + 随机数)
	data := fmt.Sprintf("%s-%s-%d", t, senderID, mathrand.Int())
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GenerateSessionID 生成会话ID
func GenerateSessionID() string {
	return GenerateUUID()
}

// GetConversationID 生成标准会话ID（以客户端规则为准）
// sessionType: 1=单聊, 2=群聊, 3=聊天室, 4=系统消息
// userID1: 当前用户ID
// userID2: 对方用户ID 或 群组ID/聊天室ID
func GetConversationID(sessionType int, userID1, userID2 string) string {
	switch sessionType {
	case 1: // 单聊
		// 确保ID顺序一致（字母排序），保证双方生成的 conversationID 相同
		ids := []string{userID1, userID2}
		sort.Strings(ids)
		return fmt.Sprintf("single_%s_%s", ids[0], ids[1])
	case 2: // 群聊
		return fmt.Sprintf("group_%s", userID2) // userID2 作为 groupID
	case 3: // 聊天室
		return fmt.Sprintf("chatroom_%s", userID2) // userID2 作为 chatroomID
	case 4: // 系统消息
		return fmt.Sprintf("system_%s", userID2) // userID2 作为 systemID
	default:
		return ""
	}
}

