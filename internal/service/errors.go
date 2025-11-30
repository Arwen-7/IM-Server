package service

import "errors"

var (
	// Permission errors
	ErrPermissionDenied = errors.New("permission denied")
	
	// User errors
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidToken     = errors.New("invalid token")
	
	// Group errors
	ErrGroupNotFound       = errors.New("group not found")
	ErrAlreadyGroupMember  = errors.New("already a group member")
	ErrNotGroupMember      = errors.New("not a group member")
	ErrOwnerCannotLeave    = errors.New("owner cannot leave group")
)

