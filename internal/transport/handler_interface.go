package transport

// MessageHandler 消息处理器接口
// 用于解耦 transport 和 handler 包，避免循环依赖
type MessageHandler interface {
	HandleMessage(conn Connection, data []byte) error
}

