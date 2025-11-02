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
	connManager *transport.ConnectionManager
	userService *service.UserService
	msgService  *service.MessageService
	convService *service.ConversationService
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(
	connManager *transport.ConnectionManager,
	userService *service.UserService,
	msgService *service.MessageService,
	convService *service.ConversationService,
) *MessageHandler {
	return &MessageHandler{
		connManager: connManager,
		userService: userService,
		msgService:  msgService,
		convService: convService,
	}
}

// HandleMessage 处理消息
func (h *MessageHandler) HandleMessage(conn transport.Connection, data []byte) error {
	// 解析WebSocket消息
	wsMsg, err := protocol.UnmarshalWebSocketMessage(data)
	if err != nil {
		logger.Error("Failed to unmarshal message", zap.Error(err))
		return err
	}

	logger.Debug("Received message",
		zap.String("conn_id", conn.GetID()),
		zap.String("command", wsMsg.Command.String()),
		zap.Uint32("sequence", wsMsg.Sequence))

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
	case protocol.CMD_SYNC_REQ:
		return h.handleSync(conn, wsMsg)
	case protocol.CMD_READ_RECEIPT_REQ:
		return h.handleReadReceipt(conn, wsMsg)
	case protocol.CMD_TYPING_STATUS_REQ:
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

	// 创建消息
	msg := &model.Message{
		ID:             utils.GenerateMessageID(),
		ClientMsgID:    req.ClientMsgId,
		ConversationID: req.ConversationId,
		SenderID:       userID,
		ReceiverID:     req.ReceiverId,
		GroupID:        req.GroupId,
		MessageType:    int(req.MessageType),
		Content:        string(req.Content),
		SendTime:       req.SendTime,
		ServerTime:     now,
		Status:         1, // 已发送
	}

	// 保存消息
	if err := h.msgService.SaveMessage(msg); err != nil {
		logger.Error("Failed to save message", zap.Error(err))
		resp := &protocol.SendMessageResponse{
			ErrorCode: protocol.ERR_UNKNOWN,
			ErrorMsg:  "Failed to save message",
		}
		return h.sendResponse(conn, protocol.CommandType_CMD_SEND_MSG_RSP, wsMsg.Sequence, resp)
	}

	// 更新会话
	h.convService.UpdateLastMessage(req.ConversationId, msg.ID, string(req.Content), now)

	// 发送响应（返回客户端的消息 ID，让客户端能正确匹配 ACK）
	resp := &protocol.SendMessageResponse{
		ErrorCode:  protocol.ERR_SUCCESS,
		ErrorMsg:   "Success",
		MessageId:  req.ClientMsgId, // ✅ 返回客户端的消息 ID
		Seq:        msg.Seq,
		ServerTime: now,
	}
	if err := h.sendResponse(conn, protocol.CMD_SEND_MSG_RSP, wsMsg.Sequence, resp); err != nil {
		return err
	}

	// 推送给接收者
	if req.ReceiverId != "" {
		h.pushMessageToUser(req.ReceiverId, msg)
	}

	logger.Info("Message sent", zap.String("msg_id", msg.ID), zap.String("sender", userID), zap.String("receiver", req.ReceiverId))
	return nil
}

// handleSync 处理同步
func (h *MessageHandler) handleSync(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	var req protocol.SyncRequest
	if err := protocol.Unmarshal(wsMsg.Body, &req); err != nil {
		return err
	}

	userID := conn.GetUserID()
	if userID == "" {
		return nil
	}

	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	// 同步消息
	messages, serverMaxSeq, hasMore, err := h.msgService.SyncMessages(userID, req.MinSeq, req.MaxSeq, limit)
	if err != nil {
		logger.Error("Failed to sync messages", zap.Error(err))
		resp := &protocol.SyncResponse{
			ErrorCode: protocol.ERR_UNKNOWN,
			ErrorMsg:  "Failed to sync messages",
		}
		return h.sendResponse(conn, protocol.CMD_SYNC_RSP, wsMsg.Sequence, resp)
	}

	// 转换消息
	var pushMessages []*protocol.PushMessage
	for _, msg := range messages {
		pushMsg := &protocol.PushMessage{
			MessageId:      msg.ID,
			ClientMsgId:    msg.ClientMsgID,
			ConversationId: msg.ConversationID,
			SenderId:       msg.SenderID,
			ReceiverId:     msg.ReceiverID,
			GroupId:        msg.GroupID,
			MessageType:    int32(msg.MessageType),
			Content:        []byte(msg.Content),
			SendTime:       msg.SendTime,
			ServerTime:     msg.ServerTime,
			Seq:            msg.Seq,
		}
		pushMessages = append(pushMessages, pushMsg)
	}

	resp := &protocol.SyncResponse{
		ErrorCode:    protocol.ERR_SUCCESS,
		ErrorMsg:     "Success",
		Messages:     pushMessages,
		ServerMaxSeq: serverMaxSeq,
		HasMore:      hasMore,
	}

	logger.Info("Sync messages", zap.String("user_id", userID), zap.Int("count", len(messages)))
	return h.sendResponse(conn, protocol.CMD_SYNC_RSP, wsMsg.Sequence, resp)
}

// handleMessageAck 处理消息ACK
func (h *MessageHandler) handleMessageAck(conn transport.Connection, wsMsg *protocol.WebSocketMessage) error {
	var req protocol.MessageAck
	if err := protocol.Unmarshal(wsMsg.Body, &req); err != nil {
		return err
	}

	logger.Debug("Message ACK", zap.String("msg_id", req.MessageId), zap.Int64("seq", req.Seq))
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
		return nil
	}

	now := utils.GetCurrentMillis()

	// 保存已读回执
	for _, msgID := range req.MessageIds {
		h.msgService.SaveReadReceipt(msgID, req.ConversationId, userID, now)
	}

	// 清除未读数
	h.convService.ClearUnreadCount(req.ConversationId)

	resp := &protocol.ReadReceiptResponse{
		ErrorCode: protocol.ERR_SUCCESS,
		ErrorMsg:  "Success",
	}

	return h.sendResponse(conn, protocol.CMD_READ_RECEIPT_RSP, wsMsg.Sequence, resp)
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
	if err := h.msgService.RevokeMessage(req.MessageId, userID); err != nil {
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

	logger.Info("Message revoked", zap.String("msg_id", req.MessageId), zap.String("user_id", userID))
	return h.sendResponse(conn, protocol.CMD_REVOKE_MSG_RSP, wsMsg.Sequence, resp)
}

// sendResponse 发送响应
func (h *MessageHandler) sendResponse(conn transport.Connection, command protocol.CommandType, sequence uint32, message proto.Message) error {
	body, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	wsMsg := &protocol.WebSocketMessage{
		Command:   command,
		Sequence:  sequence,
		Body:      body,
		Timestamp: utils.GetCurrentMillis(),
	}

	data, err := protocol.MarshalWebSocketMessage(wsMsg)
	if err != nil {
		return err
	}

	return conn.Send(data)
}

// pushMessageToUser 推送消息给用户
func (h *MessageHandler) pushMessageToUser(userID string, msg *model.Message) {
	pushMsg := &protocol.PushMessage{
		MessageId:      msg.ID,
		ClientMsgId:    msg.ClientMsgID,
		ConversationId: msg.ConversationID,
		SenderId:       msg.SenderID,
		ReceiverId:     msg.ReceiverID,
		GroupId:        msg.GroupID,
		MessageType:    int32(msg.MessageType),
		Content:        []byte(msg.Content),
		SendTime:       msg.SendTime,
		ServerTime:     msg.ServerTime,
		Seq:            msg.Seq,
	}

	body, err := proto.Marshal(pushMsg)
	if err != nil {
		logger.Error("Failed to marshal push message", zap.Error(err))
		return
	}

	wsMsg := &protocol.WebSocketMessage{
		Command:   protocol.CMD_PUSH_MSG,
		Sequence:  0,
		Body:      body,
		Timestamp: utils.GetCurrentMillis(),
	}

	data, err := proto.Marshal(wsMsg)
	if err != nil {
		logger.Error("Failed to marshal websocket message", zap.Error(err))
		return
	}

	// 发送给用户
	if err := h.connManager.SendToUser(userID, data); err != nil {
		logger.Warn("Failed to push message to user", zap.String("user_id", userID), zap.Error(err))
	}
}

