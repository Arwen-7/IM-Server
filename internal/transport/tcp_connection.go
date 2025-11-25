package transport

import (
	"net"
	"sync"
	"time"

	"github.com/arwen/im-server/pkg/logger"
	"go.uber.org/zap"
)

// TCPConnection TCP连接
type TCPConnection struct {
	id         string
	userID     string
	conn       net.Conn
	sendCh     chan []byte
	closeCh    chan struct{}
	lastActive time.Time
	mu         sync.RWMutex
	closed     bool
}

// NewTCPConnection 创建TCP连接
func NewTCPConnection(id string, conn net.Conn) *TCPConnection {
	c := &TCPConnection{
		id:         id,
		conn:       conn,
		sendCh:     make(chan []byte, 256),
		closeCh:    make(chan struct{}),
		lastActive: time.Now(),
	}
	
	go c.writePump()
	return c
}

func (c *TCPConnection) GetID() string {
	return c.id
}

func (c *TCPConnection) GetUserID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userID
}

func (c *TCPConnection) SetUserID(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userID = userID
}

func (c *TCPConnection) GetType() ConnectionType {
	return ConnectionTypeTCP
}

func (c *TCPConnection) Send(data []byte) error {
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

func (c *TCPConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil
	}
	
	c.closed = true
	close(c.closeCh)
	return c.conn.Close()
}

func (c *TCPConnection) IsAlive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.closed && time.Since(c.lastActive) < 90*time.Second
}

func (c *TCPConnection) UpdateLastActive() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastActive = time.Now()
}

func (c *TCPConnection) writePump() {
	defer func() {
		c.Close()
	}()
	
	for {
		select {
		case data := <-c.sendCh:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := c.conn.Write(data); err != nil {
				logger.Error("TCP write error", zap.Error(err), zap.String("conn_id", c.id))
				return
			}
		case <-c.closeCh:
			return
		}
	}
}

