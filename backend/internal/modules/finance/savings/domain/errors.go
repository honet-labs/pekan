package domain

import "errors"

var (
	ErrInvalidName      = errors.New("invalid savings name")
	ErrInvalidAmount    = errors.New("invalid savings amount")
	ErrInvalidCurrency  = errors.New("invalid currency")
	ErrInvalidStatus    = errors.New("invalid status")
	ErrSavingsNotFound  = errors.New("savings goal not found")
)

