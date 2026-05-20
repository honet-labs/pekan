package domain

import "context"

type CreateAttachmentRecordInput struct {
	TenantID         string
	OwnerType        OwnerType
	OwnerID          string
	FileID           string
	Provider         string
	ObjectKey        string
	OriginalFilename string
	StoredFilename   string
	MimeType         string
	SizeBytes        int64
	UploadedBy       string
}

type Repository interface {
	EnsureOwnerExists(ctx context.Context, tenantID string, ownerType OwnerType, ownerID string) error
	CreateAttachmentRecord(ctx context.Context, in CreateAttachmentRecordInput) (Attachment, error)
	GetAttachmentByID(ctx context.Context, tenantID string, ownerType OwnerType, ownerID, attachmentID string) (Attachment, error)
	ListByOwner(ctx context.Context, tenantID string, ownerType OwnerType, ownerID string) ([]Attachment, error)
	SoftDeleteAttachment(ctx context.Context, tenantID string, ownerType OwnerType, ownerID, attachmentID, actorUserID string) (Attachment, error)
}

