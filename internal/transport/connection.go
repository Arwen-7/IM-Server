package transport

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ConnectionType 连接类型
type ConnectionType int

const (
	ConnectionTypeWebSocket ConnectionType = 1
	ConnectionTypeTCP       ConnectionType = 2
)

// Connection 连接接口
type Connection interface {
	GetID() string
	GetUserID() string
	SetUserID(userID string)
	GetType() ConnectionType
	Send(data []byte) error
	Close() error
	IsAlive() bool
	UpdateLastActive()
}

// WSConnection WebSocket连接
type WSConnection struct {
	id         string
	userID     string
	conn       *websocket.Conn
	sendCh     chan []byte
	closeCh    chan struct{}
	lastActive time.Time
	mu         sync.RWMutex
	closed     bool
}

// NewWSConnection 创建WebSocket连接
func NewWSConnection(id string, conn *websocket.Conn) *WSConnection {
	c := &WSConnection{
		id:         id,
		conn:       conn,
		sendCh:     make(chan []byte, 256),
		closeCh:    make(chan struct{}),
		lastActive: time.Now(),
	}
	
	go c.writePump()
	return c
}

func (c *WSConnection) GetID() string {
	return c.id
}

func (c *WSConnection) GetUserID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userID
}

func (c *WSConnection) SetUserID(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userID = userID
}

func (c *WSConnection) GetType() ConnectionType {
	return ConnectionTypeWebSocket
}

func (c *WSConnection) Send(data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.closed {
		return ErrConnectionClosed
	}
	
	select {
	case c.sendCh <- data:
		return nil
	default:
		return ErrSendBufferFull
	}
}

func (c *WSConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil
	}
	
	c.closed = true
	close(c.closeCh)
	return c.conn.Close()
}

func (c *WSConnection) IsAlive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.closed && time.Since(c.lastActive) < 90*time.Second
}

func (c *WSConnection) UpdateLastActive() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastActive = time.Now()
}

func (c *WSConnection) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
	}()
	
	for {
		select {
		case data := <-c.sendCh:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.closeCh:
			return
		}
	}
}

