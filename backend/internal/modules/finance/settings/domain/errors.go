package domain

import "errors"

var (
	ErrInvalidChannelCode = errors.New("invalid channel_code")
	ErrInvalidTemplate    = errors.New("invalid template payload")
	ErrMembershipNotFound = errors.New("membership not found")
	ErrRoleNotFound       = errors.New("one or more roles not found in tenant")
	ErrRoleSystemLocked   = errors.New("system role cannot be modified")
	ErrRoleCodeDuplicate  = errors.New("role code already exists")
	ErrInvalidRoleCode    = errors.New("invalid role code")
	ErrInvalidRoleName    = errors.New("invalid role name")
	ErrUserNotFound       = errors.New("user membership not found")
	ErrInvalidUserEmail   = errors.New("invalid user email")
	ErrInvalidUserName    = errors.New("invalid user name")
	ErrInvalidPassword    = errors.New("password must be at least 8 characters")
	ErrUserEmailDuplicate = errors.New("email already exists")
)
