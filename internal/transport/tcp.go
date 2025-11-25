package transport

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/arwen/im-server/internal/protocol"
	"github.com/arwen/im-server/pkg/logger"
	"github.com/arwen/im-server/pkg/utils"
	"go.uber.org/zap"
)

// TCPServer TCP服务器
type TCPServer struct {
	manager        *ConnectionManager
	messageHandler MessageHandler
	listener       net.Listener
}

// NewTCPServer 创建TCP服务器
func NewTCPServer(manager *ConnectionManager, messageHandler MessageHandler) *TCPServer {
	return &TCPServer{
		manager:        manager,
		messageHandler: messageHandler,
	}
}

// Start 启动TCP服务器
func (s *TCPServer) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	
	s.listener = listener
	logger.Info("TCP server starting", zap.String("addr", addr))
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Failed to accept connection", zap.Error(err))
			continue
		}
		
		// 创建连接
		connID := utils.GenerateUUID()
		tcpConn := NewTCPConnection(connID, conn)
		s.manager.AddConnection(tcpConn)
		
		logger.Info("New TCP connection",
			zap.String("conn_id", connID),
			zap.String("remote_addr", conn.RemoteAddr().String()))
		
		// 处理连接
		go s.handleConnection(tcpConn, conn)
	}
}

// Stop 停止TCP服务器
func (s *TCPServer) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *TCPServer) handleConnection(tcpConn *TCPConnection, conn net.Conn) {
	defer func() {
		s.manager.RemoveConnection(tcpConn.GetID())
		tcpConn.Close()
	}()
	
	// 创建编解码器
	codec := NewTCPCodec()
	
	// 设置读取缓冲区
	buffer := make([]byte, 4096)
	
	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	
	for {
		// 读取数据
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				logger.Error("TCP read error", zap.Error(err), zap.String("conn_id", tcpConn.GetID()))
			}
			break
		}
		
		if n == 0 {
			continue
		}
		
		// 更新活跃时间
		tcpConn.UpdateLastActive()
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		
		// 解码数据包（处理粘包/拆包）
		packets, err := codec.Decode(buffer[:n])
		if err != nil {
			logger.Error("Failed to decode packet", zap.Error(err), zap.String("conn_id", tcpConn.GetID()))
			break
		}
		
		// 处理每个完整的数据包
		for _, packet := range packets {
			if err := s.handlePacket(tcpConn, packet); err != nil {
				logger.Error("Failed to handle packet",
					zap.Error(err),
					zap.String("conn_id", tcpConn.GetID()),
					zap.Uint16("command", packet.Header.Command))
			}
		}
	}
}

func (s *TCPServer) handlePacket(conn *TCPConnection, packet *protocol.Packet) error {
	logger.Debug("TCP packet received",
		zap.String("conn_id", conn.GetID()),
		zap.Uint16("command", packet.Header.Command),
		zap.Uint32("sequence", packet.Header.Sequence),
		zap.Int("body_len", len(packet.Body)))
	
	// 特殊处理心跳（不需要业务层处理）
	if packet.Header.Command == uint16(protocol.CMD_HEARTBEAT_REQ) {
		return s.handleHeartbeat(conn, packet.Header.Sequence)
	}
	
	// 直接传递 packet，让 handler 处理
	return s.messageHandler.HandleTCPPacket(conn, packet)
}

func (s *TCPServer) handleHeartbeat(conn *TCPConnection, sequence uint32) error {
	logger.Debug("Heartbeat received", zap.String("conn_id", conn.GetID()))
	
	// 回复心跳响应
	response := protocol.EncodePacket(uint16(protocol.CMD_HEARTBEAT_RSP), sequence, nil)
	return conn.Send(response)
}

// SendPacket 发送数据包
func (s *TCPServer) SendPacket(conn Connection, command uint16, sequence uint32, body []byte) error {
	data := protocol.EncodePacket(command, sequence, body)
	return conn.Send(data)
}

// SendPacketToUser 发送数据包给指定用户
func (s *TCPServer) SendPacketToUser(userID string, command uint16, sequence uint32, body []byte) error {
	data := protocol.EncodePacket(command, sequence, body)
	return s.manager.SendToUser(userID, data)
}

// SendToConnection 发送原始数据到连接
func (s *TCPServer) SendToConnection(connID string, data []byte) error {
	return s.manager.SendToConnection(connID, data)
}

// SendToUser 发送原始数据给用户
func (s *TCPServer) SendToUser(userID string, data []byte) error {
	return s.manager.SendToUser(userID, data)
}

// GetConnectionInfo 获取连接信息
func (s *TCPServer) GetConnectionInfo(connID string) (string, error) {
	conn, exists := s.manager.GetConnection(connID)
	if !exists {
		return "", ErrConnectionNotFound
	}
	
	return fmt.Sprintf("TCP Connection [ID=%s, UserID=%s, Type=%d]",
		conn.GetID(), conn.GetUserID(), conn.GetType()), nil
}

