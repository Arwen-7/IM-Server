package protocol

import (
	"encoding/binary"
	"errors"
)

const (
	// PacketHeaderSize 包头大小（16字节）
	PacketHeaderSize = 16
	// MagicNumber 协议魔数
	MagicNumber = 0xEF89
	// ProtocolVersion 协议版本
	ProtocolVersion = 1
)

// PacketHeader 包头
type PacketHeader struct {
	Magic    uint16 // 2字节：魔数
	Version  uint8  // 1字节：版本
	Command  uint16 // 2字节：命令类型
	Sequence uint32 // 4字节：序列号
	BodyLen  uint32 // 4字节：包体长度
	Reserved [3]byte // 3字节：保留
}

// Packet 数据包
type Packet struct {
	Header *PacketHeader
	Body   []byte
}

// EncodePacketHeader 编码包头
func EncodePacketHeader(header *PacketHeader) []byte {
	buf := make([]byte, PacketHeaderSize)
	
	binary.BigEndian.PutUint16(buf[0:2], header.Magic)
	buf[2] = header.Version
	binary.BigEndian.PutUint16(buf[3:5], header.Command)
	binary.BigEndian.PutUint32(buf[5:9], header.Sequence)
	binary.BigEndian.PutUint32(buf[9:13], header.BodyLen)
	copy(buf[13:16], header.Reserved[:])
	
	return buf
}

// DecodePacketHeader 解码包头
func DecodePacketHeader(data []byte) (*PacketHeader, error) {
	if len(data) < PacketHeaderSize {
		return nil, errors.New("invalid packet header size")
	}
	
	header := &PacketHeader{
		Magic:    binary.BigEndian.Uint16(data[0:2]),
		Version:  data[2],
		Command:  binary.BigEndian.Uint16(data[3:5]),
		Sequence: binary.BigEndian.Uint32(data[5:9]),
		BodyLen:  binary.BigEndian.Uint32(data[9:13]),
	}
	copy(header.Reserved[:], data[13:16])
	
	// 验证魔数
	if header.Magic != MagicNumber {
		return nil, errors.New("invalid magic number")
	}
	
	// 验证版本
	if header.Version != ProtocolVersion {
		return nil, errors.New("unsupported protocol version")
	}
	
	return header, nil
}

// EncodePacket 编码数据包
func EncodePacket(command uint16, sequence uint32, body []byte) []byte {
	header := &PacketHeader{
		Magic:    MagicNumber,
		Version:  ProtocolVersion,
		Command:  command,
		Sequence: sequence,
		BodyLen:  uint32(len(body)),
	}
	
	headerBytes := EncodePacketHeader(header)
	return append(headerBytes, body...)
}

// DecodePacket 解码数据包
func DecodePacket(data []byte) (*Packet, error) {
	if len(data) < PacketHeaderSize {
		return nil, errors.New("packet too short")
	}
	
	header, err := DecodePacketHeader(data[:PacketHeaderSize])
	if err != nil {
		return nil, err
	}
	
	if len(data) < PacketHeaderSize+int(header.BodyLen) {
		return nil, errors.New("incomplete packet body")
	}
	
	body := data[PacketHeaderSize : PacketHeaderSize+header.BodyLen]
	
	return &Packet{
		Header: header,
		Body:   body,
	}, nil
}

