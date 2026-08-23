package reimbursements

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

type Reimbursement struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	EmployeeID  uuid.UUID
	CategoryID  uuid.UUID
	Title       string
	Description *string
	ExpenseDate time.Time
	Amount      string
	Status      string
	CurrentStep int
	SubmittedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`
}

func (Reimbursement) TableName() string { return "reimbursements" }

// detailRow carries joined names for list/detail reads.
type detailRow struct {
	Reimbursement `gorm:"embedded"`
	EmployeeName  string
	CategoryName  string
	CategoryCode  string
}

func baseQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&detailRow{}).
		Select("reimbursements.*", "users.name AS employee_name",
			"categories.name AS category_name", "categories.code AS category_code").
		Joins("JOIN users ON users.id = reimbursements.employee_id").
		Joins("JOIN categories ON categories.id = reimbursements.category_id").
		Where("reimbursements.deleted_at IS NULL")
}

type ItemRow struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	ReimbursementID uuid.UUID
	Description     string
	Quantity        int
	UnitPrice       string
	LineTotal       string
	CreatedAt       time.Time
}

func (ItemRow) TableName() string { return "reimbursement_items" }

type AttachmentRow struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	ReimbursementID  uuid.UUID
	UploadedBy       uuid.UUID
	StorageKey       string
	OriginalFilename string
	MimeType         string
	SizeBytes        int64
	CreatedAt        time.Time
}

func (AttachmentRow) TableName() string { return "attachments" }

var sortWhitelist = map[string]string{
	"title":        "title",
	"amount":       "amount",
	"status":       "status",
	"expense_date": "expense_date",
	"created_at":   "created_at",
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// scopeFilter applies the role-based visibility rule (docs/02): employee=own,
// manager=own department, finance/admin=all.
func scopeFilter(db *gorm.DB, role string, userID, deptID uuid.UUID) *gorm.DB {
	switch role {
	case "finance", "admin":
		return db
	case "manager":
		return db.Where("reimbursements.employee_id IN (SELECT id FROM users WHERE department_id = ?)", deptID)
	default:
		return db.Where("reimbursements.employee_id = ?", userID)
	}
}

type ListFilters struct {
	Status     string
	CategoryID string
	DateFrom   string
	DateTo     string
}

func applyListFilters(db *gorm.DB, f ListFilters) *gorm.DB {
	if f.Status != "" {
		db = db.Where("reimbursements.status = ?::reimb_status", f.Status)
	}
	if f.CategoryID != "" {
		db = db.Where("reimbursements.category_id = ?", f.CategoryID)
	}
	if f.DateFrom != "" {
		db = db.Where("reimbursements.expense_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		db = db.Where("reimbursements.expense_date <= ?", f.DateTo)
	}
	return db
}

func (r *Repository) List(ctx context.Context, p listq.Params, f ListFilters, role string, userID, deptID uuid.UUID) ([]detailRow, int64, error) {
	q := scopeFilter(baseQuery(r.db.WithContext(ctx)), role, userID, deptID)
	q = applyListFilters(q, f)
	if p.Search != "" {
		q = q.Where("(reimbursements.title ILIKE ? OR reimbursements.description ILIKE ?)", "%"+p.Search+"%", "%"+p.Search+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]detailRow, 0, p.Limit)
	err := q.
		Order("reimbursements." + p.Sort + " " + p.Order + ", reimbursements.id " + p.Order).
		Limit(p.Limit).Offset(p.Offset).
		Scan(&rows).Error
	return rows, total, err
}

// GetDetail returns the claim with joined names after enforcing scope.
func (r *Repository) GetDetail(ctx context.Context, id uuid.UUID, role string, userID, deptID uuid.UUID) (*detailRow, error) {
	var row detailRow
	err := scopeFilter(baseQuery(r.db.WithContext(ctx)), role, userID, deptID).
		Where("reimbursements.id = ?", id).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("Claim not found")
	}
	return &row, err
}

func (r *Repository) Items(ctx context.Context, reimbID uuid.UUID) ([]ItemRow, error) {
	var rows []ItemRow
	err := r.db.WithContext(ctx).
		Where("reimbursement_id = ?", reimbID).
		Order("created_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *Repository) Attachments(ctx context.Context, reimbID uuid.UUID) ([]AttachmentRow, error) {
	var rows []AttachmentRow
	err := r.db.WithContext(ctx).
		Where("reimbursement_id = ?", reimbID).
		Order("created_at ASC").
		Find(&rows).Error
	return rows, err
}

// --- writes ---

// EnsureActiveCategory rejects unknown/inactive categories up front.
func (r *Repository) EnsureActiveCategory(ctx context.Context, id uuid.UUID) error {
	var n int64
	err := r.db.WithContext(ctx).Table("categories").
		Where("id = ? AND is_active = TRUE", id).Count(&n).Error
	if err != nil {
		return apperr.Internal(err)
	}
	if n == 0 {
		return apperr.Validation("category_id does not reference an active category")
	}
	return nil
}

func itemArgs(items []ItemRequest) [][]any {
	args := make([][]any, 0, len(items))
	for _, it := range items {
		args = append(args, []any{it.Description, it.Quantity, it.UnitPrice.String()})
	}
	return args
}

// CreateClaim inserts header + items and recomputes the header amount inside
// one transaction. Amount derives from the DB-generated line_total column.
func (r *Repository) CreateClaim(ctx context.Context, id, userID, categoryID uuid.UUID, title, description string, expenseDate time.Time, items []ItemRequest) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO reimbursements (id, employee_id, category_id, title, description, expense_date)
			VALUES (?, ?, ?, ?, ?, ?)`,
			id, userID, categoryID, title, normDescPtr(description), expenseDate).Error; err != nil {
			return err
		}
		if err := insertItems(tx, id, items); err != nil {
			return err
		}
		return recalcAmount(tx, id)
	})
}

func (r *Repository) UpdateClaim(ctx context.Context, id, categoryID uuid.UUID, title, description string, expenseDate time.Time, items []ItemRequest) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE reimbursements
			SET category_id = ?, title = ?, description = ?, expense_date = ?, updated_at = now()
			WHERE id = ?`,
			categoryID, title, normDescPtr(description), expenseDate, id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM reimbursement_items WHERE reimbursement_id = ?", id).Error; err != nil {
			return err
		}
		if err := insertItems(tx, id, items); err != nil {
			return err
		}
		return recalcAmount(tx, id)
	})
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Exec("UPDATE reimbursements SET deleted_at = now() WHERE id = ?", id).Error
}

func insertItems(tx *gorm.DB, reimbID uuid.UUID, items []ItemRequest) error {
	for _, it := range items {
		if err := tx.Exec(`
			INSERT INTO reimbursement_items (reimbursement_id, description, quantity, unit_price)
			VALUES (?, ?, ?, ?)`,
			reimbID, it.Description, it.Quantity, it.UnitPrice.String()).Error; err != nil {
			return err
		}
	}
	return nil
}

// recalcAmount: single UPDATE from SUM(line_total) — arithmetic stays in PG,
// exact numeric, no float ever involved.
func recalcAmount(tx *gorm.DB, reimbID uuid.UUID) error {
	return tx.Exec(`
		UPDATE reimbursements
		SET amount = COALESCE((SELECT SUM(line_total) FROM reimbursement_items WHERE reimbursement_id = ?), 0),
		    updated_at = now()
		WHERE id = ?`, reimbID, reimbID).Error
}

func normDescPtr(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

// --- attachments ---

func (r *Repository) CreateAttachment(ctx context.Context, a *AttachmentRow) error {
	return r.db.WithContext(ctx).Create(a).Error
}

// GetAttachment returns the row plus enough claim context for scoping.
func (r *Repository) GetAttachment(ctx context.Context, attID uuid.UUID) (*AttachmentRow, uuid.UUID, uuid.UUID, string, error) {
	var row struct {
		AttachmentRow `gorm:"embedded"`
		EmployeeID    uuid.UUID
		DepartmentID  *uuid.UUID
		Status        string
	}
	err := r.db.WithContext(ctx).
		Table("attachments").
		Select("attachments.*, r.employee_id, u.department_id, r.status").
		Joins("JOIN reimbursements r ON r.id = attachments.reimbursement_id").
		Joins("JOIN users u ON u.id = r.employee_id").
		Where("attachments.id = ?", attID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, uuid.Nil, uuid.Nil, "", apperr.NotFound("Attachment not found")
	}
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, "", err
	}
	return &row.AttachmentRow, row.EmployeeID, deref(row.DepartmentID), row.Status, nil
}

func deref(v *uuid.UUID) uuid.UUID {
	if v == nil {
		return uuid.Nil
	}
	return *v
}
