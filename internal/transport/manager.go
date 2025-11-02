package transport

import (
	"sync"

	"github.com/arwen/im-server/pkg/logger"
	"go.uber.org/zap"
)

// ConnectionManager 连接管理器
type ConnectionManager struct {
	connections map[string]Connection // connID -> Connection
	userConns   map[string]string     // userID -> connID
	mu          sync.RWMutex
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]Connection),
		userConns:   make(map[string]string),
	}
}

// AddConnection 添加连接
func (m *ConnectionManager) AddConnection(conn Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.connections[conn.GetID()] = conn
	logger.Info("Connection added", zap.String("conn_id", conn.GetID()))
}

// RemoveConnection 移除连接
func (m *ConnectionManager) RemoveConnection(connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	conn, exists := m.connections[connID]
	if !exists {
		return
	}
	
	// 移除用户映射
	userID := conn.GetUserID()
	if userID != "" {
		delete(m.userConns, userID)
	}
	
	delete(m.connections, connID)
	logger.Info("Connection removed", zap.String("conn_id", connID), zap.String("user_id", userID))
}

// GetConnection 获取连接
func (m *ConnectionManager) GetConnection(connID string) (Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	conn, exists := m.connections[connID]
	return conn, exists
}

// BindUser 绑定用户
func (m *ConnectionManager) BindUser(connID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	conn, exists := m.connections[connID]
	if !exists {
		return ErrConnectionNotFound
	}
	
	// 检查用户是否已有连接
	if oldConnID, exists := m.userConns[userID]; exists && oldConnID != connID {
		// 踢掉旧连接
		if oldConn, exists := m.connections[oldConnID]; exists {
			logger.Info("Kicking old connection", zap.String("user_id", userID), zap.String("old_conn_id", oldConnID))
			oldConn.Close()
			delete(m.connections, oldConnID)
		}
	}
	
	conn.SetUserID(userID)
	m.userConns[userID] = connID
	
	logger.Info("User bound to connection", zap.String("user_id", userID), zap.String("conn_id", connID))
	return nil
}

// GetUserConnection 获取用户连接
func (m *ConnectionManager) GetUserConnection(userID string) (Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	connID, exists := m.userConns[userID]
	if !exists {
		return nil, false
	}
	
	conn, exists := m.connections[connID]
	return conn, exists
}

// SendToUser 发送消息给用户
func (m *ConnectionManager) SendToUser(userID string, data []byte) error {
	conn, exists := m.GetUserConnection(userID)
	if !exists {
		return ErrUserNotOnline
	}
	
	return conn.Send(data)
}

// SendToConnection 发送消息给连接
func (m *ConnectionManager) SendToConnection(connID string, data []byte) error {
	conn, exists := m.GetConnection(connID)
	if !exists {
		return ErrConnectionNotFound
	}
	
	return conn.Send(data)
}

// GetConnectionCount 获取连接数
func (m *ConnectionManager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// GetOnlineUserCount 获取在线用户数
func (m *ConnectionManager) GetOnlineUserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.userConns)
}

// IsUserOnline 检查用户是否在线
func (m *ConnectionManager) IsUserOnline(userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	connID, exists := m.userConns[userID]
	if !exists {
		return false
	}
	
	conn, exists := m.connections[connID]
	return exists && conn.IsAlive()
}

