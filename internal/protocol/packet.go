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

// PacketHeader 包头（与客户端格式一致）
type PacketHeader struct {
	Magic    uint16 // 2字节：魔数 0xEF89
	Version  uint8  // 1字节：版本
	Flags    uint8  // 1字节：标志位（预留，用于扩展：加密、压缩等）
	Command  uint16 // 2字节：命令类型
	Sequence uint32 // 4字节：序列号
	BodyLen  uint32 // 4字节：包体长度
	CRC16    uint16 // 2字节：CRC16校验值（校验前14字节）
}

// Packet 数据包
type Packet struct {
	Header *PacketHeader
	Body   []byte
}

// CRC16 计算CRC16校验值（CRC-16/CCITT-FALSE）
func CRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	polynomial := uint16(0x1021)
	
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ polynomial
			} else {
				crc = crc << 1
			}
		}
	}
	
	return crc
}

// EncodePacketHeader 编码包头
func EncodePacketHeader(header *PacketHeader) []byte {
	buf := make([]byte, PacketHeaderSize)
	
	// 前14字节（用于CRC计算）
	binary.BigEndian.PutUint16(buf[0:2], header.Magic)
	buf[2] = header.Version
	buf[3] = header.Flags
	binary.BigEndian.PutUint16(buf[4:6], header.Command)
	binary.BigEndian.PutUint32(buf[6:10], header.Sequence)
	binary.BigEndian.PutUint32(buf[10:14], header.BodyLen)
	
	// 计算CRC16（校验前14字节）
	crc := CRC16(buf[:14])
	binary.BigEndian.PutUint16(buf[14:16], crc)
	
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
		Flags:    data[3],
		Command:  binary.BigEndian.Uint16(data[4:6]),
		Sequence: binary.BigEndian.Uint32(data[6:10]),
		BodyLen:  binary.BigEndian.Uint32(data[10:14]),
		CRC16:    binary.BigEndian.Uint16(data[14:16]),
	}
	
	// 验证魔数
	if header.Magic != MagicNumber {
		return nil, errors.New("invalid magic number")
	}
	
	// 验证版本
	if header.Version != ProtocolVersion {
		return nil, errors.New("unsupported protocol version")
	}
	
	// 验证CRC16（校验前14字节）
	calculatedCRC := CRC16(data[:14])
	if calculatedCRC != header.CRC16 {
		return nil, errors.New("invalid CRC16 checksum")
	}
	
	return header, nil
}

// EncodePacket 编码数据包
func EncodePacket(command uint16, sequence uint32, body []byte) []byte {
	header := &PacketHeader{
		Magic:    MagicNumber,
		Version:  ProtocolVersion,
		Flags:    0, // 默认无标志
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

