package tests

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	transactiondomain "pekan/backend/internal/modules/finance/transactions/domain"
	transactionusecase "pekan/backend/internal/modules/finance/transactions/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/storage"
)

func TestAttachmentUploadRejectsMIMEExtensionMismatch(t *testing.T) {
	t.Parallel()

	svc := transactionusecase.NewAttachmentService(
		&secureAttachmentRepo{},
		&fakeReceiptsRepo{},
		access.NewAuthorizer(),
		fakeAuditLogger{},
		readingStorage{},
	)

	ctx := withTenantContext(
		[]string{"finance"},
		[]string{"finance.transactions.write"},
		[]string{"finance.transactions.attach"},
		"tenant-a",
	)

	_, err := svc.Upload(ctx, transactionusecase.UploadAttachmentInput{
		TenantID:         "tenant-a",
		ActorUserID:      "user-a",
		TransactionID:    "trx-1",
		OriginalFilename: "receipt.exe",
		MimeType:         "application/pdf",
		File:             bytes.NewBufferString("%PDF-1.7\nsample"),
	})
	if !errors.Is(err, transactiondomain.ErrInvalidFileType) {
		t.Fatalf("expected ErrInvalidFileType, got=%v", err)
	}
}

func TestAttachmentDownloadBlockedWhenScanPending(t *testing.T) {
	t.Parallel()

	svc := transactionusecase.NewAttachmentService(
		&secureAttachmentRepo{
			attachment: transactiondomain.Attachment{
				ID:               "att-1",
				TenantID:         "tenant-a",
				TransactionID:    "trx-1",
				MimeType:         "application/pdf",
				OriginalFilename: "receipt.pdf",
				ObjectKey:        "tenant-a/finance/transactions/trx-1/receipt.pdf",
				ScanStatus:       "pending",
			},
		},
		&fakeReceiptsRepo{},
		access.NewAuthorizer(),
		fakeAuditLogger{},
		readingStorage{},
	)

	ctx := withTenantContext(
		[]string{"finance"},
		[]string{"finance.transactions.read"},
		[]string{"finance.transactions.attachment.read"},
		"tenant-a",
	)

	_, err := svc.Download(ctx, "tenant-a", "trx-1", "att-1")
	if !errors.Is(err, transactiondomain.ErrAttachmentScanPending) {
		t.Fatalf("expected ErrAttachmentScanPending, got=%v", err)
	}
}

func TestAttachmentSetScanStatusRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	svc := transactionusecase.NewAttachmentService(
		&secureAttachmentRepo{},
		&fakeReceiptsRepo{},
		access.NewAuthorizer(),
		fakeAuditLogger{},
		readingStorage{},
	)

	ctx := withTenantContext(
		[]string{"finance"},
		[]string{"finance.transactions.write"},
		[]string{"finance.transactions.scan.manage"},
		"tenant-a",
	)

	err := svc.SetScanStatus(ctx, transactionusecase.SetAttachmentScanStatusInput{
		TenantID:      "tenant-a",
		TransactionID: "trx-1",
		AttachmentID:  "att-1",
		ScanStatus:    "unknown",
	})
	if !errors.Is(err, transactiondomain.ErrInvalidScanStatus) {
		t.Fatalf("expected ErrInvalidScanStatus, got=%v", err)
	}
}

type secureAttachmentRepo struct {
	attachment transactiondomain.Attachment
}

func (r *secureAttachmentRepo) EnsureTransactionExists(_ context.Context, _, _ string) error {
	return nil
}

func (r *secureAttachmentRepo) CreateAttachmentRecord(_ context.Context, in transactiondomain.CreateAttachmentRecordInput) (transactiondomain.Attachment, error) {
	return transactiondomain.Attachment{
		ID:               "att-1",
		TenantID:         in.TenantID,
		TransactionID:    in.TransactionID,
		FileID:           in.FileID,
		Provider:         in.Provider,
		ObjectKey:        in.ObjectKey,
		OriginalFilename: in.OriginalFilename,
		StoredFilename:   in.StoredFilename,
		MimeType:         in.MimeType,
		SizeBytes:        in.SizeBytes,
		ScanStatus:       "pending",
	}, nil
}

func (r *secureAttachmentRepo) GetAttachmentByID(_ context.Context, _, _, _ string) (transactiondomain.Attachment, error) {
	if r.attachment.ID == "" {
		return transactiondomain.Attachment{}, transactiondomain.ErrAttachmentNotFound
	}
	return r.attachment, nil
}

func (r *secureAttachmentRepo) ListAttachmentsByTransaction(_ context.Context, _, _ string) ([]transactiondomain.Attachment, error) {
	return []transactiondomain.Attachment{}, nil
}

func (r *secureAttachmentRepo) SetAttachmentScanStatus(_ context.Context, _, _, _, _ string) error {
	return nil
}

type readingStorage struct{}

func (readingStorage) Put(_ context.Context, in storage.PutObjectInput) (storage.PutObjectOutput, error) {
	_, err := io.Copy(io.Discard, in.Body)
	if err != nil {
		return storage.PutObjectOutput{}, err
	}
	return storage.PutObjectOutput{
		Provider:  "local",
		ObjectKey: in.ObjectKey,
	}, nil
}

func (readingStorage) Open(_ context.Context, _ storage.GetObjectInput) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBufferString("file")), nil
}

func (readingStorage) Delete(_ context.Context, _ storage.GetObjectInput) error {
	return nil
}


