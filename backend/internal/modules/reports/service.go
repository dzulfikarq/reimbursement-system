package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/mumtaz/reimbursement-system/backend/internal/modules/tasks"
	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
)

type Service struct {
	client *asynq.Client
	rdb    *redis.Client
}

func NewService(client *asynq.Client, rdb *redis.Client) *Service {
	return &Service{client: client, rdb: rdb}
}

type ExportStatusResponse struct {
	JobID   string  `json:"job_id"`
	Status  string  `json:"status"` // queued | processing | done | failed
	Error   string  `json:"error,omitempty"`
	FileURL *string `json:"file_url,omitempty"`
}

// QueueExport validates filters, registers the job in Redis and enqueues it.
func (s *Service) QueueExport(ctx context.Context, month string, status string, requestedBy uuid.UUID) (*ExportStatusResponse, error) {
	if _, err := time.Parse("2006-01", month); err != nil {
		return nil, apperr.Validation("month must be YYYY-MM")
	}
	switch status {
	case "", "all", "DRAFT", "SUBMITTED", "APPROVED", "REJECTED", "PAID", "CANCELLED":
	default:
		return nil, apperr.Validation("unknown status filter")
	}

	jobID := uuid.New()
	payload, _ := json.Marshal(map[string]string{
		"job_id":       jobID.String(),
		"month":        month,
		"status":       status,
		"requested_by": requestedBy.String(),
	})
	if _, err := s.client.EnqueueContext(ctx,
		asynq.NewTask(tasks.TypeReportGen, payload), asynq.MaxRetry(3)); err != nil {
		return nil, apperr.Internal(err)
	}
	s.rdb.HSet(ctx, jobKey(jobID.String()), "status", "queued")
	s.rdb.Expire(ctx, jobKey(jobID.String()), 24*time.Hour)
	return &ExportStatusResponse{JobID: jobID.String(), Status: "queued"}, nil
}

// Status returns job state; presigns the CSV when done.
func (s *Service) Status(ctx context.Context, jobID uuid.UUID, presign func(key string) (string, error)) (*ExportStatusResponse, error) {
	raw, err := s.rdb.HGetAll(ctx, jobKey(jobID.String())).Result()
	if err != nil || len(raw) == 0 {
		return nil, apperr.NotFound("Export job not found")
	}
	res := &ExportStatusResponse{JobID: jobID.String(), Status: raw["status"], Error: raw["error"]}
	if res.Status == "done" && raw["file_key"] != "" {
		url, err := presign(raw["file_key"])
		if err != nil {
			return nil, apperr.Internal(err)
		}
		res.FileURL = &url
	}
	return res, nil
}

func jobKey(id string) string { return fmt.Sprintf("exportjob:%s", id) }
