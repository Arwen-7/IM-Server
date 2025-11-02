package service

import "errors"

var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidToken     = errors.New("invalid token")
)

