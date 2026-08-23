package reports

import (
	"context"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/response"
)

type Handler struct {
	svc    *Service
	mc     MinioPresigner
	bucket string
}

// MinioPresigner is the slice of the minio client we need — keeps tests light.
type MinioPresigner interface {
	PresignedGetObject(ctx context.Context, bucket, object string, expiry time.Duration, reqParams url.Values) (*url.URL, error)
}

func NewHandler(svc *Service, mc MinioPresigner, bucket string) *Handler {
	return &Handler{svc: svc, mc: mc, bucket: bucket}
}

// presignPublic builds a download URL pointing at the public MinIO endpoint so
// browsers can fetch it (same trick as attachment downloads).
func (h *Handler) presignPublic(key string) (string, error) {
	u, err := h.mc.PresignedGetObject(context.Background(), h.bucket, key, 24*time.Hour, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// Export godoc
// @Summary Queue async CSV export of claims (Finance/Admin only)
// @Tags reports
// @Param month query string false "YYYY-MM"
// @Param status query string false "claim status filter"
// @Success 202 {object} response.Envelope
// @Router /reports/export [get]
func (h *Handler) Export(c *gin.Context) {
	rawUser, _ := c.Get("auth_user_id")
	userID := rawUser.(uuid.UUID)

	month := c.Query("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	res, err := h.svc.QueueExport(c.Request.Context(), month, c.Query("status"), userID)
	if err != nil {
		response.Err(c, err)
		return
	}
	c.Header("Location", "/api/v1/reports/export/"+res.JobID)
	response.Created(c, res, "Export queued")
}

// ExportStatus godoc
// @Summary Poll export job; returns download URL when done
// @Tags reports
// @Param jobId path string true "job id"
// @Success 200 {object} response.Envelope
// @Router /reports/export/{jobId} [get]
func (h *Handler) ExportStatus(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		response.Err(c, apperr.Validation("invalid job id"))
		return
	}
	res, err := h.svc.Status(c.Request.Context(), jobID, h.presignPublic)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, res, "Export status")
}

// RegisterRoutes wires report endpoints (Finance/Admin only).
func RegisterRoutes(v1 *gin.RouterGroup, h *Handler, authn, financeAdmin gin.HandlerFunc) {
	g := v1.Group("/reports", authn)
	g.GET("/export", financeAdmin, h.Export)
	g.GET("/export/:jobId", financeAdmin, h.ExportStatus)
}
