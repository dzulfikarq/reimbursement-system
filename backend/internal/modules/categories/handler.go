package categories

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

// RegisterRoutes: list/get any authed (active-only for non-admin), mutations
// admin only.
func RegisterRoutes(v1 *gin.RouterGroup, h *Handler, authn gin.HandlerFunc, adminOnly gin.HandlerFunc) {
	g := v1.Group("/categories")
	g.GET("", authn, h.List)
	g.GET("/:id", authn, h.Get)
	g.POST("", authn, adminOnly, h.Create)
	g.PATCH("/:id", authn, adminOnly, h.Update)
	g.DELETE("/:id", authn, adminOnly, h.Delete)
}

// List godoc
// @Summary List categories
// @Description Admin sees inactive categories too; other roles active-only.
// @Tags categories
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit (max 100)"
// @Param search query string false "Search name/code"
// @Param sort query string false "code | name | is_active | created_at"
// @Param order query string false "ASC | DESC"
// @Success 200 {object} response.Envelope
// @Router /categories [get]
func (h *Handler) List(c *gin.Context) {
	isAdmin := c.GetString("auth_role") == "admin"
	p := listq.Parse(c, "created_at", sortWhitelist)
	res, err := h.svc.List(c.Request.Context(), p, isAdmin)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "OK")
}

// Get godoc
// @Summary Get category by ID
// @Tags categories
// @Success 200 {object} response.Envelope
// @Failure 404 {object} response.Envelope
// @Router /categories/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Err(c, apperr.NotFound("Category not found"))
		return
	}
	res, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "OK")
}

// Create godoc
// @Summary Create category (admin)
// @Tags categories
// @Accept json
// @Success 201 {object} response.Envelope
// @Router /categories [post]
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
	response.Created(c, res, "Category created")
}

// Update godoc
// @Summary Update category (admin); code immutable
// @Tags categories
// @Accept json
// @Success 200 {object} response.Envelope
// @Router /categories/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Err(c, apperr.NotFound("Category not found"))
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
	response.OK(c, res, "Category updated")
}

// Delete godoc
// @Summary Delete category (admin) — blocked while referenced by claims
// @Tags categories
// @Success 204 {object} response.Envelope
// @Failure 409 {object} response.Envelope
// @Router /categories/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Err(c, apperr.NotFound("Category not found"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Err(c, err)
		return
	}
	response.NoContent(c)
}
