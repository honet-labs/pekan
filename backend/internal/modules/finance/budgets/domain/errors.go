package domain

import "errors"

var (
	ErrInvalidName         = errors.New("invalid budget name")
	ErrInvalidAmount       = errors.New("invalid budget amount")
	ErrInvalidCurrency     = errors.New("invalid currency")
	ErrInvalidPeriod       = errors.New("invalid period")
	ErrInvalidDateRange    = errors.New("invalid date range")
	ErrInvalidAlert        = errors.New("invalid alert threshold")
	ErrInvalidStatus       = errors.New("invalid status")
	ErrBudgetNotFound      = errors.New("budget not found")
	ErrCategoryNotFound    = errors.New("category not found")
	ErrInputTooLong        = errors.New("input string too long")
)
