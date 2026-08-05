package domain

import "errors"

var (
	ErrInvalidTitle       = errors.New("invalid reminder title")
	ErrInvalidAmount      = errors.New("invalid reminder amount")
	ErrInvalidCurrency    = errors.New("invalid currency")
	ErrInvalidDate        = errors.New("invalid due date")
	ErrInvalidRepeat      = errors.New("invalid repeat interval")
	ErrInvalidStatus      = errors.New("invalid status")
	ErrReminderNotFound   = errors.New("reminder not found")
)

