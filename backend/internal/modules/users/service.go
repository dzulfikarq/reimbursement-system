package users

import (
	"context"

	"github.com/google/uuid"

	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/password"
	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	listq "github.com/mumtaz/reimbursement-system/backend/internal/pkg/listq"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

type ListResult struct {
	Items      []UserResponse `json:"items"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	Total      int64          `json:"total"`
	TotalPages int            `json:"total_pages"`
}

// List: admin-only route; role/department filters validated loosely — unknown
// values just return empty results (no injection risk, params bound).
func (s *Service) List(ctx context.Context, p listq.Params, role, departmentID string) (*ListResult, error) {
	rows, total, err := s.repo.List(ctx, p, role, departmentID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	items := make([]UserResponse, 0, len(rows))
	for i := range rows {
		items = append(items, toResponse(&rows[i].User, rows[i].DepartmentName))
	}
	return &ListResult{Items: items, Page: p.Page, Limit: p.Limit, Total: total, TotalPages: p.TotalPages(total)}, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*UserResponse, error) {
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	u := &User{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        normEmail(req.Email),
		PasswordHash: hash,
		Role:         req.Role,
		IsActive:     true,
	}
	if req.DepartmentID != nil {
		depID, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			return nil, apperr.Validation("department_id is invalid")
		}
		u.DepartmentID = &depID
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	resp := toResponse(u, nil)
	return &resp, nil
}

// Update: partial semantics — only provided fields change. Email and password
// are immutable here (password via /reset-password).
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*UserResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	u := &row.User

	if req.Name != nil {
		u.Name = *req.Name
	}
	if req.Role != nil {
		u.Role = *req.Role
	}
	if req.DepartmentID != nil {
		depID, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			return nil, apperr.Validation("department_id is invalid")
		}
		u.DepartmentID = &depID
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	resp := toResponse(u, row.DepartmentName)
	return &resp, nil
}

// ResetPassword: admin sets a new password; no self-service flow in M2
// (forgot-password comes later if needed).
func (s *Service) ResetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	hash, err := password.Hash(newPassword)
	if err != nil {
		return apperr.Internal(err)
	}
	return s.repo.UpdatePassword(ctx, id, hash)
}

func toResponse(u *User, deptName *string) UserResponse {
	var depID *string
	if u.DepartmentID != nil {
		id := u.DepartmentID.String()
		depID = &id
	}
	return UserResponse{
		ID:             u.ID.String(),
		Name:           u.Name,
		Email:          u.Email,
		Role:           u.Role,
		DepartmentID:   depID,
		DepartmentName: deptName,
		IsActive:       u.IsActive,
		CreatedAt:      u.CreatedAt,
	}
}
