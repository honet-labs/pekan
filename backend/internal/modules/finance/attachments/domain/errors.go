package domain

import "errors"

var (
	ErrInvalidOwnerType      = errors.New("invalid owner_type")
	ErrOwnerNotFound         = errors.New("owner not found")
	ErrAttachmentNotFound    = errors.New("attachment not found")
	ErrInvalidFileType       = errors.New("invalid file type; only jpg/jpeg/png/webp are allowed")
	ErrFileTooLarge          = errors.New("file too large; max size is 10MB")
	ErrAttachmentInfected    = errors.New("attachment scan status is infected")
	ErrAttachmentScanPending = errors.New("attachment scan is not completed")
)
