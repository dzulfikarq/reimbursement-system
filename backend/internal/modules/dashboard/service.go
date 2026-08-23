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

func (s *Service) Summary(ctx context.Context, role string, userID, deptID uuid.UUID) (*SummaryResponse, error) {
	var res SummaryResponse
	err := s.cached(ctx, fmt.Sprintf("dash:sum:%s:%s:%s", role, userID, deptID), &res, func() error {
		pending, err := s.repo.PendingCount(ctx, role, userID, deptID)
		if err != nil {
			return apperr.Internal(err)
		}
		total, err := s.repo.MonthlyTotal(ctx, role, userID, deptID, monthStart(time.Now()))
		if err != nil {
			return apperr.Internal(err)
		}
		res.PendingCount = pending
		res.MonthlyTotal = total

		approved, rejected, err := s.repo.ApprovalCounts(ctx, role, userID, deptID)
		if err != nil {
			return apperr.Internal(err)
		}
		if decided := approved + rejected; decided > 0 {
			rate := math.Round(float64(approved)/float64(decided)*1000) / 10
			res.ApprovalRate = &rate
		}

		usage, err := s.repo.BudgetUsage(ctx, deptScope(role, deptID), monthStart(time.Now()))
		if err != nil {
			return apperr.Internal(err)
		}
		res.BudgetUsage = make([]DepartmentUsage, 0, len(usage))
		for _, u := range usage {
			budget := nz(u.Budget)
			spend := nz(u.Spend)
			pct := pctOf(spend, budget)
			res.BudgetUsage = append(res.BudgetUsage, DepartmentUsage{
				DepartmentID:   u.ID.String(),
				DepartmentName: u.Name,
				MonthlyBudget:  budget,
				MonthlySpend:   spend,
				UsedPercent:    pct,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *Service) MonthlyTrend(ctx context.Context, role string, userID, deptID uuid.UUID, months int) (*TrendResponse, error) {
	if months < 1 || months > 24 {
		return nil, apperr.BadRequest("months must be between 1 and 24")
	}
	var res TrendResponse
	err := s.cached(ctx, fmt.Sprintf("dash:trend:%s:%s:%s:%d", role, userID, deptID, months), &res, func() error {
		series, err := s.repo.MonthlyTrend(ctx, role, userID, deptID, months)
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

func (s *Service) CategoryBreakdown(ctx context.Context, role string, userID, deptID uuid.UUID, month string) (*BreakdownResponse, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	ms, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, apperr.Validation("month must be YYYY-MM")
	}
	var res BreakdownResponse
	err = s.cached(ctx, fmt.Sprintf("dash:breakdown:%s:%s:%s:%s", role, userID, deptID, month), &res, func() error {
		items, err := s.repo.CategoryBreakdown(ctx, role, userID, deptID, ms)
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

// deptScope: employees see their own department's budget; managers their
// department; finance/admin see every department.
func deptScope(role string, deptID uuid.UUID) uuid.UUID {
	switch role {
	case "employee", "manager":
		return deptID
	default:
		return uuid.Nil // all departments
	}
}

func monthStart(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

func nz(s *string) string {
	if s == nil || *s == "" {
		return "0.00"
	}
	return *s
}

// pctOf computes spend/budget*100 using integer cents — no float money.
func pctOf(spend, budget string) float64 {
	totalS := cents(spend)
	totalB := cents(budget)
	if totalB <= 0 {
		return 0
	}
	return math.Round(float64(totalS)/float64(totalB)*1000) / 10
}

// cents parses a plain decimal string ("123.45") into integer cents.
func cents(s string) int64 {
	var w int64
	var c int64
	dot := -1
	for i, ch := range s {
		if ch == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		fmt.Sscanf(s, "%d", &w)
		return w * 100
	}
	fmt.Sscanf(s[:dot], "%d", &w)
	frac := s[dot+1:]
	if len(frac) > 2 {
		frac = frac[:2]
	}
	for _, ch := range frac {
		c = c*10 + int64(ch-'0')
	}
	if len(frac) == 1 {
		c *= 10
	}
	return w*100 + c
}
