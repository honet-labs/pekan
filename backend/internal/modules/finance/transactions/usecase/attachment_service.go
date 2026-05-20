package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/transactions/domain"
	receiptDomain "pekan/backend/internal/modules/finance/receipts/domain"
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

type ReceiptsRepository interface {
	GetReceiptScanByID(ctx context.Context, tenantID, scanID string) (receiptDomain.ReceiptScan, error)
}

type AttachmentService struct {
	repo         domain.AttachmentRepository
	receiptsRepo ReceiptsRepository
	authz        Authorizer
	audit        AuditLogger
	storage      storage.ObjectStorage
}

func NewAttachmentService(repo domain.AttachmentRepository, receiptsRepo ReceiptsRepository, authz Authorizer, audit AuditLogger, storage storage.ObjectStorage) *AttachmentService {
	return &AttachmentService{
		repo:         repo,
		receiptsRepo: receiptsRepo,
		authz:        authz,
		audit:        audit,
		storage:      storage,
	}
}

type UploadAttachmentInput struct {
	TenantID         string
	ActorUserID      string
	TransactionID    string
	OriginalFilename string
	MimeType         string
	SizeBytes        int64
	File             io.Reader
}

func (s *AttachmentService) Upload(ctx context.Context, in UploadAttachmentInput) (domain.Attachment, error) {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return domain.Attachment{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.write"); err != nil {
		return domain.Attachment{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.transactions.attach", "finance.transactions.update", "finance.transactions.create"); err != nil {
		return domain.Attachment{}, err
	}
	if err := s.repo.EnsureTransactionExists(ctx, in.TenantID, in.TransactionID); err != nil {
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
		"transactions",
		in.TransactionID,
		storedFilename,
	))

	sizeLimiter := &io.LimitedReader{
		R: replayableStream,
		N: maxAttachmentBytes + 1,
	}

	putOut, err := s.storage.Put(ctx, storage.PutObjectInput{
		TenantID:    in.TenantID,
		Module:      "finance.transactions",
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
		TransactionID:    in.TransactionID,
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

	_ = s.audit.Write(ctx, "finance.transaction.attachment.upload", "finance_transaction_attachment", attachment.ID, nil, attachment)
	return attachment, nil
}

func (s *AttachmentService) List(ctx context.Context, tenantID, transactionID string) ([]domain.Attachment, error) {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.read"); err != nil {
		return nil, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.transactions.attachment.read", "finance.transactions.read"); err != nil {
		return nil, err
	}

	return s.repo.ListAttachmentsByTransaction(ctx, tenantID, transactionID)
}

type DownloadAttachmentOutput struct {
	Attachment domain.Attachment
	Reader     io.ReadCloser
}

type SetAttachmentScanStatusInput struct {
	TenantID      string
	TransactionID string
	AttachmentID  string
	ScanStatus    string
	Reason        *string
}

func (s *AttachmentService) AttachFromScan(ctx context.Context, tenantID, actorUserID, transactionID, scanID string) (domain.Attachment, error) {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return domain.Attachment{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.write"); err != nil {
		return domain.Attachment{}, err
	}

	scan, err := s.receiptsRepo.GetReceiptScanByID(ctx, tenantID, scanID)
	if err != nil {
		return domain.Attachment{}, err
	}

	srcKey := fmt.Sprintf("%s/finance/receipt-scan/%s", tenantID, scanID)
	src, err := s.storage.Open(ctx, storage.GetObjectInput{ObjectKey: srcKey})
	if err != nil {
		return domain.Attachment{}, err
	}
	defer src.Close()

	cr := &countingReader{r: src}

	fileID := uuid.NewString()
	originalFilename := scan.OriginalFilename
	if originalFilename == "" {
		originalFilename = "receipt.jpg"
	}
	storedFilename := fileID + "_" + sanitizeFilename(originalFilename)
	destKey := filepath.ToSlash(filepath.Join(
		tenantID,
		"finance",
		"transactions",
		transactionID,
		storedFilename,
	))

	putOut, err := s.storage.Put(ctx, storage.PutObjectInput{
		TenantID:    tenantID,
		Module:      "finance.transactions",
		ObjectKey:   destKey,
		ContentType: scan.MimeType,
		Body:        cr,
	})
	if err != nil {
		return domain.Attachment{}, err
	}

	attachment, err := s.repo.CreateAttachmentRecord(ctx, domain.CreateAttachmentRecordInput{
		TenantID:         tenantID,
		TransactionID:    transactionID,
		FileID:           fileID,
		Provider:         putOut.Provider,
		ObjectKey:        putOut.ObjectKey,
		OriginalFilename: originalFilename,
		StoredFilename:   storedFilename,
		MimeType:         scan.MimeType,
		SizeBytes:        cr.n,
		UploadedBy:       actorUserID,
	})
	if err != nil {
		_ = s.storage.Delete(ctx, storage.GetObjectInput{ObjectKey: putOut.ObjectKey})
		return domain.Attachment{}, err
	}

	_ = s.audit.Write(ctx, "finance.transaction.attachment.from_scan", "finance_transaction_attachment", attachment.ID, nil, attachment)
	return attachment, nil
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (n int, err error) {
	n, err = c.r.Read(p)
	c.n += int64(n)
	return
}

func (s *AttachmentService) Download(ctx context.Context, tenantID, transactionID, attachmentID string) (DownloadAttachmentOutput, error) {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return DownloadAttachmentOutput{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.read"); err != nil {
		return DownloadAttachmentOutput{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.transactions.attachment.read", "finance.transactions.read"); err != nil {
		return DownloadAttachmentOutput{}, err
	}

	attachment, err := s.repo.GetAttachmentByID(ctx, tenantID, transactionID, attachmentID)
	if err != nil {
		return DownloadAttachmentOutput{}, err
	}
	if err := validateAttachmentScanStatus(attachment.ScanStatus); err != nil {
		return DownloadAttachmentOutput{}, err
	}

	reader, err := s.storage.Open(ctx, storage.GetObjectInput{ObjectKey: attachment.ObjectKey})
	if err != nil {
		return DownloadAttachmentOutput{}, err
	}

	_ = s.audit.Write(ctx, "finance.transaction.attachment.download", "finance_transaction_attachment", attachment.ID, nil, map[string]any{
		"downloaded_at": time.Now().UTC(),
	})

	return DownloadAttachmentOutput{
		Attachment: attachment,
		Reader:     reader,
	}, nil
}

func (s *AttachmentService) SetScanStatus(ctx context.Context, in SetAttachmentScanStatusInput) error {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.write"); err != nil {
		return err
	}
	if err := s.ensureAnyPermission(ctx, "finance.transactions.scan.manage", "finance.transactions.update"); err != nil {
		return err
	}

	normalizedStatus := normalizeScanStatus(in.ScanStatus)
	if !isAllowedScanStatus(normalizedStatus) {
		return domain.ErrInvalidScanStatus
	}

	if err := s.repo.SetAttachmentScanStatus(ctx, in.TenantID, in.TransactionID, in.AttachmentID, normalizedStatus); err != nil {
		return err
	}

	_ = s.audit.Write(ctx, "finance.transaction.attachment.scan_status.set", "finance_transaction_attachment", in.AttachmentID, nil, map[string]any{
		"scan_status": normalizedStatus,
		"reason":      in.Reason,
	})
	return nil
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
	switch normalizeScanStatus(status) {
	case "clean":
		return nil
	case "infected":
		return domain.ErrAttachmentInfected
	default:
		return domain.ErrAttachmentScanPending
	}
}

func normalizeScanStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func isAllowedScanStatus(status string) bool {
	switch status {
	case "pending", "clean", "infected", "failed":
		return true
	default:
		return false
	}
}

func (s *AttachmentService) ensureAnyPermission(ctx context.Context, permissionCodes ...string) error {
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
