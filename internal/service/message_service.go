package service

import (
	"fmt"
	"log"

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
	// ✅ 基于会话ID生成序列号（同一会话内的消息序列连续）
	seq, err := s.GenerateSeq(msg.ConversationID)
	if err != nil {
		return err
	}
	msg.Seq = seq
	
	// ✅ 生成服务端消息ID（全局唯一，类似 OpenIM）
	msg.ServerMsgID = utils.GenerateMessageID(msg.SenderID)

	// 保存消息（会话内的 client_msg_id 唯一性由数据库索引保证）
	return repository.DB.Create(msg).Error
}

// GetMessageByClientMsgID 根据会话ID和客户端消息ID获取消息（用于幂等）
func (s *MessageService) GetMessageByClientMsgID(conversationID, clientMsgID string) (*model.Message, error) {
	var msg model.Message
	err := repository.DB.Where("conversation_id = ? AND client_msg_id = ?", conversationID, clientMsgID).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetMessageBySeq 根据会话ID和Seq获取消息
func (s *MessageService) GetMessageBySeq(conversationID string, seq int64) (*model.Message, error) {
	var msg model.Message
	err := repository.DB.Where("conversation_id = ? AND seq = ?", conversationID, seq).First(&msg).Error
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

// GenerateSeq 生成消息序列号（基于会话ID）
func (s *MessageService) GenerateSeq(conversationID string) (int64, error) {
	var seq model.MessageSequence
	err := repository.DB.Where("conversation_id = ?", conversationID).First(&seq).Error
	
	if err != nil {
		// 不存在则创建
		seq = model.MessageSequence{
			ID:             utils.GenerateID(),
			ConversationID: conversationID,
			MaxSeq:         1,
		}
		if err := repository.DB.Create(&seq).Error; err != nil {
			return 0, err
		}
		return 1, nil
	}

	// 更新序列号（会话内递增）
	newSeq := seq.MaxSeq + 1
	err = repository.DB.Model(&seq).Update("max_seq", newSeq).Error
	if err != nil {
		return 0, err
	}

	return newSeq, nil
}

// GetMaxSeq 获取会话的最大序列号
func (s *MessageService) GetMaxSeq(conversationID string) (int64, error) {
	var seq model.MessageSequence
	err := repository.DB.Where("conversation_id = ?", conversationID).First(&seq).Error
	if err != nil {
		return 0, nil // 不存在则返回 0
	}
	return seq.MaxSeq, nil
}

// GetUserConversationIDs 获取用户的所有会话ID（用于重装App场景）
func (s *MessageService) GetUserConversationIDs(userID string) ([]string, error) {
	var conversationIDs []string
	
	// 从消息表中查询用户相关的所有不同的 conversationID
	err := repository.DB.Model(&model.Message{}).
		Distinct("conversation_id").
		Where("(sender_id = ? OR receiver_id = ?) AND status != 4", userID, userID).
		Pluck("conversation_id", &conversationIDs).Error
	
	return conversationIDs, err
}

// GetConversationMaxSeqMap 获取多个会话的最大 seq（批量查询）
func (s *MessageService) GetConversationMaxSeqMap(conversationIDs []string) (map[string]int64, error) {
	if len(conversationIDs) == 0 {
		return map[string]int64{}, nil
	}
	
	var results []struct {
		ConversationID string
		MaxSeq         int64
	}
	
	err := repository.DB.Model(&model.MessageSequence{}).
		Select("conversation_id, max_seq").
		Where("conversation_id IN ?", conversationIDs).
		Find(&results).Error
	
	if err != nil {
		return nil, err
	}
	
	seqMap := make(map[string]int64)
	for _, r := range results {
		seqMap[r.ConversationID] = r.MaxSeq
	}
	
	return seqMap, nil
}

// SyncConversationMessages 增量同步单个会话消息（从 lastSeq 之后拉取）
func (s *MessageService) SyncConversationMessages(conversationID string, lastSeq int64, count int) ([]*model.Message, int64, bool, int64, error) {
	var messages []*model.Message
	
	// 查询该会话在 lastSeq 之后的消息
	err := repository.DB.Where("conversation_id = ? AND seq > ? AND status != 4", conversationID, lastSeq).
		Order("seq ASC").
		Limit(count).
		Find(&messages).Error
	
	if err != nil {
		return nil, 0, false, 0, err
	}

	// 获取该会话的服务器最大序列号
	serverMaxSeq, err := s.GetMaxSeq(conversationID)
	if err != nil {
		return nil, 0, false, 0, err
	}

	// 计算是否还有更多消息
	hasMore := len(messages) >= count
	
	// 计算总共需要同步的消息数量
	var totalCount int64
	repository.DB.Model(&model.Message{}).
		Where("conversation_id = ? AND seq > ? AND status != 4", conversationID, lastSeq).
		Count(&totalCount)

	return messages, serverMaxSeq, hasMore, totalCount, nil
}

// RevokeMessage 撤回消息（通过客户端消息ID）
func (s *MessageService) RevokeMessage(clientMsgID, userID string) error {
	// 查询消息
	var msg model.Message
	err := repository.DB.Where("client_msg_id = ?", clientMsgID).First(&msg).Error
	if err != nil {
		return err
	}

	// 检查权限（只有发送者可以撤回）
	if msg.SenderID != userID {
		return ErrPermissionDenied
	}

	// 更新状态为已撤回
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

// MarkMessagesAsRead 批量标记消息为已读
func (s *MessageService) MarkMessagesAsRead(conversationID string, messageIDs []string, userID string, readTime int64) error {
	// 1. 更新消息状态为已读（如果需要在 messages 表记录已读状态）
	// 注意：这里假设是单聊场景，群聊需要单独的已读表
	
	// 2. 批量保存已读回执记录
	for _, msgID := range messageIDs {
		receipt := &model.MessageReadReceipt{
			ID:             utils.GenerateID(),
			MessageID:      msgID,
			ConversationID: conversationID,
			UserID:         userID,
			ReadTime:       readTime,
		}
		if err := repository.DB.Create(receipt).Error; err != nil {
			return err
		}
	}
	
	return nil
}

// GetUnreadMessagesInConversation 获取会话中的未读消息ID列表
func (s *MessageService) GetUnreadMessagesInConversation(conversationID string, userID string) ([]string, error) {
	var messageIDs []string
	
	// 查询该会话中所有发给该用户且未被该用户读取的消息
	err := repository.DB.Table("messages").
		Select("messages.client_msg_id").
		Joins("LEFT JOIN message_read_receipts ON messages.client_msg_id = message_read_receipts.message_id AND message_read_receipts.user_id = ?", userID).
		Where("messages.conversation_id = ? AND messages.receiver_id = ? AND messages.status != 4 AND message_read_receipts.id IS NULL", conversationID, userID).
		Pluck("messages.client_msg_id", &messageIDs).Error
	
	return messageIDs, err
}

// CheckMessagesReadStatus 批量检查消息的已读状态
// 返回一个 map[messageID]isRead，用于同步时判断消息是否已读
func (s *MessageService) CheckMessagesReadStatus(messageIDs []string, userID string) (map[string]bool, error) {
	result := make(map[string]bool)
	
	// 初始化所有消息为未读
	for _, msgID := range messageIDs {
		result[msgID] = false
	}
	
	// 查询已读回执表
	var readReceipts []model.MessageReadReceipt
	err := repository.DB.Where("message_id IN ? AND user_id = ?", messageIDs, userID).
		Find(&readReceipts).Error
	if err != nil {
		return result, err
	}
	
	// 标记已读的消息
	for _, receipt := range readReceipts {
		result[receipt.MessageID] = true
	}
	
	return result, nil
}

// BatchSyncMessages 批量同步多个会话的消息（一次请求返回所有结果）
func (s *MessageService) BatchSyncMessages(userID string, conversationStates map[string]int64, maxCountPerConv int) ([]*BatchSyncResult, error) {
	if maxCountPerConv <= 0 {
		maxCountPerConv = 100
	}
	if maxCountPerConv > 500 {
		maxCountPerConv = 500
	}

	// 1. 获取用户的所有会话
	conversationIDs, err := s.GetUserConversationIDs(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user conversations: %w", err)
	}

	// 2. 对每个会话进行同步
	results := make([]*BatchSyncResult, 0, len(conversationIDs))
	
	for _, conversationID := range conversationIDs {
		// 获取客户端的 lastSeq（如果客户端没有提供，默认为0，表示全量同步）
		lastSeq := conversationStates[conversationID]
		
		// 同步该会话的消息
		messages, maxSeq, hasMore, _, err := s.SyncConversationMessages(conversationID, lastSeq, maxCountPerConv)
		if err != nil {
			log.Printf("⚠️ Failed to sync conversation %s: %v", conversationID, err)
			continue  // 单个会话失败不影响其他会话
		}
		
		// 计算本次同步到的 seq
		syncedSeq := lastSeq
		if len(messages) > 0 {
			syncedSeq = messages[len(messages)-1].Seq
		}
		
		// 如果有消息或者需要同步，加入结果
		if len(messages) > 0 || maxSeq > lastSeq {
			results = append(results, &BatchSyncResult{
				ConversationID: conversationID,
				Messages:       messages,
				MaxSeq:         maxSeq,
				SyncedSeq:      syncedSeq,
				HasMore:        hasMore,
			})
		}
	}
	
	return results, nil
}

// BatchSyncResult 批量同步的单个会话结果
type BatchSyncResult struct {
	ConversationID string
	Messages       []*model.Message
	MaxSeq         int64
	SyncedSeq      int64
	HasMore        bool
}

// SyncMessagesInRange 范围同步消息（用于补拉丢失的消息）
// 参数：
//   - conversationID: 会话ID
//   - startSeq: 起始 seq（包含）
//   - endSeq: 结束 seq（包含）
//   - count: 单次拉取数量限制（默认100，最大500）
// 返回：
//   - messages: 消息列表
//   - actualStartSeq: 实际返回的起始 seq
//   - actualEndSeq: 实际返回的结束 seq
//   - hasMore: 是否还有更多消息
func (s *MessageService) SyncMessagesInRange(conversationID string, startSeq, endSeq int64, count int) (messages []*model.Message, actualStartSeq, actualEndSeq int64, hasMore bool, err error) {
	// 参数校验
	if count <= 0 {
		count = 100
	}
	if count > 500 {
		count = 500
	}
	
	if startSeq > endSeq {
		return nil, 0, 0, false, fmt.Errorf("invalid seq range: startSeq (%d) > endSeq (%d)", startSeq, endSeq)
	}
	
	// 查询指定范围的消息
	err = repository.DB.
		Where("conversation_id = ? AND seq >= ? AND seq <= ?", conversationID, startSeq, endSeq).
		Order("seq ASC").
		Limit(count).
		Find(&messages).Error
	
	if err != nil {
		return nil, 0, 0, false, err
	}
	
	// 如果没有消息，返回
	if len(messages) == 0 {
		return messages, startSeq, startSeq - 1, false, nil
	}
	
	// 计算实际返回的范围
	actualStartSeq = messages[0].Seq
	actualEndSeq = messages[len(messages)-1].Seq
	
	// 判断是否还有更多消息（实际返回的结束 seq 小于请求的结束 seq）
	hasMore = actualEndSeq < endSeq
	
	log.Printf("✅ Range sync: conversation=%s, range=[%d,%d], returned=[%d,%d], count=%d, hasMore=%v", 
		conversationID, startSeq, endSeq, actualStartSeq, actualEndSeq, len(messages), hasMore)
	
	return messages, actualStartSeq, actualEndSeq, hasMore, nil
}

