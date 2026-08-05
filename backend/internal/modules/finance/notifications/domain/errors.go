package domain

import "errors"

var (
	ErrInvalidTitle        = errors.New("invalid notification title")
	ErrInvalidMessage      = errors.New("invalid notification message")
	ErrInvalidType         = errors.New("invalid notification type")
	ErrInvalidStatus       = errors.New("invalid notification status")
	ErrNotificationNotFound = errors.New("notification not found")
)

