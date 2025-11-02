package transport

import "errors"

var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrSendBufferFull   = errors.New("send buffer full")
	ErrConnectionNotFound = errors.New("connection not found")
	ErrUserNotOnline = errors.New("user not online")
)

