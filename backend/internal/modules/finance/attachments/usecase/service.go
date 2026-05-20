package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/attachments/domain"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/storage"
)

const (
	maxAttachmentBytes = 10 * 1024 * 1024
)

var allowedMimeTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

var allowedExtensionsByMIME = map[string]map[string]struct{}{
	"image/jpeg": {
		".jpg":  {},
		".jpeg": {},
	},
	"image/png": {
		".png": {},
	},
	"image/webp": {
		".webp": {},
	},
}

type Authorizer interface {
	EnsureModule(ctx context.Context, module string) error
	EnsureFeature(ctx context.Context, feature string) error
	EnsurePermission(ctx context.Context, permission string) error
}

type AuditLogger interface {
	Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error
}

type Service struct {
	repo    domain.Repository
	authz   Authorizer
	audit   AuditLogger
	storage storage.ObjectStorage
}

func NewService(repo domain.Repository, authz Authorizer, audit AuditLogger, storageProvider storage.ObjectStorage) *Service {
	return &Service{
		repo:    repo,
		authz:   authz,
		audit:   audit,
		storage: storageProvider,
	}
}

type UploadInput struct {
	TenantID         string
	ActorUserID      string
	OwnerType        string
	OwnerID          string
	OriginalFilename string
	File             io.Reader
}

func (s *Service) Upload(ctx context.Context, in UploadInput) (domain.Attachment, error) {
	ownerType, err := normalizeOwnerType(in.OwnerType)
	if err != nil {
		return domain.Attachment{}, err
	}
	if err := s.ensureAccess(ctx, ownerType, true); err != nil {
		return domain.Attachment{}, err
	}
	if err := s.repo.EnsureOwnerExists(ctx, in.TenantID, ownerType, in.OwnerID); err != nil {
		return domain.Attachment{}, err
	}

	detectedMIME, replayableStream, err := detectMimeAndReplay(in.File)
	if err != nil {
		return domain.Attachment{}, err
	}
	if _, ok := allowedMimeTypes[detectedMIME]; !ok {
		return domain.Attachment{}, domain.ErrInvalidFileType
	}
	if !isAllowedFileExtension(in.OriginalFilename, detectedMIME) {
		return domain.Attachment{}, domain.ErrInvalidFileType
	}

	fileID := uuid.NewString()
	storedFilename := fileID + "_" + sanitizeFilename(in.OriginalFilename)
	objectKey := filepath.ToSlash(filepath.Join(
		in.TenantID,
		"finance",
		string(ownerType),
		in.OwnerID,
		storedFilename,
	))

	sizeLimiter := &io.LimitedReader{
		R: replayableStream,
		N: maxAttachmentBytes + 1,
	}

	putOut, err := s.storage.Put(ctx, storage.PutObjectInput{
		TenantID:    in.TenantID,
		Module:      "finance." + string(ownerType),
		ObjectKey:   objectKey,
		ContentType: detectedMIME,
		Body:        sizeLimiter,
	})
	if err != nil {
		return domain.Attachment{}, err
	}

	actualSize := (maxAttachmentBytes + 1) - sizeLimiter.N
	if sizeLimiter.N <= 0 {
		_ = s.storage.Delete(ctx, storage.GetObjectInput{ObjectKey: putOut.ObjectKey})
		return domain.Attachment{}, domain.ErrFileTooLarge
	}

	attachment, err := s.repo.CreateAttachmentRecord(ctx, domain.CreateAttachmentRecordInput{
		TenantID:         in.TenantID,
		OwnerType:        ownerType,
		OwnerID:          in.OwnerID,
		FileID:           fileID,
		Provider:         putOut.Provider,
		ObjectKey:        putOut.ObjectKey,
		OriginalFilename: in.OriginalFilename,
		StoredFilename:   storedFilename,
		MimeType:         detectedMIME,
		SizeBytes:        actualSize,
		UploadedBy:       in.ActorUserID,
	})
	if err != nil {
		_ = s.storage.Delete(ctx, storage.GetObjectInput{ObjectKey: putOut.ObjectKey})
		return domain.Attachment{}, err
	}

	_ = s.audit.Write(ctx, auditAction(ownerType, "upload"), "finance_entity_attachment", attachment.ID, nil, attachment)
	return attachment, nil
}

type ListInput struct {
	TenantID  string
	OwnerType string
	OwnerID   string
}

func (s *Service) List(ctx context.Context, in ListInput) ([]domain.Attachment, error) {
	ownerType, err := normalizeOwnerType(in.OwnerType)
	if err != nil {
		return nil, err
	}
	if err := s.ensureAccess(ctx, ownerType, false); err != nil {
		return nil, err
	}
	if err := s.repo.EnsureOwnerExists(ctx, in.TenantID, ownerType, in.OwnerID); err != nil {
		return nil, err
	}
	return s.repo.ListByOwner(ctx, in.TenantID, ownerType, in.OwnerID)
}

type DownloadOutput struct {
	Attachment domain.Attachment
	Reader     io.ReadCloser
}

func (s *Service) Download(ctx context.Context, tenantID, ownerTypeRaw, ownerID, attachmentID string) (DownloadOutput, error) {
	ownerType, err := normalizeOwnerType(ownerTypeRaw)
	if err != nil {
		return DownloadOutput{}, err
	}
	if err := s.ensureAccess(ctx, ownerType, false); err != nil {
		return DownloadOutput{}, err
	}

	attachment, err := s.repo.GetAttachmentByID(ctx, tenantID, ownerType, ownerID, attachmentID)
	if err != nil {
		return DownloadOutput{}, err
	}
	if err := validateAttachmentScanStatus(attachment.ScanStatus); err != nil {
		return DownloadOutput{}, err
	}

	reader, err := s.storage.Open(ctx, storage.GetObjectInput{ObjectKey: attachment.ObjectKey})
	if err != nil {
		return DownloadOutput{}, err
	}

	_ = s.audit.Write(ctx, auditAction(ownerType, "download"), "finance_entity_attachment", attachment.ID, nil, map[string]any{
		"downloaded_at": time.Now().UTC(),
	})

	return DownloadOutput{
		Attachment: attachment,
		Reader:     reader,
	}, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, actorUserID, ownerTypeRaw, ownerID, attachmentID string) error {
	ownerType, err := normalizeOwnerType(ownerTypeRaw)
	if err != nil {
		return err
	}
	if err := s.ensureAccess(ctx, ownerType, true); err != nil {
		return err
	}

	attachment, err := s.repo.SoftDeleteAttachment(ctx, tenantID, ownerType, ownerID, attachmentID, actorUserID)
	if err != nil {
		return err
	}

	_ = s.storage.Delete(ctx, storage.GetObjectInput{ObjectKey: attachment.ObjectKey})
	_ = s.audit.Write(ctx, auditAction(ownerType, "delete"), "finance_entity_attachment", attachment.ID, attachment, map[string]any{
		"deleted": true,
	})
	return nil
}

func (s *Service) ensureAccess(ctx context.Context, ownerType domain.OwnerType, write bool) error {
	var featureCode string
	permissionCodes := make([]string, 0, 3)

	switch ownerType {
	case domain.OwnerTypeSavings:
		if err := s.authz.EnsureModule(ctx, "finance.savings"); err != nil {
			return err
		}
		if write {
			featureCode = "finance.savings.write"
			permissionCodes = append(permissionCodes, "finance.savings.attach", "finance.savings.update", "finance.savings.create")
		} else {
			featureCode = "finance.savings.read"
			permissionCodes = append(permissionCodes, "finance.savings.attachment.read", "finance.savings.read")
		}
	case domain.OwnerTypeBudgets:
		if err := s.authz.EnsureModule(ctx, "finance.budgets"); err != nil {
			return err
		}
		if write {
			featureCode = "finance.budgets.write"
			permissionCodes = append(permissionCodes, "finance.budgets.attach", "finance.budgets.update", "finance.budgets.create")
		} else {
			featureCode = "finance.budgets.read"
			permissionCodes = append(permissionCodes, "finance.budgets.attachment.read", "finance.budgets.read")
		}
	case domain.OwnerTypeReminders:
		if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
			return err
		}
		if write {
			featureCode = "finance.reminders.write"
			permissionCodes = append(permissionCodes, "finance.reminders.attach", "finance.reminders.update", "finance.reminders.create")
		} else {
			featureCode = "finance.reminders.read"
			permissionCodes = append(permissionCodes, "finance.reminders.attachment.read", "finance.reminders.read")
		}
	default:
		return domain.ErrInvalidOwnerType
	}

	if err := s.authz.EnsureFeature(ctx, featureCode); err != nil {
		return err
	}
	return s.ensureAnyPermission(ctx, permissionCodes...)
}

func (s *Service) ensureAnyPermission(ctx context.Context, permissionCodes ...string) error {
	var deniedErr error
	for _, permissionCode := range permissionCodes {
		if strings.TrimSpace(permissionCode) == "" {
			continue
		}
		err := s.authz.EnsurePermission(ctx, permissionCode)
		if err == nil {
			return nil
		}
		if errors.Is(err, access.ErrPermissionDenied) {
			deniedErr = err
			continue
		}
		return err
	}
	if deniedErr != nil {
		return deniedErr
	}
	return access.ErrPermissionDenied
}

func auditAction(ownerType domain.OwnerType, action string) string {
	return "finance." + string(ownerType) + ".attachment." + action
}

func normalizeOwnerType(raw string) (domain.OwnerType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(domain.OwnerTypeSavings):
		return domain.OwnerTypeSavings, nil
	case string(domain.OwnerTypeBudgets):
		return domain.OwnerTypeBudgets, nil
	case string(domain.OwnerTypeReminders):
		return domain.OwnerTypeReminders, nil
	default:
		return "", domain.ErrInvalidOwnerType
	}
}

func sanitizeFilename(name string) string {
	safe := strings.ReplaceAll(name, " ", "_")
	safe = strings.ReplaceAll(safe, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	if safe == "" {
		return "attachment.bin"
	}
	return safe
}

func detectMimeAndReplay(file io.Reader) (string, io.Reader, error) {
	header := make([]byte, 512)
	n, err := io.ReadFull(file, header)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", nil, err
	}
	if n == 0 {
		return "", nil, domain.ErrInvalidFileType
	}

	detected := normalizeMIME(http.DetectContentType(header[:n]))
	return detected, io.MultiReader(bytes.NewReader(header[:n]), file), nil
}

func normalizeMIME(raw string) string {
	mainType := strings.TrimSpace(strings.Split(raw, ";")[0])
	return strings.ToLower(mainType)
}

func isAllowedFileExtension(filename, mimeType string) bool {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	allowedExts, ok := allowedExtensionsByMIME[mimeType]
	if !ok {
		return false
	}
	_, ok = allowedExts[ext]
	return ok
}

func validateAttachmentScanStatus(status string) error {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "clean":
		return nil
	case "infected":
		return domain.ErrAttachmentInfected
	default:
		return domain.ErrAttachmentScanPending
	}
}
