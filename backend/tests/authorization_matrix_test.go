package tests

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	transactiondomain "pekan/backend/internal/modules/finance/transactions/domain"
	transactionusecase "pekan/backend/internal/modules/finance/transactions/usecase"
	receiptdomain "pekan/backend/internal/modules/finance/receipts/domain"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/storage"
	"pekan/backend/internal/platform/tenancy"
)

func TestTransactionCreateAuthorizationMatrix(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name        string
		modules     []string
		features    []string
		permissions []string
		wantErr     error
	}

	cases := []testCase{
		{
			name:        "allowed",
			modules:     []string{"finance"},
			features:    []string{"finance.transactions.write"},
			permissions: []string{"finance.transactions.create"},
			wantErr:     nil,
		},
		{
			name:        "module_disabled",
			modules:     []string{},
			features:    []string{"finance.transactions.write"},
			permissions: []string{"finance.transactions.create"},
			wantErr:     access.ErrModuleDisabled,
		},
		{
			name:        "feature_locked",
			modules:     []string{"finance"},
			features:    []string{},
			permissions: []string{"finance.transactions.create"},
			wantErr:     access.ErrFeatureLocked,
		},
		{
			name:        "permission_denied",
			modules:     []string{"finance"},
			features:    []string{"finance.transactions.write"},
			permissions: []string{},
			wantErr:     access.ErrPermissionDenied,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := &fakeTransactionRepo{}
			svc := transactionusecase.NewService(repo, access.NewAuthorizer(), fakeAuditLogger{})

			ctx := withTenantContext(tc.modules, tc.features, tc.permissions, "tenant-a")
			_, err := svc.Create(ctx, transactionusecase.CreateInput{
				TenantID:        "tenant-a",
				ActorUserID:     "user-a",
				AccountID:       "account-1",
				Type:            transactiondomain.TransactionTypeExpense,
				AmountMinor:     1000,
				Currency:        "IDR",
				TransactionDate: time.Now().UTC(),
			})
			if !errorIs(err, tc.wantErr) {
				t.Fatalf("unexpected error: got=%v want=%v", err, tc.wantErr)
			}
		})
	}
}

func TestTransactionGetByIDBOLA(t *testing.T) {
	t.Parallel()

	repo := &fakeTransactionRepo{
		recordTenantID: "tenant-b",
	}
	svc := transactionusecase.NewService(repo, access.NewAuthorizer(), fakeAuditLogger{})
	ctx := withTenantContext(
		[]string{"finance"},
		[]string{"finance.transactions.read"},
		[]string{"finance.transactions.read"},
		"tenant-a",
	)

	_, err := svc.GetByID(ctx, "tenant-a", "trx-1")
	if !errors.Is(err, transactiondomain.ErrTransactionNotFound) {
		t.Fatalf("expected ErrTransactionNotFound for cross-tenant BOLA check, got=%v", err)
	}
}

func TestAttachmentAuthorizationMatrix(t *testing.T) {
	t.Parallel()

	attachmentSvc := transactionusecase.NewAttachmentService(
		&fakeAttachmentRepo{},
		&fakeReceiptsRepo{},
		access.NewAuthorizer(),
		fakeAuditLogger{},
		fakeStorage{},
	)

	type testCase struct {
		name        string
		modules     []string
		features    []string
		permissions []string
		wantErr     error
	}
	cases := []testCase{
		{
			name:        "upload_allowed",
			modules:     []string{"finance"},
			features:    []string{"finance.transactions.write"},
			permissions: []string{"finance.transactions.attach"},
			wantErr:     nil,
		},
		{
			name:        "upload_permission_denied",
			modules:     []string{"finance"},
			features:    []string{"finance.transactions.write"},
			permissions: []string{},
			wantErr:     access.ErrPermissionDenied,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := withTenantContext(tc.modules, tc.features, tc.permissions, "tenant-a")
			_, err := attachmentSvc.Upload(ctx, transactionusecase.UploadAttachmentInput{
				TenantID:         "tenant-a",
				ActorUserID:      "user-a",
				TransactionID:    "trx-1",
				OriginalFilename: "receipt.png",
				MimeType:         "image/png",
				SizeBytes:        128,
				File:             bytes.NewBufferString("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\x0aIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\x0d\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82"),
			})
			if !errorIs(err, tc.wantErr) {
				t.Fatalf("unexpected error: got=%v want=%v", err, tc.wantErr)
			}
		})
	}
}

type fakeTransactionRepo struct {
	recordTenantID string
}

func (f *fakeTransactionRepo) Create(_ context.Context, trx transactiondomain.Transaction) (transactiondomain.Transaction, error) {
	trx.ID = "trx-1"
	return trx, nil
}

func (f *fakeTransactionRepo) GetByID(_ context.Context, tenantID, _ string) (transactiondomain.Transaction, error) {
	if f.recordTenantID != "" && tenantID != f.recordTenantID {
		return transactiondomain.Transaction{}, transactiondomain.ErrTransactionNotFound
	}
	return transactiondomain.Transaction{
		ID:       "trx-1",
		TenantID: tenantID,
	}, nil
}

func (f *fakeTransactionRepo) List(_ context.Context, filter transactiondomain.ListFilter) ([]transactiondomain.Transaction, int64, error) {
	return []transactiondomain.Transaction{{ID: "trx-1", TenantID: filter.TenantID}}, 1, nil
}

func (f *fakeTransactionRepo) Update(_ context.Context, trx transactiondomain.Transaction) (transactiondomain.Transaction, error) {
	return trx, nil
}

func (f *fakeTransactionRepo) SoftDelete(_ context.Context, _, _, _ string) error {
	return nil
}

func (f *fakeTransactionRepo) ResolveCategoryID(_ context.Context, _, _ string, categoryID, _ *string, _ transactiondomain.TransactionType) (*string, error) {
	return categoryID, nil
}

func (f *fakeTransactionRepo) ValidateReferences(_ context.Context, _, _ string, _ *string, _ transactiondomain.TransactionType) error {
	return nil
}

func (f *fakeTransactionRepo) ValidateSavingsGoals(_ context.Context, _ string, _ []string) error {
	return nil
}

func (f *fakeTransactionRepo) ReplaceSavingsLinks(_ context.Context, _, _, _ string, _ int64, _ []string) error {
	return nil
}

func (f *fakeTransactionRepo) ListSavingsLinks(_ context.Context, _ string, _ []string) (map[string][]string, map[string][]string, error) {
	return map[string][]string{}, map[string][]string{}, nil
}

func (f *fakeTransactionRepo) ListSavingsAllocationsByTransaction(_ context.Context, _, _ string) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (f *fakeTransactionRepo) AdjustSavingsCurrentAmounts(_ context.Context, _, _ string, _ map[string]int64) error {
	return nil
}

func (f *fakeTransactionRepo) ListBySavingsID(_ context.Context, _, _ string) ([]transactiondomain.Transaction, error) {
	return []transactiondomain.Transaction{}, nil
}

func (f *fakeTransactionRepo) ListItems(_ context.Context, _, _ string) ([]transactiondomain.TransactionItem, error) {
	return []transactiondomain.TransactionItem{}, nil
}

func (f *fakeTransactionRepo) ReplaceItems(_ context.Context, _, _, _ string, _ []transactiondomain.TransactionItem) error {
	return nil
}

func (f *fakeTransactionRepo) ListItemsByTransactionIDs(_ context.Context, _ string, _ []string) (map[string][]transactiondomain.TransactionItem, error) {
	return map[string][]transactiondomain.TransactionItem{}, nil
}

type fakeAttachmentRepo struct{}

func (f *fakeAttachmentRepo) EnsureTransactionExists(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeAttachmentRepo) CreateAttachmentRecord(_ context.Context, in transactiondomain.CreateAttachmentRecordInput) (transactiondomain.Attachment, error) {
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
		CreatedAt:        time.Now().UTC(),
	}, nil
}

func (f *fakeAttachmentRepo) GetAttachmentByID(_ context.Context, _, _, _ string) (transactiondomain.Attachment, error) {
	return transactiondomain.Attachment{
		ID:               "att-1",
		MimeType:         "image/png",
		OriginalFilename: "receipt.png",
		ObjectKey:        "tenant-a/receipt.png",
		ScanStatus:       "clean",
	}, nil
}

func (f *fakeAttachmentRepo) ListAttachmentsByTransaction(_ context.Context, _, _ string) ([]transactiondomain.Attachment, error) {
	return []transactiondomain.Attachment{}, nil
}

func (f *fakeAttachmentRepo) SetAttachmentScanStatus(_ context.Context, _, _, _, _ string) error {
	return nil
}

type fakeReceiptsRepo struct{}

func (f *fakeReceiptsRepo) GetReceiptScanByID(_ context.Context, _, _ string) (receiptdomain.ReceiptScan, error) {
	return receiptdomain.ReceiptScan{}, errors.New("not implemented")
}

type fakeStorage struct{}

func (f fakeStorage) Put(_ context.Context, in storage.PutObjectInput) (storage.PutObjectOutput, error) {
	return storage.PutObjectOutput{
		Provider:  "local",
		ObjectKey: in.ObjectKey,
	}, nil
}
func (f fakeStorage) Delete(_ context.Context, _ storage.GetObjectInput) error { return nil }
func (f fakeStorage) Open(_ context.Context, _ storage.GetObjectInput) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBufferString("x")), nil
}

type fakeAuditLogger struct{}

func (fakeAuditLogger) Write(_ context.Context, _, _, _ string, _, _ any) error { return nil }

func withTenantContext(modules, features, permissions []string, tenantID string) context.Context {
	return tenancy.WithContext(context.Background(), tenancy.Context{
		UserID:      "user-a",
		TenantID:    tenantID,
		Email:       "user@tenant.test",
		Modules:     toSet(modules),
		Features:    toSet(features),
		Permissions: toSet(permissions),
	})
}

func toSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func errorIs(err error, target error) bool {
	if target == nil {
		return err == nil
	}
	return errors.Is(err, target)
}
