package departments

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

// RegisterRoutes: GETs for any authed user (form dropdowns), mutations admin
// only (docs/04).
func RegisterRoutes(v1 *gin.RouterGroup, h *Handler, authn gin.HandlerFunc, adminOnly gin.HandlerFunc) {
	g := v1.Group("/departments")
	g.GET("", authn, h.List)
	g.GET("/:id", authn, h.Get)
	g.POST("", authn, adminOnly, h.Create)
	g.PATCH("/:id", authn, adminOnly, h.Update)
	g.DELETE("/:id", authn, adminOnly, h.Delete)
}

// List godoc
// @Summary List departments
// @Tags departments
// @Produce json
// @Param page query int false "Page (default 1)"
// @Param limit query int false "Limit (default 10, max 100)"
// @Param search query string false "Search by name"
// @Param sort query string false "name | created_at"
// @Param order query string false "ASC | DESC"
// @Success 200 {object} response.Envelope
// @Router /departments [get]
func (h *Handler) List(c *gin.Context) {
	p := listq.Parse(c, "created_at", sortWhitelist)
	res, err := h.svc.List(c.Request.Context(), p)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "OK")
}

// Get godoc
// @Summary Get department by ID
// @Tags departments
// @Success 200 {object} response.Envelope
// @Failure 404 {object} response.Envelope
// @Router /departments/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Err(c, apperr.NotFound("Department not found"))
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
// @Summary Create department (admin)
// @Tags departments
// @Accept json
// @Success 201 {object} response.Envelope
// @Router /departments [post]
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
	response.Created(c, res, "Department created")
}

// Update godoc
// @Summary Update department (admin)
// @Tags departments
// @Accept json
// @Success 200 {object} response.Envelope
// @Router /departments/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Err(c, apperr.NotFound("Department not found"))
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
	response.OK(c, res, "Department updated")
}

// Delete godoc
// @Summary Delete department (admin) — blocked while referenced
// @Tags departments
// @Success 204 {object} response.Envelope
// @Failure 409 {object} response.Envelope
// @Router /departments/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Err(c, apperr.NotFound("Department not found"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Err(c, err)
		return
	}
	response.NoContent(c)
}
