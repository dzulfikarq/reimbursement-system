package users

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	listq "github.com/mumtaz/reimbursement-system/backend/internal/pkg/listq"
)

// User mirrors the shared users table; auth module owns the session-facing
// subset, this module owns admin CRUD over the same rows.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name         string
	Email        string
	PasswordHash string
	Role         string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (User) TableName() string { return "users" }

var sortWhitelist = map[string]string{
	"name":       "name",
	"email":      "email",
	"role":       "role",
	"is_active":  "is_active",
	"created_at": "created_at",
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// listRow keeps the embedded shape used by Scan.
type listRow struct {
	User `gorm:"embedded"`
}

func baseQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&listRow{}).
		Select("users.id", "users.name", "users.email",
			"users.password_hash", "users.role", "users.is_active", "users.created_at", "users.updated_at")
}

func (r *Repository) List(ctx context.Context, p listq.Params, role string) ([]listRow, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Table("users")
	if p.Search != "" {
		q = q.Where("name ILIKE ? OR email ILIKE ?", "%"+p.Search+"%", "%"+p.Search+"%")
	}
	if role != "" {
		q = q.Where("role = ?", role)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]listRow, 0, p.Limit)
	err := applyFilters(baseQuery(r.db.WithContext(ctx)), p.Search, role).
		Order("users." + p.Sort + " " + p.Order + ", users.id " + p.Order).
		Limit(p.Limit).Offset(p.Offset).
		Scan(&rows).Error
	return rows, total, err
}

// applyFilters mutates and returns db so count + list stay identical.
func applyFilters(db *gorm.DB, search, role string) *gorm.DB {
	if search != "" {
		db = db.Where("users.name ILIKE ? OR users.email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if role != "" {
		db = db.Where("users.role = ?", role)
	}
	return db
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*listRow, error) {
	var row listRow
	err := baseQuery(r.db.WithContext(ctx)).Where("users.id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("User not found")
	}
	return &row, err
}

func (r *Repository) Create(ctx context.Context, u *User) error {
	err := r.db.WithContext(ctx).Create(u).Error
	if err != nil && apperr.IsUniqueViolation(err) {
		return apperr.Conflict("Email already registered")
	}
	return err
}

// Update touches name/role/is_active — never password or email.
func (r *Repository) Update(ctx context.Context, u *User) error {
	err := r.db.WithContext(ctx).
		Exec("UPDATE users SET name = ?, role = ?::user_role, is_active = ?, updated_at = now() WHERE id = ?",
			u.Name, u.Role, u.IsActive, u.ID).Error
	return err
}

func (r *Repository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	res := r.db.WithContext(ctx).
		Exec("UPDATE users SET password_hash = ?, updated_at = now() WHERE id = ?", hash, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.NotFound("User not found")
	}
	return nil
}

func normEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }
