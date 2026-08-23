package categories

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	listq "github.com/mumtaz/reimbursement-system/backend/internal/pkg/listq"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

var sortWhitelist = map[string]string{
	"code":       "code",
	"name":       "name",
	"is_active":  "is_active",
	"created_at": "created_at",
}

func (r *Repository) List(ctx context.Context, p listq.Params, activeOnly bool) ([]Category, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&Category{})
	if p.Search != "" {
		q = q.Where("name ILIKE ? OR code ILIKE ?", "%"+p.Search+"%", "%"+p.Search+"%")
	}
	if activeOnly {
		q = q.Where("is_active = TRUE")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Category
	err := q.
		Order(p.Sort + " " + p.Order + ", id " + p.Order).
		Limit(p.Limit).Offset(p.Offset).
		Find(&rows).Error
	return rows, total, err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Category, error) {
	var cat Category
	if err := r.db.WithContext(ctx).First(&cat, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("Category not found")
		}
		return nil, err
	}
	return &cat, nil
}

func (r *Repository) Create(ctx context.Context, cat *Category) error {
	err := r.db.WithContext(ctx).Create(cat).Error
	return mapUniq(err)
}

// Code immutable: update touches name/limit/is_active only.
func (r *Repository) Update(ctx context.Context, cat *Category) error {
	err := r.db.WithContext(ctx).
		Exec("UPDATE categories SET name = ?, monthly_limit_per_employee = ?, is_active = ?, updated_at = now() WHERE id = ?",
			cat.Name, cat.MonthlyLimitPerEmployee, cat.IsActive, cat.ID).Error
	return mapUniq(err)
}

// Delete blocked while reimbursements reference the category (docs/04: 409).
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	var refs int64
	err := r.db.WithContext(ctx).Table("reimbursements").
		Where("category_id = ?", id).Count(&refs).Error
	// ponytail: claims table arrives in M3; undefined-table == no references yet.
	if err != nil && !isUndefinedTable(err) {
		return err
	}
	if refs > 0 {
		return apperr.Conflict("Cannot delete: category is used by reimbursement claims")
	}
	res := r.db.WithContext(ctx).Delete(&Category{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.NotFound("Category not found")
	}
	return nil
}

func mapUniq(err error) error {
	if err != nil && apperr.IsUniqueViolation(err) {
		return apperr.Conflict("Category code or name already exists")
	}
	return err
}

// PostgreSQL 42P01 — table not present yet (pre-M3).
func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}
