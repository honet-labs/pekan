package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	authhttp "pekan/backend/internal/modules/core/auth/delivery/http"
	authdomain "pekan/backend/internal/modules/core/auth/domain"
	authusecase "pekan/backend/internal/modules/core/auth/usecase"
	masterhttp "pekan/backend/internal/modules/finance/master/delivery/http"
	masterdomain "pekan/backend/internal/modules/finance/master/domain"
	masterusecase "pekan/backend/internal/modules/finance/master/usecase"
	transactionhttp "pekan/backend/internal/modules/finance/transactions/delivery/http"
	transactiondomain "pekan/backend/internal/modules/finance/transactions/domain"
	transactionusecase "pekan/backend/internal/modules/finance/transactions/usecase"
	platformauth "pekan/backend/internal/platform/auth"
	"pekan/backend/internal/platform/middleware"
)

func TestHTTPEndpointsAuthLoginSuccess(t *testing.T) {
	t.Parallel()

	jwtSvc := platformauth.NewService("test", "secret", 15*time.Minute)
	router := buildTestRouter(jwtSvc)

	body := map[string]any{
		"email":    "owner@pekan.local",
		"password": "password",
		"tenant_id": "tenant-a",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHTTPEndpointsProtectedRequiresAuth(t *testing.T) {
	t.Parallel()

	jwtSvc := platformauth.NewService("test", "secret", 15*time.Minute)
	router := buildTestRouter(jwtSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/accounts", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHTTPEndpointsTenantHeaderMismatch(t *testing.T) {
	t.Parallel()

	jwtSvc := platformauth.NewService("test", "secret", 15*time.Minute)
	router := buildTestRouter(jwtSvc)
	token := issueAccessToken(t, jwtSvc, "tenant-a")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/accounts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", "tenant-b")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHTTPEndpointsTransactionCreateInvalidDate(t *testing.T) {
	t.Parallel()

	jwtSvc := platformauth.NewService("test", "secret", 15*time.Minute)
	router := buildTestRouter(jwtSvc)
	token := issueAccessToken(t, jwtSvc, "tenant-a")

	raw := []byte(`{
	  "account_id":"acc-1",
	  "type":"expense",
	  "amount_minor":1000,
	  "currency":"IDR",
	  "transaction_date":"2026/01/01"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/transactions", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func buildTestRouter(jwtSvc *platformauth.Service) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recovery)

	authHandler := authhttp.NewHandler(fakeAuthService{})
	masterHandler := masterhttp.NewHandler(fakeMasterService{})
	transactionHandler := transactionhttp.NewHandler(fakeTransactionService{}, fakeAttachmentService{})

	router.Route("/api/v1", func(r chi.Router) {
		authHandler.RegisterPublicRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtSvc, fakeSessionStore{}))
			r.Use(middleware.Tenant)
			r.Use(middleware.AuditContext)
			masterHandler.RegisterRoutes(r)
			transactionHandler.RegisterRoutes(r)
		})
	})
	return router
}

type fakeSessionStore struct{}

func (fakeSessionStore) IsAccountLocked(_ context.Context, _, _ string) (bool, time.Duration, error) {
	return false, 0, nil
}
func (fakeSessionStore) RecordFailedLogin(_ context.Context, _, _ string) (bool, time.Duration, error) {
	return false, 0, nil
}
func (fakeSessionStore) ClearFailedLogin(_ context.Context, _, _ string) error { return nil }
func (fakeSessionStore) RevokeToken(_ context.Context, _ string, _ time.Duration) error {
	return nil
}
func (fakeSessionStore) IsTokenRevoked(_ context.Context, _ string) (bool, error) { return false, nil }


func issueAccessToken(t *testing.T, jwtSvc *platformauth.Service, tenantID string) string {
	t.Helper()
	token, _, err := jwtSvc.IssueAccessToken(platformauth.IssueAccessTokenInput{
		UserID:      "user-a",
		Email:       "owner@pekan.local",
		TenantID:    tenantID,
		SessionID:   "session-a",
		Permissions: []string{"finance.accounts.read", "finance.transactions.create", "finance.transactions.read"},
		Features:    []string{"finance.masterdata.read", "finance.transactions.read", "finance.transactions.write"},
		Modules:     []string{"finance", "finance.masterdata"},
	})
	if err != nil {
		t.Fatalf("issue token error: %v", err)
	}
	return token
}

type fakeAuthService struct{}

func (fakeAuthService) Login(_ context.Context, in authusecase.LoginInput) (authusecase.LoginOutput, error) {
	return authusecase.LoginOutput{
		Tokens: authusecase.TokenOutput{
			AccessToken:           "access",
			RefreshToken:          "refresh",
			AccessTokenExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
			RefreshTokenExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		},
		User: authdomain.User{
			ID:    "user-a",
			Email: in.Email,
		},
		Membership: authdomain.Membership{
			ID:       "membership-a",
			TenantID: "tenant-a",
			UserID:   "user-a",
			Status:   "active",
		},
		Access: authdomain.AccessProfile{
			Permissions: []string{"finance.transactions.read"},
			Features:    []string{"finance.transactions.read"},
			Modules:     []string{"finance"},
		},
	}, nil
}

func (fakeAuthService) Refresh(_ context.Context, _ authusecase.RefreshInput) (authusecase.RefreshOutput, error) {
	return authusecase.RefreshOutput{}, nil
}

func (fakeAuthService) Logout(_ context.Context, _ string) error           { return nil }
func (fakeAuthService) LogoutAll(_ context.Context, _ string) error        { return nil }
func (fakeAuthService) RequestPasswordReset(_ context.Context, _, _, _ string) error {
	return nil
}
func (fakeAuthService) ForgotTenant(_ context.Context, _, _ string) error { return nil }
func (fakeAuthService) ResetPassword(_ context.Context, _, _, _ string) error { return nil }

func (fakeAuthService) GetContext(_ context.Context, _, _ string) (authusecase.ContextOutput, error) {
	return authusecase.ContextOutput{}, nil
}

func (fakeAuthService) SwitchTenant(_ context.Context, _ authusecase.SwitchTenantInput) (authusecase.TokenOutput, error) {
	return authusecase.TokenOutput{}, nil
}

func (fakeAuthService) GetProfile(_ context.Context, _ string) (authdomain.UserProfile, error) {
	return authdomain.UserProfile{
		UserID:   "user-a",
		Username: "testuser",
	}, nil
}

func (fakeAuthService) UpdateProfile(_ context.Context, in authusecase.UpdateProfileInput) (authdomain.UserProfile, error) {
	return authdomain.UserProfile{
		UserID:   in.UserID,
		Username: in.Username,
		Phone:    in.Phone,
		Address:  in.Address,
	}, nil
}

func (fakeAuthService) RegisterInit(_ context.Context, _ authusecase.RegisterInitInput) (authusecase.RegisterInitOutput, error) {
	return authusecase.RegisterInitOutput{}, nil
}

func (fakeAuthService) RegisterVerify(_ context.Context, _ authusecase.RegisterVerifyInput) error {
	return nil
}

type fakeMasterService struct{}

func (fakeMasterService) CreateAccount(_ context.Context, in masterusecase.CreateAccountInput) (masterdomain.Account, error) {
	return masterdomain.Account{
		ID:       "acc-1",
		TenantID: in.TenantID,
		Name:     in.Name,
	}, nil
}
func (fakeMasterService) ListAccounts(_ context.Context, tenantID string) ([]masterdomain.Account, error) {
	return []masterdomain.Account{{ID: "acc-1", TenantID: tenantID, Name: "Cash"}}, nil
}
func (fakeMasterService) CreateCategory(_ context.Context, in masterusecase.CreateCategoryInput) (masterdomain.Category, error) {
	return masterdomain.Category{
		ID:       "cat-1",
		TenantID: in.TenantID,
		Name:     in.Name,
	}, nil
}
func (fakeMasterService) ListCategories(_ context.Context, tenantID string) ([]masterdomain.Category, error) {
	return []masterdomain.Category{{ID: "cat-1", TenantID: tenantID, Name: "Food"}}, nil
}
func (fakeMasterService) DeleteCategory(_ context.Context, _, _ string) error { return nil }
func (fakeMasterService) UpdateCategory(_ context.Context, _, _ string, _ masterusecase.CreateCategoryInput) (masterdomain.Category, error) {
	return masterdomain.Category{}, nil
}
func (fakeMasterService) GetCategory(_ context.Context, _, _ string) (masterdomain.Category, error) {
	return masterdomain.Category{ID: "cat-1", Name: "Food"}, nil
}

type fakeTransactionService struct{}

func (fakeTransactionService) Create(_ context.Context, in transactionusecase.CreateInput) (transactiondomain.Transaction, error) {
	return transactiondomain.Transaction{
		ID:              "trx-1",
		TenantID:        in.TenantID,
		AccountID:       in.AccountID,
		Type:            in.Type,
		AmountMinor:     in.AmountMinor,
		Currency:        in.Currency,
		TransactionDate: in.TransactionDate,
		CreatedBy:       in.ActorUserID,
		UpdatedBy:       in.ActorUserID,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}, nil
}
func (fakeTransactionService) GetByID(_ context.Context, tenantID, transactionID string) (transactiondomain.Transaction, error) {
	return transactiondomain.Transaction{ID: transactionID, TenantID: tenantID}, nil
}
func (fakeTransactionService) List(_ context.Context, in transactionusecase.ListInput) ([]transactiondomain.Transaction, int64, error) {
	return []transactiondomain.Transaction{{ID: "trx-1", TenantID: in.TenantID}}, 1, nil
}
func (fakeTransactionService) Update(_ context.Context, in transactionusecase.UpdateInput) (transactiondomain.Transaction, error) {
	return transactiondomain.Transaction{ID: in.TransactionID, TenantID: in.TenantID}, nil
}
func (fakeTransactionService) Delete(_ context.Context, _, _, _ string) error { return nil }
func (fakeTransactionService) ListBySavingsID(_ context.Context, tenantID, _ string) ([]transactiondomain.Transaction, error) {
	return []transactiondomain.Transaction{{ID: "trx-1", TenantID: tenantID}}, nil
}

type fakeAttachmentService struct{}
func (fakeAttachmentService) List(_ context.Context, tenantID, transactionID string) ([]transactiondomain.Attachment, error) {
	return []transactiondomain.Attachment{}, nil
}
func (fakeAttachmentService) Upload(_ context.Context, _ transactionusecase.UploadAttachmentInput) (transactiondomain.Attachment, error) {
	return transactiondomain.Attachment{ID: "att-1"}, nil
}
func (fakeAttachmentService) AttachFromScan(_ context.Context, _, _, _, _ string) (transactiondomain.Attachment, error) {
	return transactiondomain.Attachment{ID: "att-1"}, nil
}
func (fakeAttachmentService) Download(_ context.Context, _, _, _ string) (transactionusecase.DownloadAttachmentOutput, error) {
	return transactionusecase.DownloadAttachmentOutput{}, nil
}
func (fakeAttachmentService) SetScanStatus(_ context.Context, _ transactionusecase.SetAttachmentScanStatusInput) error {
	return nil
}
