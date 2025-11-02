package service

import (
	"github.com/arwen/im-server/internal/model"
	"github.com/arwen/im-server/internal/repository"
	"github.com/arwen/im-server/pkg/utils"
)

// MessageService 消息服务
type MessageService struct{}

// NewMessageService 创建消息服务
func NewMessageService() *MessageService {
	return &MessageService{}
}

// SaveMessage 保存消息
func (s *MessageService) SaveMessage(msg *model.Message) error {
	// 生成消息序列号
	seq, err := s.GenerateSeq(msg.SenderID)
	if err != nil {
		return err
	}
	msg.Seq = seq

	// 保存消息
	return repository.DB.Create(msg).Error
}

// GetMessageByID 根据ID获取消息
func (s *MessageService) GetMessageByID(messageID string) (*model.Message, error) {
	var msg model.Message
	err := repository.DB.Where("id = ?", messageID).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetConversationMessages 获取会话消息
func (s *MessageService) GetConversationMessages(conversationID string, limit, offset int) ([]*model.Message, error) {
	var messages []*model.Message
	err := repository.DB.Where("conversation_id = ? AND status != 4", conversationID).
		Order("seq DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

// GenerateSeq 生成消息序列号
func (s *MessageService) GenerateSeq(userID string) (int64, error) {
	var seq model.MessageSequence
	err := repository.DB.Where("user_id = ?", userID).First(&seq).Error
	
	if err != nil {
		// 不存在则创建
		seq = model.MessageSequence{
			ID:     utils.GenerateID(),
			UserID: userID,
			MaxSeq: 1,
		}
		if err := repository.DB.Create(&seq).Error; err != nil {
			return 0, err
		}
		return 1, nil
	}

	// 更新序列号
	newSeq := seq.MaxSeq + 1
	err = repository.DB.Model(&seq).Update("max_seq", newSeq).Error
	if err != nil {
		return 0, err
	}

	return newSeq, nil
}

// GetMaxSeq 获取用户最大序列号
func (s *MessageService) GetMaxSeq(userID string) (int64, error) {
	var seq model.MessageSequence
	err := repository.DB.Where("user_id = ?", userID).First(&seq).Error
	if err != nil {
		return 0, nil
	}
	return seq.MaxSeq, nil
}

// SyncMessages 同步消息
func (s *MessageService) SyncMessages(userID string, minSeq, maxSeq int64, limit int) ([]*model.Message, int64, bool, error) {
	var messages []*model.Message
	
	// 查询用户相关的消息（作为发送者或接收者）
	err := repository.DB.Where("(sender_id = ? OR receiver_id = ?) AND seq > ? AND status != 4", userID, userID, maxSeq).
		Order("seq ASC").
		Limit(limit).
		Find(&messages).Error
	
	if err != nil {
		return nil, 0, false, err
	}

	// 获取服务器最大序列号
	serverMaxSeq, err := s.GetMaxSeq(userID)
	if err != nil {
		return nil, 0, false, err
	}

	hasMore := len(messages) >= limit
	return messages, serverMaxSeq, hasMore, nil
}

// RevokeMessage 撤回消息
func (s *MessageService) RevokeMessage(messageID, userID string) error {
	// 查询消息
	var msg model.Message
	err := repository.DB.Where("id = ?", messageID).First(&msg).Error
	if err != nil {
		return err
	}

	// 检查权限（只有发送者可以撤回）
	if msg.SenderID != userID {
		return ErrPermissionDenied
	}

	// 更新状态
	return repository.DB.Model(&msg).Update("status", 4).Error
}

// SaveReadReceipt 保存已读回执
func (s *MessageService) SaveReadReceipt(messageID, conversationID, userID string, readTime int64) error {
	receipt := &model.MessageReadReceipt{
		ID:             utils.GenerateID(),
		MessageID:      messageID,
		ConversationID: conversationID,
		UserID:         userID,
		ReadTime:       readTime,
	}
	return repository.DB.Create(receipt).Error
}

