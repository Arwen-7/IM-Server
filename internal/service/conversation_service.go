package service

import (
	"github.com/arwen/im-server/internal/model"
	"github.com/arwen/im-server/internal/repository"
	"github.com/arwen/im-server/pkg/utils"
)

// ConversationService 会话服务
type ConversationService struct{}

// NewConversationService 创建会话服务
func NewConversationService() *ConversationService {
	return &ConversationService{}
}

// GetOrCreateConversation 获取或创建会话
func (s *ConversationService) GetOrCreateConversation(userID, targetID string, convType int) (*model.Conversation, error) {
	var conv model.Conversation
	
	// 查询已存在的会话
	err := repository.DB.Where("user_id = ? AND target_id = ? AND type = ?", userID, targetID, convType).
		First(&conv).Error
	
	if err == nil {
		return &conv, nil
	}

	// 创建新会话
	conv = model.Conversation{
		ID:       utils.GenerateID(),
		Type:     convType,
		UserID:   userID,
		TargetID: targetID,
		Status:   1,
	}

	if err := repository.DB.Create(&conv).Error; err != nil {
		return nil, err
	}

	return &conv, nil
}

// UpdateLastMessage 更新会话最后一条消息
func (s *ConversationService) UpdateLastMessage(conversationID, messageID, lastMessage string, messageTime int64) error {
	return repository.DB.Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"last_message_id": messageID,
			"last_message":    lastMessage,
			"last_message_at": messageTime,
		}).Error
}

// IncrementUnreadCount 增加未读数
func (s *ConversationService) IncrementUnreadCount(conversationID string) error {
	return repository.DB.Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Update("unread_count", repository.DB.Raw("unread_count + ?", 1)).Error
}

// ClearUnreadCount 清除未读数
func (s *ConversationService) ClearUnreadCount(conversationID string) error {
	return repository.DB.Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Update("unread_count", 0).Error
}

// GetUserConversations 获取用户会话列表
func (s *ConversationService) GetUserConversations(userID string) ([]*model.Conversation, error) {
	var conversations []*model.Conversation
	err := repository.DB.Where("user_id = ? AND status = 1", userID).
		Order("last_message_at DESC").
		Find(&conversations).Error
	return conversations, err
}

