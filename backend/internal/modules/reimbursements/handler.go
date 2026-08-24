package reimbursements

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/mumtaz/reimbursement-system/backend/internal/middleware"
	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	listq "github.com/mumtaz/reimbursement-system/backend/internal/pkg/listq"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/response"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/upload"
)

type Handler struct {
	svc   *Service
	wf    *WorkflowService
	store *AttachmentStore
}

func NewHandler(svc *Service, wf *WorkflowService, store *AttachmentStore) *Handler {
	return &Handler{svc: svc, wf: wf, store: store}
}

// RegisterRoutes wires claim endpoints. All require authentication; scope +
// ownership enforced in the service (docs/02).
func RegisterRoutes(v1 *gin.RouterGroup, h *Handler, authn gin.HandlerFunc) {
	g := v1.Group("/reimbursements", authn)
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/:id", h.GetDetail)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.POST("/:id/attachments", h.UploadAttachment)
	g.POST("/:id/submit", h.Submit)
	g.POST("/:id/approve", h.Approve)
	g.POST("/:id/reject", h.Reject)
	g.POST("/:id/cancel", h.Cancel)
	g.POST("/:id/pay", h.Pay)

	v1.GET("/attachments/:id/download", authn, h.DownloadAttachment)
}

func identity(c *gin.Context) (role string, userID uuid.UUID, ok bool) {
	role = c.GetString(middleware.CtxRole)
	rawUser, _ := c.Get(middleware.CtxUserID)

	u, userOK := rawUser.(uuid.UUID)
	if !userOK {
		c.Error(apperr.Unauthorized("Authentication required"))
		return "", uuid.Nil, false
	}
	return role, u, true
}

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// List godoc
// @Summary List claims (scoped: employee=own, manager/finance/admin=all)
// @Tags reimbursements
// @Param page query int false "Page"
// @Param limit query int false "Limit (max 100)"
// @Param search query string false "Search title/description"
// @Param status query string false "DRAFT | SUBMITTED | APPROVED | REJECTED | PAID | CANCELLED"
// @Param category_id query string false "Category UUID"
// @Param date_from query string false "YYYY-MM-DD"
// @Param date_to query string false "YYYY-MM-DD"
// @Param sort query string false "title | amount | status | expense_date | created_at"
// @Param order query string false "ASC | DESC"
// @Success 200 {object} response.Envelope
// @Router /reimbursements [get]
func (h *Handler) List(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	p := listq.Parse(c, "created_at", sortWhitelist)

	var filters ListFilters
	filters.Status = c.Query("status")
	switch filters.Status {
	case "", "DRAFT", "SUBMITTED", "APPROVED", "REJECTED", "PAID", "CANCELLED":
	default:
		response.Err(c, apperr.Validation("status filter is invalid"))
		return
	}
	filters.CategoryID = c.Query("category_id")
	if filters.CategoryID != "" {
		if _, err := uuid.Parse(filters.CategoryID); err != nil {
			response.Err(c, apperr.Validation("category_id filter is invalid"))
			return
		}
	}
	filters.DateFrom = c.Query("date_from")
	filters.DateTo = c.Query("date_to")

	res, err := h.svc.List(c.Request.Context(), p, filters, role, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "OK")
}

// Create godoc
// @Summary Create DRAFT claim with items; amount computed server-side
// @Tags reimbursements
// @Accept json
// @Success 201 {object} response.Envelope
// @Router /reimbursements [post]
func (h *Handler) Create(c *gin.Context) {
	_, userID, ok := identity(c)
	if !ok {
		return
	}
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	res, err := h.svc.Create(c.Request.Context(), req, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, res, "Claim created")
}

// GetDetail godoc
// @Summary Claim detail — items + attachment metadata (scope-checked)
// @Tags reimbursements
// @Success 200 {object} response.Envelope
// @Failure 404 {object} response.Envelope
// @Router /reimbursements/{id} [get]
func (h *Handler) GetDetail(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	id, valid := parseID(c)
	if !valid {
		response.Err(c, apperr.NotFound("Claim not found"))
		return
	}
	res, err := h.svc.GetDetail(c.Request.Context(), id, role, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "OK")
}

// Update godoc
// @Summary Edit own claim — DRAFT/REJECTED only; items replaced wholesale
// @Tags reimbursements
// @Accept json
// @Success 200 {object} response.Envelope
// @Router /reimbursements/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	id, valid := parseID(c)
	if !valid {
		response.Err(c, apperr.NotFound("Claim not found"))
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	res, err := h.svc.Update(c.Request.Context(), id, req, role, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "Claim updated")
}

// Delete godoc
// @Summary Delete own DRAFT claim → 204
// @Tags reimbursements
// @Success 204 {object} response.Envelope
// @Router /reimbursements/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	id, valid := parseID(c)
	if !valid {
		response.Err(c, apperr.NotFound("Claim not found"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id, role, userID); err != nil {
		response.Err(c, err)
		return
	}
	response.NoContent(c)
}

// UploadAttachment godoc
// @Summary Upload receipt (multipart `file`, ≤5 MB, png/jpeg/webp/pdf by magic bytes)
// @Tags reimbursements
// @Accept multipart/form-data
// @Success 201 {object} response.Envelope
// @Router /reimbursements/{id}/attachments [post]
func (h *Handler) UploadAttachment(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	id, valid := parseID(c)
	if !valid {
		response.Err(c, apperr.NotFound("Claim not found"))
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Err(c, apperr.Validation("file field is required"))
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		response.Err(c, apperr.BadRequest("Could not read uploaded file"))
		return
	}
	defer src.Close()

	content, mime, verr := upload.Validate(src)
	if verr != nil {
		response.Err(c, apperr.Validation(verr.Error()))
		return
	}

	res, err := h.svc.AddAttachment(c.Request.Context(), h.store, id, role, userID, fileHeader.Filename, content, mime)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, res, "Receipt attached")
}

// Submit godoc
// @Summary Submit claim — full policy check, generates approval steps
// @Tags reimbursements
// @Success 200 {object} response.Envelope
// @Failure 409 {object} response.Envelope "DUPLICATE_SUSPECTED / state conflict"
// @Failure 422 {object} response.Envelope "BUSINESS_RULE_VIOLATED with all violations"
// @Router /reimbursements/{id}/submit [post]
func (h *Handler) Submit(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	id, valid := parseID(c)
	if !valid {
		response.Err(c, apperr.NotFound("Claim not found"))
		return
	}
	res, err := h.wf.Submit(c.Request.Context(), id, role, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "Claim submitted")
}

// Approve godoc
// @Summary Approve current pending step (role must match; sequential)
// @Tags reimbursements
// @Success 200 {object} response.Envelope
// @Router /reimbursements/{id}/approve [post]
func (h *Handler) Approve(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	id, valid := parseID(c)
	if !valid {
		response.Err(c, apperr.NotFound("Claim not found"))
		return
	}
	res, err := h.wf.Approve(c.Request.Context(), id, role, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "Claim approved")
}

// Reject godoc
// @Summary Reject — note required
// @Tags reimbursements
// @Accept json
// @Success 200 {object} response.Envelope
// @Router /reimbursements/{id}/reject [post]
func (h *Handler) Reject(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	id, valid := parseID(c)
	if !valid {
		response.Err(c, apperr.NotFound("Claim not found"))
		return
	}
	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err) // binding error surfaces note requirement via validator
		return
	}
	res, err := h.wf.Reject(c.Request.Context(), id, role, userID, req.Note)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "Claim rejected")
}

// Cancel godoc
// @Summary Cancel own SUBMITTED claim before any approval acted → CANCELLED
// @Tags reimbursements
// @Success 200 {object} response.Envelope
// @Router /reimbursements/{id}/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	id, valid := parseID(c)
	if !valid {
		response.Err(c, apperr.NotFound("Claim not found"))
		return
	}
	res, err := h.wf.Cancel(c.Request.Context(), id, role, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "Claim cancelled")
}

// Pay godoc
// @Summary Mark APPROVED claim as PAID (Finance only)
// @Tags reimbursements
// @Success 200 {object} response.Envelope
// @Router /reimbursements/{id}/pay [post]
func (h *Handler) Pay(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	id, valid := parseID(c)
	if !valid {
		response.Err(c, apperr.NotFound("Claim not found"))
		return
	}
	res, err := h.wf.Pay(c.Request.Context(), id, role, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "Claim paid")
}
// @Summary Presigned download URL (302, 60 s TTL; scope-checked)
// @Tags attachments
// @Success 302 {string} 302 "redirect to MinIO"
// @Router /attachments/{id}/download [get]
func (h *Handler) DownloadAttachment(c *gin.Context) {
	role, userID, ok := identity(c)
	if !ok {
		return
	}
	id, valid := parseID(c)
	if !valid {
		response.Err(c, apperr.NotFound("Attachment not found"))
		return
	}
	url, err := h.svc.DownloadURL(c.Request.Context(), h.store, id, role, userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	c.Redirect(302, url)
}
