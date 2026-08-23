package dashboard

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// identity reads auth context keys set by the AuthN middleware.
func identity(c *gin.Context) (role string, userID, deptID uuid.UUID, ok bool) {
	rawID, _ := c.Get("auth_user_id")
	rawRole, _ := c.Get("auth_role")
	rawDept, _ := c.Get("auth_department_id")
	id, valid := rawID.(uuid.UUID)
	if !valid {
		c.Error(apperr.Unauthorized("Missing or invalid credentials"))
		return "", uuid.Nil, uuid.Nil, false
	}
	role, _ = rawRole.(string)
	deptID, _ = rawDept.(uuid.UUID)
	return role, id, deptID, true
}

// Summary godoc
// @Summary Role-scoped dashboard: pending count, monthly total, approval rate, budget usage
// @Tags dashboard
// @Success 200 {object} response.Envelope
// @Router /dashboard/summary [get]
func (h *Handler) Summary(c *gin.Context) {
	role, userID, deptID, ok := identity(c)
	if !ok {
		return
	}
	res, err := h.svc.Summary(c.Request.Context(), role, userID, deptID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "Summary")
}

// MonthlyTrend godoc
// @Summary Claim totals for the last N months (?months=6, max 24)
// @Tags dashboard
// @Param months query int false "1-24"
// @Success 200 {object} response.Envelope
// @Router /dashboard/monthly-trend [get]
func (h *Handler) MonthlyTrend(c *gin.Context) {
	role, userID, deptID, ok := identity(c)
	if !ok {
		return
	}
	months := 6
	if raw := c.Query("months"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			response.Err(c, apperr.Validation("months must be an integer"))
			return
		}
		months = n
	}
	res, err := h.svc.MonthlyTrend(c.Request.Context(), role, userID, deptID, months)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "Monthly trend")
}

// CategoryBreakdown godoc
// @Summary Per-category totals for a month (?month=YYYY-MM, default current)
// @Tags dashboard
// @Param month query string false "YYYY-MM"
// @Success 200 {object} response.Envelope
// @Router /dashboard/category-breakdown [get]
func (h *Handler) CategoryBreakdown(c *gin.Context) {
	role, userID, deptID, ok := identity(c)
	if !ok {
		return
	}
	res, err := h.svc.CategoryBreakdown(c.Request.Context(), role, userID, deptID, c.Query("month"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "Category breakdown")
}

// RegisterRoutes wires dashboard endpoints (read-only; auth required).
func RegisterRoutes(v1 *gin.RouterGroup, h *Handler, authn gin.HandlerFunc) {
	g := v1.Group("/dashboard", authn)
	g.GET("/summary", h.Summary)
	g.GET("/monthly-trend", h.MonthlyTrend)
	g.GET("/category-breakdown", h.CategoryBreakdown)
}
