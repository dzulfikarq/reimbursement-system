package departments

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
	Items      []DepartmentResponse `json:"items"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	Total      int64                `json:"total"`
	TotalPages int                  `json:"total_pages"`
}

func (s *Service) List(ctx context.Context, p listq.Params) (*ListResult, error) {
	rows, total, err := s.repo.List(ctx, p)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	items := make([]DepartmentResponse, 0, len(rows))
	for i := range rows {
		items = append(items, toResponse(&rows[i]))
	}
	return &ListResult{Items: items, Page: p.Page, Limit: p.Limit, Total: total, TotalPages: p.TotalPages(total)}, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*DepartmentResponse, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toResponse(d)
	return &resp, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*DepartmentResponse, error) {
	d := &Department{
		ID:            uuid.New(),
		Name:          req.Name,
		MonthlyBudget: numToStrPtr(req.MonthlyBudget),
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	resp := toResponse(d)
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*DepartmentResponse, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Name = req.Name
	d.MonthlyBudget = numToStrPtr(req.MonthlyBudget)
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	resp := toResponse(d)
	return &resp, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func toResponse(d *Department) DepartmentResponse {
	var budget *json.Number
	if d.MonthlyBudget != nil {
		b := json.Number(*d.MonthlyBudget)
		budget = &b
	}
	return DepartmentResponse{
		ID:            d.ID.String(),
		Name:          d.Name,
		MonthlyBudget: budget,
		CreatedAt:     d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     d.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// json.Number → *string for the numeric(14,2) column; PG casts the literal.
func numToStrPtr(n *json.Number) *string {
	if n == nil {
		return nil
	}
	s := n.String()
	return &s
}
