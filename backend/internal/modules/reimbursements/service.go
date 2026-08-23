package reimbursements

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	listq "github.com/mumtaz/reimbursement-system/backend/internal/pkg/listq"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

type ListResult struct {
	Items      []ReimbursementResponse `json:"items"`
	Page       int                     `json:"page"`
	Limit      int                     `json:"limit"`
	Total      int64                   `json:"total"`
	TotalPages int                     `json:"total_pages"`
}

func (s *Service) List(ctx context.Context, p listq.Params, f ListFilters, role string, userID, deptID uuid.UUID) (*ListResult, error) {
	rows, total, err := s.repo.List(ctx, p, f, role, userID, deptID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	items := make([]ReimbursementResponse, 0, len(rows))
	for i := range rows {
		items = append(items, toResponse(&rows[i].Reimbursement, rows[i].EmployeeName, rows[i].CategoryName, rows[i].CategoryCode))
	}
	return &ListResult{Items: items, Page: p.Page, Limit: p.Limit, Total: total, TotalPages: p.TotalPages(total)}, nil
}

func (s *Service) GetDetail(ctx context.Context, id uuid.UUID, role string, userID, deptID uuid.UUID) (*DetailResponse, error) {
	row, err := s.repo.GetDetail(ctx, id, role, userID, deptID)
	if err != nil {
		return nil, err
	}

	itemRows, err := s.repo.Items(ctx, id)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	attRows, err := s.repo.Attachments(ctx, id)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	detail := &DetailResponse{
		ReimbursementResponse: toResponse(&row.Reimbursement, row.EmployeeName, row.CategoryName, row.CategoryCode),
		Items:                 make([]ItemResponse, 0, len(itemRows)),
		Attachments:           make([]AttachmentResponse, 0, len(attRows)),
	}
	for _, it := range itemRows {
		up := json.Number(it.UnitPrice)
		lt := json.Number(it.LineTotal)
		detail.Items = append(detail.Items, ItemResponse{
			ID:          it.ID.String(),
			Description: it.Description,
			Quantity:    it.Quantity,
			UnitPrice:   &up,
			LineTotal:   &lt,
		})
	}
	for _, a := range attRows {
		detail.Attachments = append(detail.Attachments, AttachmentResponse{
			ID:               a.ID.String(),
			OriginalFilename: a.OriginalFilename,
			MimeType:         a.MimeType,
			SizeBytes:        a.SizeBytes,
			CreatedAt:        a.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return detail, nil
}

// Create stores a DRAFT with server-computed amount (Σ qty × unit_price done
// by PG over the generated line_total column — client value never trusted).
func (s *Service) Create(ctx context.Context, req CreateRequest, userID uuid.UUID) (*DetailResponse, error) {
	expenseDate, err := validateExpenseDate(req.ExpenseDate)
	if err != nil {
		return nil, err
	}
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, apperr.Validation("category_id is invalid")
	}
	if err := s.repo.EnsureActiveCategory(ctx, categoryID); err != nil {
		return nil, err
	}

	id := uuid.New()
	err = s.repo.CreateClaim(ctx, id, userID, categoryID, req.Title, req.Description, expenseDate, req.Items)
	if err != nil {
		return nil, err
	}
	return s.GetDetail(ctx, id, "admin", userID, uuid.Nil) // owner just created it; admin scope = unrestricted
}

// Update: owner only, DRAFT/REJECTED only (docs/02 rule 5). Items replaced
// wholesale inside one transaction.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest, role string, userID, deptID uuid.UUID) (*DetailResponse, error) {
	current, err := s.repo.GetDetail(ctx, id, role, userID, deptID)
	if err != nil {
		return nil, err
	}
	if current.EmployeeID != userID {
		return nil, apperr.Forbidden("Only the owner can edit this claim")
	}
	if current.Status != "DRAFT" && current.Status != "REJECTED" {
		return nil, apperr.Conflict("Only DRAFT or REJECTED claims can be edited")
	}

	expenseDate, err := validateExpenseDate(req.ExpenseDate)
	if err != nil {
		return nil, err
	}
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, apperr.Validation("category_id is invalid")
	}
	if err := s.repo.EnsureActiveCategory(ctx, categoryID); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateClaim(ctx, id, categoryID, req.Title, req.Description, expenseDate, req.Items); err != nil {
		return nil, err
	}
	return s.GetDetail(ctx, id, "admin", userID, uuid.Nil)
}

// Delete: owner + DRAFT only → soft delete (audit trail survives, docs/03).
func (s *Service) Delete(ctx context.Context, id uuid.UUID, role string, userID, deptID uuid.UUID) error {
	current, err := s.repo.GetDetail(ctx, id, role, userID, deptID)
	if err != nil {
		return err
	}
	if current.EmployeeID != userID {
		return apperr.Forbidden("Only the owner can delete this claim")
	}
	if current.Status != "DRAFT" {
		return apperr.Conflict("Only DRAFT claims can be deleted")
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// expense date format validated by binding; here: not in future beyond 30 days.
func validateExpenseDate(raw string) (time.Time, error) {
	d, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, apperr.Validation("expense_date must be YYYY-MM-DD")
	}
	if d.After(time.Now().AddDate(0, 0, 30)) {
		return time.Time{}, apperr.Validation("expense_date cannot be more than 30 days in the future")
	}
	return d, nil
}

func normDesc(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func toResponse(r *Reimbursement, employeeName, categoryName, categoryCode string) ReimbursementResponse {
	desc := ""
	if r.Description != nil {
		desc = *r.Description
	}
	amount := json.Number(r.Amount)
	return ReimbursementResponse{
		ID:           r.ID.String(),
		EmployeeID:   r.EmployeeID.String(),
		EmployeeName: employeeName,
		CategoryID:   r.CategoryID.String(),
		CategoryName: categoryName,
		CategoryCode: categoryCode,
		Title:        r.Title,
		Description:  desc,
		ExpenseDate:  r.ExpenseDate.Format("2006-01-02"),
		Amount:       &amount,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    r.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
