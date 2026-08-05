package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"pekan/backend/internal/modules/finance/dashboard/domain"
)

type Authorizer interface {
	EnsureModule(ctx context.Context, module string) error
	EnsureFeature(ctx context.Context, feature string) error
	EnsurePermission(ctx context.Context, permission string) error
}

type Service struct {
	repo  domain.Repository
	authz Authorizer
	redis *redis.Client
}

func NewService(repo domain.Repository, authz Authorizer, rdb *redis.Client) *Service {
	return &Service{
		repo:  repo,
		authz: authz,
		redis: rdb,
	}
}

type SummaryInput struct {
	TenantID string
	DateFrom *string
	DateTo   *string
}

func (s *Service) GetSummary(ctx context.Context, in SummaryInput) (domain.Summary, error) {
	if err := s.authz.EnsureModule(ctx, "finance.dashboard"); err != nil {
		return domain.Summary{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.dashboard.read"); err != nil {
		return domain.Summary{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.dashboard.read"); err != nil {
		return domain.Summary{}, err
	}
	if s.redis != nil {
		cacheKey := fmt.Sprintf("finance:dashboard:summary:%s:%v:%v", in.TenantID, derefStr(in.DateFrom), derefStr(in.DateTo))
		if val, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
			var summary domain.Summary
			if err := json.Unmarshal([]byte(val), &summary); err == nil {
				return summary, nil
			}
		}
		
		summary, err := s.repo.GetSummary(ctx, in.TenantID, in.DateFrom, in.DateTo)
		if err == nil {
			payload, _ := json.Marshal(summary)
			s.redis.Set(ctx, cacheKey, payload, 5*time.Minute)
		}
		return summary, err
	}

	return s.repo.GetSummary(ctx, in.TenantID, in.DateFrom, in.DateTo)
}

func derefStr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

type SeriesInput struct {
	TenantID string
	DateFrom *string
	DateTo   *string
}

func (s *Service) GetDailySeries(ctx context.Context, in SeriesInput) ([]domain.SeriesPoint, error) {
	if err := s.authz.EnsureModule(ctx, "finance.dashboard"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.dashboard.read"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.dashboard.read"); err != nil {
		return nil, err
	}
	return s.repo.GetDailySeries(ctx, in.TenantID, in.DateFrom, in.DateTo)
}

type TopCategoriesInput struct {
	TenantID string
	DateFrom *string
	DateTo   *string
	Limit    int
}

func (s *Service) GetTopCategories(ctx context.Context, in TopCategoriesInput) ([]domain.CategoryTotal, error) {
	if err := s.authz.EnsureModule(ctx, "finance.dashboard"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.dashboard.read"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.dashboard.read"); err != nil {
		return nil, err
	}
	if in.Limit <= 0 || in.Limit > 50 {
		in.Limit = 10
	}
	return s.repo.GetTopCategories(ctx, in.TenantID, in.DateFrom, in.DateTo, in.Limit)
}

