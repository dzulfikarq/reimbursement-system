package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
)

// cacheTTL keeps dashboards cheap without lying about freshness.
const cacheTTL = 60 * time.Second

type Service struct {
	repo  *Repository
	cache *redis.Client
}

func NewService(repo *Repository, cache *redis.Client) *Service {
	return &Service{repo: repo, cache: cache}
}

func (s *Service) cached(ctx context.Context, key string, out any, load func() error) error {
	if s.cache != nil {
		if raw, err := s.cache.Get(ctx, key).Result(); err == nil {
			return json.Unmarshal([]byte(raw), out)
		}
	}
	if err := load(); err != nil {
		return err
	}
	if s.cache != nil {
		if raw, err := json.Marshal(out); err == nil {
			s.cache.Set(ctx, key, raw, cacheTTL)
		}
	}
	return nil
}

func (s *Service) Summary(ctx context.Context, role string, userID uuid.UUID) (*SummaryResponse, error) {
	var res SummaryResponse
	err := s.cached(ctx, fmt.Sprintf("dash:sum:%s:%s", role, userID), &res, func() error {
		pending, err := s.repo.PendingCount(ctx, role, userID)
		if err != nil {
			return apperr.Internal(err)
		}
		total, err := s.repo.MonthlyTotal(ctx, role, userID, monthStart(time.Now()))
		if err != nil {
			return apperr.Internal(err)
		}
		res.PendingCount = pending
		res.MonthlyTotal = total

		approved, rejected, err := s.repo.ApprovalCounts(ctx, role, userID)
		if err != nil {
			return apperr.Internal(err)
		}
		if decided := approved + rejected; decided > 0 {
			rate := math.Round(float64(approved)/float64(decided)*1000) / 10
			res.ApprovalRate = &rate
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *Service) MonthlyTrend(ctx context.Context, role string, userID uuid.UUID, months int) (*TrendResponse, error) {
	if months < 1 || months > 24 {
		return nil, apperr.BadRequest("months must be between 1 and 24")
	}
	var res TrendResponse
	err := s.cached(ctx, fmt.Sprintf("dash:trend:%s:%s:%d", role, userID, months), &res, func() error {
		series, err := s.repo.MonthlyTrend(ctx, role, userID, months)
		if err != nil {
			return apperr.Internal(err)
		}
		res.Months = months
		res.Series = series
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *Service) CategoryBreakdown(ctx context.Context, role string, userID uuid.UUID, month string) (*BreakdownResponse, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	ms, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, apperr.Validation("month must be YYYY-MM")
	}
	var res BreakdownResponse
	err = s.cached(ctx, fmt.Sprintf("dash:breakdown:%s:%s:%s", role, userID, month), &res, func() error {
		items, err := s.repo.CategoryBreakdown(ctx, role, userID, ms)
		if err != nil {
			return err
		}
		res.Month = month
		if items == nil {
			items = []BreakdownItem{}
		}
		res.Items = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &res, nil
}


func monthStart(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}
