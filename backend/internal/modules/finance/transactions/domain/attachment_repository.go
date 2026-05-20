package domain

import "context"

type CreateAttachmentRecordInput struct {
	TenantID         string
	TransactionID    string
	FileID           string
	Provider         string
	ObjectKey        string
	OriginalFilename string
	StoredFilename   string
	MimeType         string
	SizeBytes        int64
	UploadedBy       string
}

type AttachmentRepository interface {
	EnsureTransactionExists(ctx context.Context, tenantID, transactionID string) error
	CreateAttachmentRecord(ctx context.Context, in CreateAttachmentRecordInput) (Attachment, error)
	ListAttachmentsByTransaction(ctx context.Context, tenantID, transactionID string) ([]Attachment, error)
	GetAttachmentByID(ctx context.Context, tenantID, transactionID, attachmentID string) (Attachment, error)
	SetAttachmentScanStatus(ctx context.Context, tenantID, transactionID, attachmentID, scanStatus string) error
}
