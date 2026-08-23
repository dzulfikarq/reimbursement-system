package departments

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	listq "github.com/mumtaz/reimbursement-system/backend/internal/pkg/listq"
)

type Department struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name          string
	MonthlyBudget *string `gorm:"column:monthly_budget"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (Department) TableName() string { return "departments" }

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

var sortWhitelist = map[string]string{
	"name":       "name",
	"created_at": "created_at",
}

func (r *Repository) List(ctx context.Context, p listq.Params) ([]Department, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&Department{})
	if p.Search != "" {
		q = q.Where("name ILIKE ?", "%"+p.Search+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Department
	err := q.
		Order(p.Sort + " " + p.Order + ", id " + p.Order).
		Limit(p.Limit).Offset(p.Offset).
		Find(&rows).Error
	return rows, total, err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Department, error) {
	var d Department
	if err := r.db.WithContext(ctx).First(&d, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("Department not found")
		}
		return nil, err
	}
	return &d, nil
}

func (r *Repository) Create(ctx context.Context, d *Department) error {
	err := r.db.WithContext(ctx).Create(d).Error
	return mapUniq(err, d.Name)
}

func (r *Repository) Update(ctx context.Context, d *Department) error {
	err := r.db.WithContext(ctx).
		Exec("UPDATE departments SET name = ?, monthly_budget = ?, updated_at = now() WHERE id = ?",
			d.Name, d.MonthlyBudget, d.ID).Error
	return mapUniq(err, d.Name)
}

// Delete blocked while users still reference the department (docs/04: 409).
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	var refs int64
	if err := r.db.WithContext(ctx).Table("users").
		Where("department_id = ?", id).Count(&refs).Error; err != nil {
		return err
	}
	if refs > 0 {
		return apperr.Conflict("Cannot delete: department still has users")
	}
	res := r.db.WithContext(ctx).Delete(&Department{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.NotFound("Department not found")
	}
	return nil
}

// PG unique violation → 409 with a friendly message.
func mapUniq(err error, name string) error {
	if err != nil && apperr.IsUniqueViolation(err) {
		return apperr.Conflict("Department name '" + name + "' already exists")
	}
	return err
}
