package categories

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	listq "github.com/mumtaz/reimbursement-system/backend/internal/pkg/listq"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

type ListResult struct {
	Items      []CategoryResponse `json:"items"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	Total      int64              `json:"total"`
	TotalPages int                `json:"total_pages"`
}

// List: admin sees all (incl. inactive), other roles active-only — used for
// claim form dropdowns.
func (s *Service) List(ctx context.Context, p listq.Params, isAdmin bool) (*ListResult, error) {
	rows, total, err := s.repo.List(ctx, p, !isAdmin)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	items := make([]CategoryResponse, 0, len(rows))
	for i := range rows {
		items = append(items, toResponse(&rows[i]))
	}
	return &ListResult{Items: items, Page: p.Page, Limit: p.Limit, Total: total, TotalPages: p.TotalPages(total)}, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*CategoryResponse, error) {
	cat, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toResponse(cat)
	return &resp, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*CategoryResponse, error) {
	isActive := true
	cat := &Category{
		ID:                      uuid.New(),
		Code:                    req.Code,
		Name:                    req.Name,
		MonthlyLimitPerEmployee: numToStrPtr(req.MonthlyLimitPerEmployee),
		IsActive:                isActive,
	}
	if err := s.repo.Create(ctx, cat); err != nil {
		return nil, err
	}
	resp := toResponse(cat)
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*CategoryResponse, error) {
	cat, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	cat.Name = req.Name
	cat.MonthlyLimitPerEmployee = numToStrPtr(req.MonthlyLimitPerEmployee)
	if req.IsActive != nil {
		cat.IsActive = *req.IsActive
	}
	if err := s.repo.Update(ctx, cat); err != nil {
		return nil, err
	}
	resp := toResponse(cat)
	return &resp, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func toResponse(c *Category) CategoryResponse {
	var limit *json.Number
	if c.MonthlyLimitPerEmployee != nil {
		n := json.Number(*c.MonthlyLimitPerEmployee)
		limit = &n
	}
	return CategoryResponse{
		ID:                      c.ID.String(),
		Code:                    c.Code,
		Name:                    c.Name,
		MonthlyLimitPerEmployee: limit,
		IsActive:                c.IsActive,
		CreatedAt:               c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:               c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func numToStrPtr(n *json.Number) *string {
	if n == nil {
		return nil
	}
	s := n.String()
	return &s
}
