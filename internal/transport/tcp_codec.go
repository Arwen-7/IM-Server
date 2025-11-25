package transport

import (
	"bytes"
	"errors"
	"io"

	"github.com/arwen/im-server/internal/protocol"
)

var (
	ErrPacketTooLarge = errors.New("packet too large")
	ErrInvalidPacket  = errors.New("invalid packet")
)

const (
	// MaxPacketSize 最大包大小（4MB）
	MaxPacketSize = 4 * 1024 * 1024
)

// TCPCodec TCP 编解码器（处理粘包/拆包）
type TCPCodec struct {
	buffer *bytes.Buffer
}

// NewTCPCodec 创建TCP编解码器
func NewTCPCodec() *TCPCodec {
	return &TCPCodec{
		buffer: bytes.NewBuffer(make([]byte, 0, 4096)),
	}
}

// Decode 解码数据包（处理粘包/拆包）
// 返回解码出的完整数据包列表
func (c *TCPCodec) Decode(data []byte) ([]*protocol.Packet, error) {
	// 将新数据追加到缓冲区
	c.buffer.Write(data)
	
	packets := make([]*protocol.Packet, 0)
	
	for {
		// 检查是否有足够的数据读取包头
		if c.buffer.Len() < protocol.PacketHeaderSize {
			break
		}
		
		// 预览包头（不从缓冲区移除）
		headerBytes := c.buffer.Bytes()[:protocol.PacketHeaderSize]
		header, err := protocol.DecodePacketHeader(headerBytes)
		if err != nil {
			// 包头无效，丢弃数据
			c.buffer.Reset()
			return nil, ErrInvalidPacket
		}
		
		// 检查包体长度是否合法
		if header.BodyLen > MaxPacketSize {
			c.buffer.Reset()
			return nil, ErrPacketTooLarge
		}
		
		// 计算完整包长度
		totalLen := protocol.PacketHeaderSize + int(header.BodyLen)
		
		// 检查缓冲区是否有完整的包
		if c.buffer.Len() < totalLen {
			// 数据不完整，等待更多数据
			break
		}
		
		// 读取完整的包数据
		packetData := make([]byte, totalLen)
		n, err := c.buffer.Read(packetData)
		if err != nil && err != io.EOF {
			return packets, err
		}
		if n != totalLen {
			return packets, ErrInvalidPacket
		}
		
		// 解码数据包
		packet, err := protocol.DecodePacket(packetData)
		if err != nil {
			return packets, err
		}
		
		packets = append(packets, packet)
	}
	
	return packets, nil
}

// Reset 重置缓冲区
func (c *TCPCodec) Reset() {
	c.buffer.Reset()
}

