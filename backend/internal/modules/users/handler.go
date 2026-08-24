package users

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	listq "github.com/mumtaz/reimbursement-system/backend/internal/pkg/listq"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes: everything under /users is admin-only (docs/04).
func RegisterRoutes(v1 *gin.RouterGroup, h *Handler, authn gin.HandlerFunc, adminOnly gin.HandlerFunc) {
	g := v1.Group("/users", authn, adminOnly)
	g.GET("", h.List)
	g.POST("", h.Create)
	g.PATCH("/:id", h.Update)
	g.POST("/:id/reset-password", h.ResetPassword)
}

// List godoc
// @Summary List users (admin)
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit (max 100)"
// @Param search query string false "Search name/email"
// @Param role query string false "employee | manager | finance | admin"
// @Param sort query string false "name | email | role | is_active | created_at"
// @Param order query string false "ASC | DESC"
// @Success 200 {object} response.Envelope
// @Router /users [get]
func (h *Handler) List(c *gin.Context) {
	p := listq.Parse(c, "created_at", sortWhitelist)
	res, err := h.svc.List(c.Request.Context(), p, c.Query("role"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "OK")
}

// Create godoc
// @Summary Create user with initial password (admin)
// @Tags users
// @Accept json
// @Success 201 {object} response.Envelope
// @Router /users [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	res, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, res, "User created")
}

// Update godoc
// @Summary Update user role/name/is_active (admin)
// @Tags users
// @Accept json
// @Success 200 {object} response.Envelope
// @Router /users/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Err(c, apperr.NotFound("User not found"))
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	res, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "User updated")
}

// ResetPassword godoc
// @Summary Reset user password (admin)
// @Tags users
// @Accept json
// @Success 200 {object} response.Envelope
// @Router /users/{id}/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Err(c, apperr.NotFound("User not found"))
		return
	}
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), id, req.NewPassword); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, nil, "Password reset")
}
