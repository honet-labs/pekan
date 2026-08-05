package domain

import "errors"

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserInactive         = errors.New("user inactive")
	ErrMembershipNotFound   = errors.New("membership not found")
	ErrMembershipSuspended  = errors.New("membership suspended")
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionExpired       = errors.New("session expired")
	ErrSessionRevoked       = errors.New("session revoked")
	ErrRefreshTokenReused   = errors.New("refresh token reuse detected")
	ErrAccessProfileMissing = errors.New("access profile missing")
)
