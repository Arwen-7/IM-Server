package handler

import (
	"github.com/arwen/im-server/internal/model"
	"github.com/arwen/im-server/internal/protocol"
	"github.com/arwen/im-server/internal/service"
	"github.com/arwen/im-server/internal/transport"
	"github.com/arwen/im-server/pkg/logger"
	"github.com/arwen/im-server/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	connManager  *transport.ConnectionManager
	userService  *service.UserService
	msgService   *service.MessageService
	convService  *service.ConversationService
	groupService *service.GroupService
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(
	connManager *transport.ConnectionManager,
	userService *service.UserService,
	msgService *service.MessageService,
	convService *service.ConversationService,
	groupService *service.GroupService,
) *MessageHandler {
	return &MessageHandler{
		connManager:  connManager,
		userService:  userService,
		msgService:   msgService,
		convService:  convService,
		groupService: groupService,
	}
}

// HandleTCPPacket 处理 TCP 数据包
func (h *MessageHandler) HandleTCPPacket(conn transport.Connection, packet *protocol.Packet) error {
	// 转换为 WebSocketMessage 格式（统一处理）
	wsMsg := &protocol.WebSocketMessage{
		Command:  protocol.CommandType(packet.Header.Command),
		Sequence: packet.Header.Sequence,
		Body:     packet.Body,
	}

	logger.Debug("Received TCP message",
		zap.String("conn_id", conn.GetID()),
		zap.Uint16("command", packet.Header.Command),
		zap.Uint32("sequence", packet.Header.Sequence))

	return h.handleMessage(conn, wsMsg)
}

// HandleMessage 处理 WebSocket 消息
func (h *MessageHandler) HandleMessage(conn transport.Connection, data []byte) error {
	// WebSocket 连接：解析 WebSocket 消息格式
	wsMsg, err := protocol.UnmarshalWebSocketMessage(data)
	if err != nil {
		logger.Error("Failed to unmarshal WebSocket message", zap.Error(err))
		return err
	}

	logger.Debug("Received WebSocket message",
		zap.String("conn_id", conn.GetID()),
		zap.String("command", wsMsg.Command.String()),
		zap.Uint32("sequence", wsMsg.Sequence))

	return h.handleMessage(conn, wsMsg)
}

// handleMessage 统一处理消息（TCP 和 WebSocket 共用）
func (h *MessageHandler) handleMessage(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {

	// 根据命令类型处理
	switch wsMsg.Command {
	case protocol.CMD_AUTH_REQ:
		return h.handleAuth(conn, wsMsg)
	case protocol.CMD_HEARTBEAT_REQ:
		return h.handleHeartbeat(conn, wsMsg)
	case protocol.CMD_SEND_MSG_REQ:
		return h.handleSendMessage(conn, wsMsg)
	case protocol.CMD_MSG_ACK:
		return h.handleMessageAck(conn, wsMsg)
	case protocol.CommandType_CMD_BATCH_SYNC_REQ:
		return h.handleBatchSync(conn, wsMsg)
	case protocol.CommandType_CMD_SYNC_RANGE_REQ:
		return h.handleSyncRange(conn, wsMsg)
	case protocol.CommandType_CMD_READ_RECEIPT_REQ:
		return h.handleReadReceipt(conn, wsMsg)
	case protocol.CommandType_CMD_TYPING_STATUS_REQ:
		return h.handleTypingStatus(conn, wsMsg)
	case protocol.CMD_REVOKE_MSG_REQ:
		return h.handleRevokeMessage(conn, wsMsg)
	default:
		logger.Warn("Unknown command", zap.String("command", wsMsg.Command.String()))
	}

	return nil
}

// handleAuth 处理认证
func (h *MessageHandler) handleAuth(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	var req protocol.AuthRequest
	if err := protocol.Unmarshal(wsMsg.Body, &req); err != nil {
		return err
	}

	logger.Info("Auth request", zap.String("user_id", req.UserId), zap.String("platform", req.Platform))

	// 验证Token（开发模式：允许 demo_token_xxx 格式）
	var userID string
	if len(req.Token) > 11 && req.Token[:11] == "demo_token_" {
		// 开发模式：从 token 中提取 userID
		userID = req.Token[11:]
		logger.Info("Dev mode: using demo token", zap.String("user_id", userID))
	} else {
		// 生产模式：验证 JWT token
		claims, err := h.userService.ValidateToken(req.Token)
		if err != nil {
			// 认证失败
			resp := &protocol.AuthResponse{
				ErrorCode: protocol.ERR_AUTH_FAILED,
				ErrorMsg:  "Invalid token",
			}
			return h.sendResponse(conn, protocol.CommandType_CMD_AUTH_RSP, wsMsg.Sequence, resp)
		}
		userID = claims.UserID
	}

	// 绑定用户连接
	if err := h.connManager.BindUser(conn.GetID(), userID); err != nil {
		resp := &protocol.AuthResponse{
			ErrorCode: protocol.ERR_UNKNOWN,
			ErrorMsg:  "Failed to bind connection",
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_AUTH_RSP, wsMsg.Sequence, resp)
	}

	// 获取最大序列号
	maxSeq, _ := h.msgService.GetMaxSeq(userID)

	// 认证成功
	resp := &protocol.AuthResponse{
		ErrorCode: protocol.ERR_SUCCESS,
		ErrorMsg:  "Success",
		MaxSeq:    maxSeq,
	}

	logger.Info("Auth success", zap.String("user_id", userID))
	return h.sendResponse(conn, protocol.CMD_AUTH_RSP, wsMsg.Sequence, resp)
}

// handleHeartbeat 处理心跳
func (h *MessageHandler) handleHeartbeat(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	resp := &protocol.HeartbeatResponse{
		ServerTime: utils.GetCurrentMillis(),
	}
	return h.sendResponse(conn, protocol.CMD_HEARTBEAT_RSP, wsMsg.Sequence, resp)
}

// handleSendMessage 处理发送消息
func (h *MessageHandler) handleSendMessage(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	var req protocol.SendMessageRequest
	if err := protocol.Unmarshal(wsMsg.Body, &req); err != nil {
		return err
	}

	userID := conn.GetUserID()
	if userID == "" {
		resp := &protocol.SendMessageResponse{
			ErrorCode: protocol.ERR_AUTH_FAILED,
			ErrorMsg:  "Not authenticated",
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_SEND_MSG_RSP, wsMsg.Sequence, resp)
	}

	now := utils.GetCurrentMillis()

	// ✅ 通过 .Message 访问 MessageInfo 字段
	msgInfo := req.Message

	// ✅ conversationID 处理：优先使用客户端传的，没传才服务端生成
	conversationID := msgInfo.ConversationId
	if conversationID == "" {
		// 客户端没传，服务端自动生成标准的 conversationID
		var sessionType int
		if msgInfo.ReceiverId != "" {
			// 单聊
			sessionType = 1
			conversationID = utils.GetConversationID(sessionType, userID, msgInfo.ReceiverId)
		} else if msgInfo.GroupId != "" {
			// 群聊
			sessionType = 2
			conversationID = utils.GetConversationID(sessionType, userID, msgInfo.GroupId)
		} else {
			logger.Error("Invalid message: no receiver or group")
			resp := &protocol.SendMessageResponse{
				ErrorCode: protocol.ERR_INVALID_PARAM,
				ErrorMsg:  "No receiver or group specified",
			}
			return h.sendResponse(conn, protocol.CommandType_CMD_SEND_MSG_RSP, wsMsg.Sequence, resp)
		}
	}

	// 创建消息
	msg := &model.Message{
		ClientMsgID:    msgInfo.ClientMsgId,
		ConversationID: conversationID, // ✅ 优先使用客户端传的，否则服务端生成
		SenderID:       userID,
		ReceiverID:     msgInfo.ReceiverId,
		GroupID:        msgInfo.GroupId,
		MessageType:    int(msgInfo.MessageType),
		Content:        string(msgInfo.Content),
		SendTime:       msgInfo.SendTime, // ✅ 统一使用 sendTime
		ServerTime:     now,
		Status:         1, // 已发送
		// Seq 由 SaveMessage 内部分配
	}

	// 保存消息（会自动分配 Seq）
	if err := h.msgService.SaveMessage(msg); err != nil {
		logger.Error("Failed to save message", zap.Error(err))
		resp := &protocol.SendMessageResponse{
			ErrorCode: protocol.ERR_UNKNOWN,
			ErrorMsg:  "Failed to save message",
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_SEND_MSG_RSP, wsMsg.Sequence, resp)
	}

	// 更新会话（使用服务端生成的 conversationID）
	h.convService.UpdateLastMessage(conversationID, msg.ClientMsgID, string(msgInfo.Content), now)

	// 发送响应（返回服务端生成的 ID 和 Seq）
	resp := &protocol.SendMessageResponse{
		ErrorCode:   protocol.ERR_SUCCESS,
		ErrorMsg:    "Success",
		ServerMsgId: msg.ServerMsgID, // ✅ 服务端生成的消息 ID（全局唯一）
		ClientMsgId: msg.ClientMsgID, // ✅ 客户端消息 ID（用于匹配本地消息）
		Seq:         msg.Seq,         // ✅ 会话内的序列号（全局有序）
		ServerTime:  now,
	}
	if err := h.sendResponse(conn, protocol.CMD_SEND_MSG_RSP, wsMsg.Sequence, resp); err != nil {
		return err
	}

	// 推送给接收者
	if msgInfo.ReceiverId != "" {
		// 单聊消息：推送给接收者
		h.pushMessageToUser(msgInfo.ReceiverId, msg)
		logger.Info("Message sent (single chat)",
			zap.String("server_msg_id", msg.ServerMsgID),
			zap.String("client_msg_id", msg.ClientMsgID),
			zap.Int64("seq", msg.Seq),
			zap.String("conversation_id", msg.ConversationID),
			zap.String("sender", userID),
			zap.String("receiver", msgInfo.ReceiverId))
	} else if msgInfo.GroupId != "" {
		// 群聊消息：推送给群组所有成员（除了发送者）
		h.pushMessageToGroup(msgInfo.GroupId, userID, msg)
		logger.Info("Message sent (group chat)",
			zap.String("server_msg_id", msg.ServerMsgID),
			zap.String("client_msg_id", msg.ClientMsgID),
			zap.Int64("seq", msg.Seq),
			zap.String("conversation_id", msg.ConversationID),
			zap.String("sender", userID),
			zap.String("group_id", msgInfo.GroupId))
	}

	return nil
}

// handleSync 处理增量同步
// handleGetConversations 处理获取会话列表请求
// handleBatchSync 处理批量同步（一次请求同步所有会话）
func (h *MessageHandler) handleBatchSync(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	var req protocol.BatchSyncRequest
	if err := protocol.Unmarshal(wsMsg.Body, &req); err != nil {
		return err
	}

	userID := conn.GetUserID()
	if userID == "" {
		resp := &protocol.BatchSyncResponse{
			ErrorCode: protocol.ErrorCode_ERR_AUTH_FAILED,
			ErrorMsg:  "Not authenticated",
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_BATCH_SYNC_RSP, wsMsg.Sequence, resp)
	}

	maxCountPerConv := int(req.MaxCountPerConversation)
	if maxCountPerConv <= 0 {
		maxCountPerConv = 100
	}

	logger.Info("Batch sync request",
		zap.String("user_id", userID),
		zap.Int("conversation_count", len(req.ConversationStates)),
		zap.Int("max_count_per_conv", maxCountPerConv))

	// 转换客户端状态为 map
	conversationStates := make(map[string]int64)
	for _, state := range req.ConversationStates {
		conversationStates[state.ConversationId] = state.LastSeq
	}

	// 批量同步
	results, err := h.msgService.BatchSyncMessages(userID, conversationStates, maxCountPerConv)
	if err != nil {
		logger.Error("Failed to batch sync messages", zap.Error(err))
		resp := &protocol.BatchSyncResponse{
			ErrorCode: protocol.ErrorCode_ERR_UNKNOWN,
			ErrorMsg:  "Failed to sync messages",
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_BATCH_SYNC_RSP, wsMsg.Sequence, resp)
	}

	// 转换结果
	var conversationMessagesList []*protocol.ConversationMessages
	totalMessageCount := 0

	for _, result := range results {
		// 收集消息 ClientMsgID 用于批量查询已读状态
		messageIDs := make([]string, len(result.Messages))
		for i, msg := range result.Messages {
			messageIDs[i] = msg.ClientMsgID
		}

		// 批量查询已读状态
		readStatusMap, err := h.msgService.CheckMessagesReadStatus(messageIDs, userID)
		if err != nil {
			logger.Error("Failed to check read status", zap.Error(err))
			readStatusMap = make(map[string]bool)
		}

		// 转换消息
		var messageInfoList []*protocol.MessageInfo
		for _, msg := range result.Messages {
			isRead := readStatusMap[msg.ClientMsgID]
			if msg.SenderID == userID {
				isRead = true
			}

			// 推断会话类型
			var conversationType int32
			if msg.GroupID != "" {
				conversationType = 2 // 群聊
			} else {
				conversationType = 1 // 单聊
			}

			msgInfo := &protocol.MessageInfo{
				ServerMsgId:      msg.ServerMsgID,
				ClientMsgId:      msg.ClientMsgID,
				ConversationId:   msg.ConversationID,
				ConversationType: conversationType, // ✅ 添加 ConversationType
				SenderId:         msg.SenderID,
				ReceiverId:       msg.ReceiverID,
				GroupId:          msg.GroupID,
				Seq:              msg.Seq,
				MessageType:      int32(msg.MessageType),
				Content:          []byte(msg.Content),
				SendTime:         msg.SendTime,
				ServerTime:       msg.ServerTime,
				CreateTime:       msg.SendTime,
				Status:           int32(msg.Status),
				IsRead:           isRead,
			}
			messageInfoList = append(messageInfoList, msgInfo)
		}

		conversationMessagesList = append(conversationMessagesList, &protocol.ConversationMessages{
			ConversationId: result.ConversationID,
			Messages:       messageInfoList,
			MaxSeq:         result.MaxSeq,
			SyncedSeq:      result.SyncedSeq,
			HasMore:        result.HasMore,
		})

		totalMessageCount += len(messageInfoList)
	}

	resp := &protocol.BatchSyncResponse{
		ErrorCode:            protocol.ErrorCode_ERR_SUCCESS,
		ErrorMsg:             "Success",
		ConversationMessages: conversationMessagesList,
		ServerTime:           utils.GetCurrentMillis(),
		TotalMessageCount:    int32(totalMessageCount),
	}

	logger.Info("Batch sync response",
		zap.String("user_id", userID),
		zap.Int("conversation_count", len(conversationMessagesList)),
		zap.Int("total_message_count", totalMessageCount))

	return h.sendResponse(conn, protocol.CommandType_CMD_BATCH_SYNC_RSP, wsMsg.Sequence, resp)
}

// handleMessageAck 处理消息ACK
func (h *MessageHandler) handleMessageAck(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	var req protocol.MessageAck
	if err := protocol.Unmarshal(wsMsg.Body, &req); err != nil {
		return err
	}

	logger.Debug("Message ACK", zap.String("msg_id", req.ServerMsgId), zap.Int64("seq", req.Seq))
	return nil
}

// handleReadReceipt 处理已读回执
func (h *MessageHandler) handleReadReceipt(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	var req protocol.ReadReceiptRequest
	if err := protocol.Unmarshal(wsMsg.Body, &req); err != nil {
		return err
	}

	userID := conn.GetUserID()
	if userID == "" {
		resp := &protocol.ReadReceiptResponse{
			ErrorCode: protocol.ErrorCode_ERR_AUTH_FAILED,
			ErrorMsg:  "Not authenticated",
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_READ_RECEIPT_RSP, wsMsg.Sequence, resp)
	}

	now := utils.GetCurrentMillis()

	// ✅ 如果 serverMsgIds 为空，说明要标记该会话所有未读消息
	messageIDs := req.ServerMsgIds
	if len(messageIDs) == 0 {
		// 查询该会话的所有未读消息
		unreadMessageIDs, err := h.msgService.GetUnreadMessagesInConversation(req.ConversationId, userID)
		if err != nil {
			logger.Error("Failed to get unread messages", zap.Error(err))
			resp := &protocol.ReadReceiptResponse{
				ErrorCode: protocol.ErrorCode_ERR_UNKNOWN,
				ErrorMsg:  "Failed to get unread messages",
			}
			return h.sendResponse(conn, protocol.CommandType_CMD_READ_RECEIPT_RSP, wsMsg.Sequence, resp)
		}
		messageIDs = unreadMessageIDs
		logger.Debug("Mark all unread messages in conversation",
			zap.String("conversation_id", req.ConversationId),
			zap.Int("count", len(messageIDs)))
	}

	// 批量标记消息为已读
	if len(messageIDs) > 0 {
		err := h.msgService.MarkMessagesAsRead(req.ConversationId, messageIDs, userID, now)
		if err != nil {
			logger.Error("Failed to mark messages as read", zap.Error(err))
			resp := &protocol.ReadReceiptResponse{
				ErrorCode: protocol.ErrorCode_ERR_UNKNOWN,
				ErrorMsg:  "Failed to mark messages as read",
			}
			return h.sendResponse(conn, protocol.CommandType_CMD_READ_RECEIPT_RSP, wsMsg.Sequence, resp)
		}
	}

	// 发送响应
	resp := &protocol.ReadReceiptResponse{
		ErrorCode: protocol.ErrorCode_ERR_SUCCESS,
		ErrorMsg:  "Success",
	}
	h.sendResponse(conn, protocol.CommandType_CMD_READ_RECEIPT_RSP, wsMsg.Sequence, resp)

	// 推送已读回执给会话中的其他用户（多端同步）
	// ✅ 使用实际标记的 messageIDs（可能是从数据库查询出来的）
	if len(messageIDs) > 0 {
		go h.pushReadReceiptToOthers(req.ConversationId, messageIDs, userID, now)
	}

	logger.Info("Read receipt processed",
		zap.String("user_id", userID),
		zap.String("conversation_id", req.ConversationId),
		zap.Int("message_count", len(messageIDs)))

	return nil
}

// pushReadReceiptToOthers 推送已读回执给会话中的其他在线用户
func (h *MessageHandler) pushReadReceiptToOthers(conversationID string, messageIDs []string, readerUserID string, readTime int64) {
	// 创建推送消息
	push := &protocol.ReadReceiptPush{
		ServerMsgIds:   messageIDs, // ✅ 使用 ServerMsgIds
		ConversationId: conversationID,
		UserId:         readerUserID,
		ReadTime:       readTime,
	}

	pushData, err := protocol.Marshal(push)
	if err != nil {
		logger.Error("Failed to marshal read receipt push", zap.Error(err))
		return
	}

	// 获取会话中消息的发送者（需要通知的用户）
	// 这里简化处理：从第一条消息获取对方用户ID
	if len(messageIDs) == 0 {
		return
	}

	msg, err := h.msgService.GetMessageByClientMsgID(conversationID, messageIDs[0])
	if err != nil {
		logger.Error("Failed to get message", zap.Error(err))
		return
	}

	// 确定要推送的目标用户（消息发送者）
	targetUserID := msg.SenderID
	if targetUserID == readerUserID {
		// 如果读者是发送者，说明是自己的消息，不需要推送
		return
	}

	// 推送给目标用户的所有在线设备
	h.pushToUser(targetUserID, protocol.CommandType_CMD_READ_RECEIPT_PUSH, pushData)

	logger.Debug("Read receipt pushed",
		zap.String("to_user", targetUserID),
		zap.String("reader", readerUserID),
		zap.Int("message_count", len(messageIDs)))
}

// handleTypingStatus 处理输入状态
func (h *MessageHandler) handleTypingStatus(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	var req protocol.TypingStatusRequest
	if err := protocol.Unmarshal(wsMsg.Body, &req); err != nil {
		return err
	}

	userID := conn.GetUserID()
	if userID == "" {
		return nil
	}

	// TODO: 推送给会话中的其他用户
	logger.Debug("Typing status", zap.String("user_id", userID), zap.String("conversation", req.ConversationId), zap.Int32("status", req.Status))
	return nil
}

// handleRevokeMessage 处理撤回消息
func (h *MessageHandler) handleRevokeMessage(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	var req protocol.RevokeMessageRequest
	if err := protocol.Unmarshal(wsMsg.Body, &req); err != nil {
		return err
	}

	userID := conn.GetUserID()
	if userID == "" {
		resp := &protocol.RevokeMessageResponse{
			ErrorCode: protocol.ERR_AUTH_FAILED,
			ErrorMsg:  "Not authenticated",
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_REVOKE_MSG_RSP, wsMsg.Sequence, resp)
	}

	// 撤回消息
	if err := h.msgService.RevokeMessage(req.ServerMsgId, userID); err != nil {
		resp := &protocol.RevokeMessageResponse{
			ErrorCode: protocol.ERR_PERMISSION_DENIED,
			ErrorMsg:  err.Error(),
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_REVOKE_MSG_RSP, wsMsg.Sequence, resp)
	}

	resp := &protocol.RevokeMessageResponse{
		ErrorCode: protocol.ERR_SUCCESS,
		ErrorMsg:  "Success",
	}

	logger.Info("Message revoked", zap.String("msg_id", req.ServerMsgId), zap.String("user_id", userID))
	return h.sendResponse(conn, protocol.CMD_REVOKE_MSG_RSP, wsMsg.Sequence, resp)
}

// handleSyncRange 处理范围同步请求（补拉丢失消息）
func (h *MessageHandler) handleSyncRange(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	var req protocol.SyncRangeRequest
	if err := protocol.Unmarshal(wsMsg.Body, &req); err != nil {
		return err
	}

	userID := conn.GetUserID()
	if userID == "" {
		resp := &protocol.SyncRangeResponse{
			ErrorCode: protocol.ErrorCode_ERR_AUTH_FAILED,
			ErrorMsg:  "Not authenticated",
			RequestId: req.RequestId,
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_SYNC_RANGE_RSP, wsMsg.Sequence, resp)
	}

	logger.Info("Range sync request",
		zap.String("user_id", userID),
		zap.String("request_id", req.RequestId),
		zap.String("conversation_id", req.ConversationId),
		zap.Int64("start_seq", req.StartSeq),
		zap.Int64("end_seq", req.EndSeq),
		zap.Int32("count", req.Count))

	// 调用服务层进行范围同步
	messages, actualStartSeq, actualEndSeq, hasMore, err := h.msgService.SyncMessagesInRange(
		req.ConversationId,
		req.StartSeq,
		req.EndSeq,
		int(req.Count),
	)
	if err != nil {
		logger.Error("Failed to perform range sync", zap.Error(err))
		resp := &protocol.SyncRangeResponse{
			ErrorCode: protocol.ErrorCode_ERR_UNKNOWN,
			ErrorMsg:  "Failed to perform range sync",
			RequestId: req.RequestId,
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_SYNC_RANGE_RSP, wsMsg.Sequence, resp)
	}

	// 转换为 Protocol MessageInfo
	var messageInfoList []*protocol.MessageInfo
	messageIDs := make([]string, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.ClientMsgID
	}

	// 批量查询已读状态
	readStatusMap, err := h.msgService.CheckMessagesReadStatus(messageIDs, userID)
	if err != nil {
		logger.Error("Failed to check read status in range sync", zap.Error(err))
		readStatusMap = make(map[string]bool) // 失败时使用空 map，默认为未读
	}

	for _, msg := range messages {
		isRead := readStatusMap[msg.ClientMsgID]
		if msg.SenderID == userID {
			isRead = true
		}

		// 推断会话类型
		var conversationType int32
		if msg.GroupID != "" {
			conversationType = 2 // 群聊
		} else {
			conversationType = 1 // 单聊
		}

		msgInfo := &protocol.MessageInfo{
			ServerMsgId:      msg.ServerMsgID,
			ClientMsgId:      msg.ClientMsgID,
			ConversationId:   msg.ConversationID,
			ConversationType: conversationType, // ✅ 添加 ConversationType
			SenderId:         msg.SenderID,
			ReceiverId:       msg.ReceiverID,
			GroupId:          msg.GroupID,
			MessageType:      int32(msg.MessageType),
			Content:          []byte(msg.Content),
			SendTime:         msg.SendTime,
			ServerTime:       msg.ServerTime,
			CreateTime:       msg.SendTime,
			Status:           int32(msg.Status),
			IsRead:           isRead,
			Seq:              msg.Seq, // ✅ 包含 seq 字段
		}
		messageInfoList = append(messageInfoList, msgInfo)
	}

	resp := &protocol.SyncRangeResponse{
		ErrorCode:      protocol.ErrorCode_ERR_SUCCESS,
		ErrorMsg:       "Success",
		RequestId:      req.RequestId,
		ConversationId: req.ConversationId,
		Messages:       messageInfoList,
		StartSeq:       actualStartSeq,
		EndSeq:         actualEndSeq,
		HasMore:        hasMore,
	}

	logger.Info("Range sync response",
		zap.String("user_id", userID),
		zap.String("request_id", req.RequestId),
		zap.Int("message_count", len(messageInfoList)),
		zap.Int64("start_seq", actualStartSeq),
		zap.Int64("end_seq", actualEndSeq),
		zap.Bool("has_more", hasMore))

	return h.sendResponse(conn, protocol.CommandType_CMD_SYNC_RANGE_RSP, wsMsg.Sequence, resp)
}

// sendResponse 发送响应
func (h *MessageHandler) sendResponse(conn transport.Connection, command protocol.CommandType, sequence uint32, message proto.Message) error {
	body, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	var data []byte

	if conn.GetType() == transport.ConnectionTypeTCP {
		// TCP 连接：使用自定义协议包格式
		data = protocol.EncodePacket(uint16(command), sequence, body)

		logger.Debug("Sending TCP response",
			zap.String("conn_id", conn.GetID()),
			zap.Uint16("command", uint16(command)),
			zap.Uint32("sequence", sequence),
			zap.Int("body_len", len(body)))

	} else {
		// WebSocket 连接：使用 WebSocket 消息格式
		wsMsg := &protocol.WebSocketMessage{
			Command:   command,
			Sequence:  sequence,
			Body:      body,
			Timestamp: utils.GetCurrentMillis(),
		}

		data, err = protocol.MarshalWebSocketMessage(wsMsg)
		if err != nil {
			return err
		}

		logger.Debug("Sending WebSocket response",
			zap.String("conn_id", conn.GetID()),
			zap.String("command", command.String()),
			zap.Uint32("sequence", sequence),
			zap.Int("body_len", len(body)))
	}

	return conn.Send(data)
}

// pushMessageToUser 推送消息给用户（✅ 使用 MessageInfo 结构）
func (h *MessageHandler) pushMessageToUser(userID string, msg *model.Message) {
	// 推断会话类型
	var conversationType int32
	if msg.GroupID != "" {
		conversationType = 2 // 群聊
	} else {
		conversationType = 1 // 单聊
	}

	pushMsg := &protocol.PushMessage{
		Message: &protocol.MessageInfo{ // ✅ 通过 Message 字段包装
			ServerMsgId:      msg.ServerMsgID, // ✅ 使用服务端消息ID
			ClientMsgId:      msg.ClientMsgID, // ✅ 客户端消息ID
			ConversationId:   msg.ConversationID,
			ConversationType: conversationType, // ✅ 添加 ConversationType
			SenderId:         msg.SenderID,
			ReceiverId:       msg.ReceiverID,
			GroupId:          msg.GroupID,
			MessageType:      int32(msg.MessageType),
			Content:          []byte(msg.Content),
			SendTime:         msg.SendTime,   // 发送时间
			ServerTime:       msg.ServerTime, // 服务器时间
			CreateTime:       msg.SendTime,   // ✅ 创建时间（使用发送时间）
			Seq:              msg.Seq,
		},
	}

	body, err := proto.Marshal(pushMsg)
	if err != nil {
		logger.Error("Failed to marshal push message", zap.Error(err))
		return
	}

	// 获取用户连接
	conn, exists := h.connManager.GetUserConnection(userID)
	if !exists {
		logger.Debug("User not online", zap.String("user_id", userID))
		return
	}

	var data []byte

	if conn.GetType() == transport.ConnectionTypeTCP {
		// TCP 连接：使用自定义协议包格式
		data = protocol.EncodePacket(uint16(protocol.CMD_PUSH_MSG), 0, body)
	} else {
		// WebSocket 连接：使用 WebSocket 消息格式
		wsMsg := &protocol.WebSocketMessage{
			Command:   protocol.CMD_PUSH_MSG,
			Sequence:  0,
			Body:      body,
			Timestamp: utils.GetCurrentMillis(),
		}

		data, err = proto.Marshal(wsMsg)
		if err != nil {
			logger.Error("Failed to marshal websocket message", zap.Error(err))
			return
		}
	}

	// 发送给用户
	if err := conn.Send(data); err != nil {
		logger.Warn("Failed to push message to user", zap.String("user_id", userID), zap.Error(err))
	}
}

// pushMessageToGroup 推送消息给群组所有成员（除了发送者）
func (h *MessageHandler) pushMessageToGroup(groupID, senderID string, msg *model.Message) {
	// 获取群组成员
	members, err := h.groupService.GetGroupMembers(nil, groupID)
	if err != nil {
		logger.Error("Failed to get group members", zap.Error(err), zap.String("group_id", groupID))
		return
	}

	// 推断会话类型
	var conversationType int32
	if msg.GroupID != "" {
		conversationType = 2 // 群聊
	} else {
		conversationType = 1 // 单聊
	}

	pushMsg := &protocol.PushMessage{
		Message: &protocol.MessageInfo{
			ServerMsgId:      msg.ServerMsgID,
			ClientMsgId:      msg.ClientMsgID,
			ConversationId:   msg.ConversationID,
			ConversationType: conversationType, // ✅ 添加 ConversationType
			SenderId:         msg.SenderID,
			ReceiverId:       msg.ReceiverID,
			GroupId:          msg.GroupID,
			MessageType:      int32(msg.MessageType),
			Content:          []byte(msg.Content),
			SendTime:         msg.SendTime,
			ServerTime:       msg.ServerTime,
			CreateTime:       msg.SendTime,
			Seq:              msg.Seq,
		},
	}

	body, err := proto.Marshal(pushMsg)
	if err != nil {
		logger.Error("Failed to marshal push message", zap.Error(err))
		return
	}

	// 推送给所有成员（除了发送者）
	pushCount := 0
	for _, member := range members {
		if member.ID == senderID {
			continue // 跳过发送者
		}

		// 获取成员连接
		conn, exists := h.connManager.GetUserConnection(member.ID)
		if !exists {
			continue // 成员不在线
		}

		var data []byte

		if conn.GetType() == transport.ConnectionTypeTCP {
			// TCP 连接
			data = protocol.EncodePacket(uint16(protocol.CMD_PUSH_MSG), 0, body)
		} else {
			// WebSocket 连接
			wsMsg := &protocol.WebSocketMessage{
				Command:   protocol.CMD_PUSH_MSG,
				Sequence:  0,
				Body:      body,
				Timestamp: utils.GetCurrentMillis(),
			}

			data, err = proto.Marshal(wsMsg)
			if err != nil {
				logger.Error("Failed to marshal websocket message", zap.Error(err))
				continue
			}
		}

		// 发送给成员
		if err := conn.Send(data); err != nil {
			logger.Warn("Failed to push message to group member",
				zap.String("user_id", member.ID),
				zap.String("group_id", groupID),
				zap.Error(err))
		} else {
			pushCount++
		}
	}

	logger.Info("Group message pushed",
		zap.String("group_id", groupID),
		zap.Int("member_count", len(members)),
		zap.Int("push_count", pushCount))
}

// pushToUser 推送消息给指定用户的所有在线设备
func (h *MessageHandler) pushToUser(userID string, command protocol.CommandType, body []byte) {
	// 获取用户连接
	conn, exists := h.connManager.GetUserConnection(userID)
	if !exists {
		logger.Debug("User not online", zap.String("user_id", userID))
		return
	}

	var data []byte
	var err error

	if conn.GetType() == transport.ConnectionTypeTCP {
		// TCP 连接：使用自定义协议包格式
		data = protocol.EncodePacket(uint16(command), 0, body)
	} else {
		// WebSocket 连接：使用 WebSocket 消息格式
		wsMsg := &protocol.WebSocketMessage{
			Command:   command,
			Sequence:  0, // 推送消息不需要序列号
			Body:      body,
			Timestamp: utils.GetCurrentMillis(),
		}

		data, err = protocol.Marshal(wsMsg)
		if err != nil {
			logger.Error("Failed to marshal websocket message", zap.Error(err))
			return
		}
	}

	// 发送给用户
	if err := conn.Send(data); err != nil {
		logger.Debug("Failed to push to user (user may be offline)",
			zap.String("user_id", userID),
			zap.String("command", command.String()))
	}
}
