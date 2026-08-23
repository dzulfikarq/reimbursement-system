package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	jwtpkg "github.com/mumtaz/reimbursement-system/backend/internal/pkg/jwt"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/password"
)

type Service struct {
	cfg    *config.Config
	repo   *Repository
	sess   *SessionStore
}

func NewService(cfg *config.Config, repo *Repository, sess *SessionStore) *Service {
	return &Service{cfg: cfg, repo: repo, sess: sess}
}

// Login verifies credentials and mints the full cookie set (access JWT,
// opaque rotating refresh, CSRF). Generic error — never reveals which field
// was wrong.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*UserResponse, string, string, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	u, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return nil, "", "", apperr.Unauthorized("Invalid email or password")
		}
		return nil, "", "", apperr.Internal(err)
	}
	if !u.IsActive {
		return nil, "", "", apperr.Forbidden("Account is disabled")
	}
	if !password.Compare(u.PasswordHash, req.Password) {
		return nil, "", "", apperr.Unauthorized("Invalid email or password")
	}

	access, err := jwtpkg.Sign(s.cfg.AppSecret, u.ID, u.Role, derefUUID(u.DepartmentID), u.Name, s.cfg.AccessTTL)
	if err != nil {
		return nil, "", "", apperr.Internal(err)
	}
	refresh, err := s.sess.Issue(ctx, u.ID)
	if err != nil {
		return nil, "", "", apperr.Internal(err)
	}

	resp, err := s.toUserResponse(ctx, u)
	if err != nil {
		return nil, "", "", apperr.Internal(err)
	}
	return resp, access, refresh, nil
}

// Refresh rotates the refresh chain. Presenting an already-rotated token
// revokes the whole family — that pattern signals token theft.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*UserResponse, string, string, error) {
	fam, jti, secret, err := parseRefreshToken(refreshToken)
	if err != nil {
		return nil, "", "", apperr.Unauthorized("Session expired")
	}

	newRefresh, userID, err := s.sess.Rotate(ctx, fam, jti, secret)
	if err != nil {
		if errors.Is(err, ErrUnknownToken) {
			_ = s.sess.RevokeFamily(ctx, fam)
			return nil, "", "", apperr.Unauthorized("Session expired")
		}
		return nil, "", "", apperr.Internal(err)
	}

	u, err := s.repo.GetByID(ctx, userID)
	if err != nil || !u.IsActive {
		_ = s.sess.RevokeFamily(ctx, fam)
		return nil, "", "", apperr.Unauthorized("Account unavailable")
	}

	access, err := jwtpkg.Sign(s.cfg.AppSecret, u.ID, u.Role, derefUUID(u.DepartmentID), u.Name, s.cfg.AccessTTL)
	if err != nil {
		return nil, "", "", apperr.Internal(err)
	}

	resp, err := s.toUserResponse(ctx, u)
	if err != nil {
		return nil, "", "", apperr.Internal(err)
	}
	return resp, access, newRefresh, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	parts := strings.Split(refreshToken, ".")
	if len(parts) != 3 {
		return nil // nothing to revoke; logout succeeds idempotently
	}
	return s.sess.RevokeFamily(ctx, parts[0])
}

func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperr.NotFound("User not found")
	}
	resp, err := s.toUserResponse(ctx, u)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return resp, nil
}

func (s *Service) toUserResponse(ctx context.Context, u *User) (*UserResponse, error) {
	resp := &UserResponse{
		ID:           u.ID.String(),
		Name:         u.Name,
		Email:        u.Email,
		Role:         u.Role,
		CreatedAt:    u.CreatedAt,
	}
	if u.DepartmentID != nil {
		id := u.DepartmentID.String()
		name, err := s.repo.DepartmentName(ctx, *u.DepartmentID)
		if err == nil {
			resp.DepartmentID = &id
			resp.DepartmentName = &name
		}
	}
	return resp, nil
}

func parseRefreshToken(tok string) (fam, jti, secret string, err error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", errors.New("malformed refresh token")
	}
	return parts[0], parts[1], parts[2], nil
}

func derefUUID(v *uuid.UUID) uuid.UUID {
	if v == nil {
		return uuid.Nil
	}
	return *v
}
