package dashboard

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// scope returns a WHERE fragment mirroring claim listing scope:
// employee=own, manager=dept, finance/admin=all (docs/02).
// ponytail: string-built SQL with fixed fragments only — no user input lands here.
func scope(role string, userID, deptID uuid.UUID) (string, []any) {
	switch role {
	case "employee":
		return "r.employee_id = ?", []any{userID}
	case "manager":
		if deptID != uuid.Nil {
			return "r.employee_id IN (SELECT id FROM users WHERE department_id = ?)", []any{deptID}
		}
		return "r.employee_id = ?", []any{userID}
	default: // finance, admin
		return "TRUE", nil
	}
}

const activeStatuses = `('SUBMITTED', 'APPROVED', 'PAID')`

func (r *Repository) PendingCount(ctx context.Context, role string, userID, deptID uuid.UUID) (int64, error) {
	var n int64
	sc, args := scope(role, userID, deptID)
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM reimbursements r
		WHERE r.status = 'SUBMITTED' AND r.deleted_at IS NULL AND `+sc, args...).Scan(&n).Error
	return n, err
}

func (r *Repository) MonthlyTotal(ctx context.Context, role string, userID, deptID uuid.UUID, monthStart time.Time) (string, error) {
	var total *string
	sc, args := scope(role, userID, deptID)
	args = append(args, monthStart)
	err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(r.amount)::text, '0')
		FROM reimbursements r
		WHERE r.status IN `+activeStatuses+` AND r.deleted_at IS NULL AND `+sc+`
		  AND date_trunc('month', r.expense_date) = ?::date`, args...).Scan(&total).Error
	if err != nil || total == nil {
		return "0", err
	}
	return *total, nil
}

// ApprovalCounts feeds the approval rate: approved / (approved + rejected).
func (r *Repository) ApprovalCounts(ctx context.Context, role string, userID, deptID uuid.UUID) (approved, rejected int64, err error) {
	sc, args := scope(role, userID, deptID)
	err = r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(CASE WHEN status = 'APPROVED' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'REJECTED' THEN 1 ELSE 0 END), 0)
		FROM reimbursements r
		WHERE r.deleted_at IS NULL AND `+sc, args...).
		Row().Scan(&approved, &rejected)
	return approved, rejected, err
}

type deptUsageRow struct {
	ID     uuid.UUID
	Name   string
	Budget *string `gorm:"column:budget"`
	Spend  *string `gorm:"column:spend"`
}

func (r *Repository) BudgetUsage(ctx context.Context, deptID uuid.UUID, monthStart time.Time) ([]deptUsageRow, error) {
	var rows []deptUsageRow
	// Placeholder order matches SQL text: month inside the subquery, then the
	// optional department filter.
	args := []any{monthStart}
	var filter string
	if deptID != uuid.Nil {
		filter = "WHERE d.id = ?"
		args = append(args, deptID)
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT d.id, d.name,
		       COALESCE(d.monthly_budget::text, '0') AS budget,
		       COALESCE(spend.total, '0')            AS spend
		FROM departments d
		LEFT JOIN (
			SELECT u.department_id AS dept_id, SUM(r.amount)::text AS total
			FROM reimbursements r JOIN users u ON u.id = r.employee_id
			WHERE r.status IN `+activeStatuses+` AND r.deleted_at IS NULL
			  AND date_trunc('month', r.expense_date) = ?::date
			GROUP BY u.department_id
		) spend ON spend.dept_id = d.id
		`+filter+`
		ORDER BY d.name`, args...).Scan(&rows).Error
	return rows, err
}

func (r *Repository) MonthlyTrend(ctx context.Context, role string, userID, deptID uuid.UUID, months int) ([]TrendPoint, error) {
	sc, scArgs := scope(role, userID, deptID)
	var rows []TrendPoint
	err := r.db.WithContext(ctx).Raw(`
		SELECT to_char(m.month, 'YYYY-MM') AS month,
		       COALESCE(SUM(r.amount)::text, '0') AS total
		FROM generate_series(
			date_trunc('month', now()) - make_interval(months => ? - 1),
			date_trunc('month', now()), interval '1 month') m(month)
		LEFT JOIN (
			SELECT expense_date, amount FROM reimbursements r
			WHERE r.status IN `+activeStatuses+` AND r.deleted_at IS NULL AND `+sc+`
		) r ON date_trunc('month', r.expense_date) = m.month
		GROUP BY m.month ORDER BY m.month`,
		append([]any{months}, scArgs...)...).Scan(&rows).Error
	return rows, err
}

func (r *Repository) CategoryBreakdown(ctx context.Context, role string, userID, deptID uuid.UUID, monthStart time.Time) ([]BreakdownItem, error) {
	sc, scArgs := scope(role, userID, deptID)
	var rows []BreakdownItem
	err := r.db.WithContext(ctx).Raw(`
		SELECT c.id AS category_id, c.name AS category_name,
		       COALESCE(agg.total, '0') AS total,
		       COALESCE(agg.cnt, 0) AS claim_count
		FROM categories c
		LEFT JOIN (
			SELECT r.category_id, SUM(r.amount)::text AS total, COUNT(*) AS cnt
			FROM reimbursements r
			WHERE r.status IN `+activeStatuses+` AND r.deleted_at IS NULL
			  AND date_trunc('month', r.expense_date) = ?::date AND `+sc+`
			GROUP BY r.category_id
		) agg ON agg.category_id = c.id
		ORDER BY COALESCE(agg.total, '0') DESC, c.name`,
		append([]any{monthStart}, scArgs...)...).Scan(&rows).Error
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return rows, nil
}
