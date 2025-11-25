package protocol

import (
	"google.golang.org/protobuf/proto"
)

// 为了方便使用，提供不带前缀的常量别名
const (
	CMD_UNKNOWN = CommandType_CMD_UNKNOWN
	
	// 连接相关
	CMD_CONNECT_REQ    = CommandType_CMD_CONNECT_REQ
	CMD_CONNECT_RSP    = CommandType_CMD_CONNECT_RSP
	CMD_DISCONNECT_REQ = CommandType_CMD_DISCONNECT_REQ
	CMD_DISCONNECT_RSP = CommandType_CMD_DISCONNECT_RSP
	CMD_HEARTBEAT_REQ  = CommandType_CMD_HEARTBEAT_REQ
	CMD_HEARTBEAT_RSP  = CommandType_CMD_HEARTBEAT_RSP
	
	// 认证相关
	CMD_AUTH_REQ   = CommandType_CMD_AUTH_REQ
	CMD_AUTH_RSP   = CommandType_CMD_AUTH_RSP
	CMD_REAUTH_REQ = CommandType_CMD_REAUTH_REQ
	CMD_REAUTH_RSP = CommandType_CMD_REAUTH_RSP
	CMD_KICK_OUT   = CommandType_CMD_KICK_OUT
	
	// 消息相关
	CMD_SEND_MSG_REQ    = CommandType_CMD_SEND_MSG_REQ
	CMD_SEND_MSG_RSP    = CommandType_CMD_SEND_MSG_RSP
	CMD_PUSH_MSG        = CommandType_CMD_PUSH_MSG
	CMD_MSG_ACK         = CommandType_CMD_MSG_ACK
	CMD_BATCH_MSG       = CommandType_CMD_BATCH_MSG
	CMD_REVOKE_MSG_REQ  = CommandType_CMD_REVOKE_MSG_REQ
	CMD_REVOKE_MSG_RSP  = CommandType_CMD_REVOKE_MSG_RSP
	CMD_REVOKE_MSG_PUSH = CommandType_CMD_REVOKE_MSG_PUSH
	
	// 同步相关
	CMD_BATCH_SYNC_REQ = CommandType_CMD_BATCH_SYNC_REQ
	CMD_BATCH_SYNC_RSP = CommandType_CMD_BATCH_SYNC_RSP
	CMD_SYNC_FINISHED  = CommandType_CMD_SYNC_FINISHED
	CMD_SYNC_RANGE_REQ = CommandType_CMD_SYNC_RANGE_REQ
	CMD_SYNC_RANGE_RSP = CommandType_CMD_SYNC_RANGE_RSP
	
	// 在线状态
	CMD_ONLINE_STATUS_REQ  = CommandType_CMD_ONLINE_STATUS_REQ
	CMD_ONLINE_STATUS_RSP  = CommandType_CMD_ONLINE_STATUS_RSP
	CMD_STATUS_CHANGE_PUSH = CommandType_CMD_STATUS_CHANGE_PUSH
	
	// 已读回执
	CMD_READ_RECEIPT_REQ  = CommandType_CMD_READ_RECEIPT_REQ
	CMD_READ_RECEIPT_RSP  = CommandType_CMD_READ_RECEIPT_RSP
	CMD_READ_RECEIPT_PUSH = CommandType_CMD_READ_RECEIPT_PUSH
	
	// 输入状态
	CMD_TYPING_STATUS_REQ  = CommandType_CMD_TYPING_STATUS_REQ
	CMD_TYPING_STATUS_PUSH = CommandType_CMD_TYPING_STATUS_PUSH
)

// 错误码别名
const (
	ERR_SUCCESS                = ErrorCode_ERR_SUCCESS
	ERR_UNKNOWN                = ErrorCode_ERR_UNKNOWN
	ERR_INVALID_PARAM          = ErrorCode_ERR_INVALID_PARAM
	ERR_AUTH_FAILED            = ErrorCode_ERR_AUTH_FAILED
	ERR_TOKEN_EXPIRED          = ErrorCode_ERR_TOKEN_EXPIRED
	ERR_PERMISSION_DENIED      = ErrorCode_ERR_PERMISSION_DENIED
	ERR_USER_NOT_EXIST         = ErrorCode_ERR_USER_NOT_EXIST
	ERR_MESSAGE_TOO_LARGE      = ErrorCode_ERR_MESSAGE_TOO_LARGE
	ERR_SEND_TOO_FAST          = ErrorCode_ERR_SEND_TOO_FAST
	ERR_CONVERSATION_NOT_EXIST = ErrorCode_ERR_CONVERSATION_NOT_EXIST
)

// Marshal 序列化消息
func Marshal(msg proto.Message) ([]byte, error) {
	return proto.Marshal(msg)
}

// Unmarshal 反序列化消息
func Unmarshal(data []byte, msg proto.Message) error {
	return proto.Unmarshal(data, msg)
}

// MarshalWebSocketMessage 序列化 WebSocket 消息
func MarshalWebSocketMessage(wsMsg *WebSocketMessage) ([]byte, error) {
	return proto.Marshal(wsMsg)
}

// UnmarshalWebSocketMessage 反序列化 WebSocket 消息
func UnmarshalWebSocketMessage(data []byte) (*WebSocketMessage, error) {
	var wsMsg WebSocketMessage
	err := proto.Unmarshal(data, &wsMsg)
	if err != nil {
		return nil, err
	}
	return &wsMsg, nil
}

