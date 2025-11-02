package transport

import (
	"net/http"
	"time"

	"github.com/arwen/im-server/pkg/logger"
	"github.com/arwen/im-server/pkg/utils"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// WebSocketServer WebSocket服务器
type WebSocketServer struct {
	manager        *ConnectionManager
	messageHandler MessageHandler
}

// NewWebSocketServer 创建WebSocket服务器
func NewWebSocketServer(manager *ConnectionManager, messageHandler MessageHandler) *WebSocketServer {
	return &WebSocketServer{
		manager:        manager,
		messageHandler: messageHandler,
	}
}

// HandleWebSocket 处理WebSocket连接
func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级连接
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade websocket", zap.Error(err))
		return
	}

	// 创建连接
	connID := utils.GenerateUUID()
	conn := NewWSConnection(connID, wsConn)
	s.manager.AddConnection(conn)

	logger.Info("New WebSocket connection", zap.String("conn_id", connID))

	// 处理连接
	go s.handleConnection(conn)
}

func (s *WebSocketServer) handleConnection(conn *WSConnection) {
	defer func() {
		s.manager.RemoveConnection(conn.GetID())
		conn.Close()
	}()

	// 设置读取超时
	conn.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	
	// 设置 Pong Handler（客户端收到我们的 Ping 后回复 Pong）
	conn.conn.SetPongHandler(func(string) error {
		conn.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.UpdateLastActive()
		return nil
	})
	
	// 设置 Ping Handler（收到客户端的 Ping，自动回复 Pong）
	conn.conn.SetPingHandler(func(appData string) error {
		logger.Debug("Received Ping from client, sending Pong", zap.String("conn_id", conn.GetID()))
		conn.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.UpdateLastActive()
		// 回复 Pong
		err := conn.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
		if err != nil {
			logger.Warn("Failed to send pong", zap.Error(err), zap.String("conn_id", conn.GetID()))
		} else {
			logger.Debug("Pong sent successfully", zap.String("conn_id", conn.GetID()))
		}
		return nil
	})

	for {
		// 读取消息
		messageType, data, err := conn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		if messageType != websocket.BinaryMessage {
			continue
		}

		// 更新活跃时间
		conn.UpdateLastActive()
		conn.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// 处理消息
		if err := s.messageHandler.HandleMessage(conn, data); err != nil {
			logger.Error("Failed to handle message", zap.Error(err), zap.String("conn_id", conn.GetID()))
		}
	}
}

// Start 启动WebSocket服务器
func (s *WebSocketServer) Start(addr string) error {
	http.HandleFunc("/ws", s.HandleWebSocket)
	logger.Info("WebSocket server starting", zap.String("addr", addr))
	return http.ListenAndServe(addr, nil)
}

