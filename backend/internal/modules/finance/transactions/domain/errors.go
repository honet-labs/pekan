package domain

import "errors"

var (
	ErrInvalidAccount      = errors.New("invalid account")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInvalidCurrency     = errors.New("invalid currency")
	ErrInvalidType         = errors.New("invalid transaction type")
	ErrInvalidSavingsSelection = errors.New("at least one savings goal must be selected")
	ErrAccountNotFound     = errors.New("account not found")
	ErrCategoryNotFound    = errors.New("category not found")
	ErrCategoryTypeMismatch = errors.New("category type mismatch")
	ErrSavingsNotFound     = errors.New("savings goal not found")
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrAttachmentNotFound  = errors.New("attachment not found")
	ErrInvalidFileType     = errors.New("invalid file type")
	ErrFileTooLarge        = errors.New("file size exceeds limit")
	ErrInvalidScanStatus   = errors.New("invalid attachment scan status")
	ErrAttachmentScanPending = errors.New("attachment is not ready for download")
	ErrAttachmentInfected    = errors.New("attachment is blocked by malware scan")
	ErrInputTooLong          = errors.New("input string too long")
)
