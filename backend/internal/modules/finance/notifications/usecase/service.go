package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"pekan/backend/internal/modules/finance/notifications/domain"
)

type Authorizer interface {
	EnsureModule(ctx context.Context, module string) error
	EnsureFeature(ctx context.Context, feature string) error
	EnsurePermission(ctx context.Context, permission string) error
}

type AuditLogger interface {
	Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error
}

type Service struct {
	repo  domain.Repository
	authz Authorizer
	audit AuditLogger
}

func NewService(repo domain.Repository, authz Authorizer, audit AuditLogger) *Service {
	return &Service{
		repo:  repo,
		authz: authz,
		audit: audit,
	}
}

type CreateInput struct {
	TenantID         string
	ActorUserID      string
	NotificationType string
	Title            string
	Message          string
	Metadata         json.RawMessage
}

func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Notification, error) {
	if err := s.authz.EnsureModule(ctx, "finance.notifications"); err != nil {
		return domain.Notification{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.notifications.write"); err != nil {
		return domain.Notification{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.notifications.create"); err != nil {
		return domain.Notification{}, err
	}
	if strings.TrimSpace(in.Title) == "" {
		return domain.Notification{}, domain.ErrInvalidTitle
	}
	if strings.TrimSpace(in.Message) == "" {
		return domain.Notification{}, domain.ErrInvalidMessage
	}
	if strings.TrimSpace(in.NotificationType) == "" {
		return domain.Notification{}, domain.ErrInvalidType
	}

	now := time.Now().UTC()
	out, err := s.repo.Create(ctx, domain.Notification{
		TenantID:         in.TenantID,
		NotificationType: strings.TrimSpace(in.NotificationType),
		Title:            strings.TrimSpace(in.Title),
		Message:          strings.TrimSpace(in.Message),
		Status:           "unread",
		Metadata:         in.Metadata,
		CreatedBy:        in.ActorUserID,
		CreatedAt:        now,
	})
	if err != nil {
		return domain.Notification{}, err
	}

	_ = s.audit.Write(ctx, "finance.notification.create", "finance_notification", out.ID, nil, out)
	return out, nil
}

type ListInput struct {
	TenantID string
	Status   *string
	Page     int
	PageSize int
}

func (s *Service) List(ctx context.Context, in ListInput) ([]domain.Notification, int64, error) {
	if err := s.authz.EnsureModule(ctx, "finance.notifications"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.notifications.read"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.notifications.read"); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, domain.ListFilter{
		TenantID: in.TenantID,
		Status:   in.Status,
		Page:     in.Page,
		PageSize: in.PageSize,
	})
}

func (s *Service) MarkRead(ctx context.Context, tenantID, notificationID string) (domain.Notification, error) {
	if err := s.authz.EnsureModule(ctx, "finance.notifications"); err != nil {
		return domain.Notification{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.notifications.write"); err != nil {
		return domain.Notification{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.notifications.update"); err != nil {
		return domain.Notification{}, err
	}
	updated, err := s.repo.MarkRead(ctx, tenantID, notificationID)
	if err != nil {
		return domain.Notification{}, err
	}
	_ = s.audit.Write(ctx, "finance.notification.read", "finance_notification", notificationID, nil, map[string]any{"status": updated.Status})
	return updated, nil
}

