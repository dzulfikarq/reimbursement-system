package reimbursements

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
)

// AttachmentStore wraps the MinIO clients: internal client uploads; presigner
// signs against the public endpoint browsers can reach.
type AttachmentStore struct {
	mc       *minio.Client
	presign  *minio.Client
	bucket   string
}

func NewAttachmentStore(mc *minio.Client, presign *minio.Client, bucket string) *AttachmentStore {
	return &AttachmentStore{mc: mc, presign: presign, bucket: bucket}
}

func (s *Service) AddAttachment(ctx context.Context, store *AttachmentStore, claimID uuid.UUID, role string, userID, deptID uuid.UUID, filename string, content []byte, mime string) (*AttachmentResponse, error) {
	current, err := s.repo.GetDetail(ctx, claimID, role, userID, deptID)
	if err != nil {
		return nil, err
	}
	if current.EmployeeID != userID {
		return nil, apperr.Forbidden("Only the owner can attach receipts")
	}
	switch current.Status {
	case "DRAFT", "REJECTED", "SUBMITTED":
	default:
		return nil, apperr.Conflict("Claim is already decided; no further uploads")
	}

	storageKey := fmt.Sprintf("receipts/%s/%s", claimID, uuid.New())
	_, err = store.mc.PutObject(ctx, store.bucket, storageKey,
		bytes.NewReader(content), int64(len(content)),
		minio.PutObjectOptions{ContentType: mime})
	if err != nil {
		return nil, apperr.Internal(err)
	}

	row := &AttachmentRow{
		ID:               uuid.New(),
		ReimbursementID:  claimID,
		UploadedBy:       userID,
		StorageKey:       storageKey,
		OriginalFilename: sanitizeFilename(filename),
		MimeType:         mime,
		SizeBytes:        int64(len(content)),
	}
	if err := s.repo.CreateAttachment(ctx, row); err != nil {
		return nil, apperr.Internal(err)
	}

	size := row.SizeBytes
	return &AttachmentResponse{
		ID:               row.ID.String(),
		OriginalFilename: row.OriginalFilename,
		MimeType:         row.MimeType,
		SizeBytes:        size,
		CreatedAt:        row.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// DownloadURL issues a 60-second presigned GET after re-checking scope.
func (s *Service) DownloadURL(ctx context.Context, store *AttachmentStore, attID uuid.UUID, role string, userID, deptID uuid.UUID) (string, error) {
	att, employeeID, employeeDeptID, _, err := s.repo.GetAttachment(ctx, attID)
	if err != nil {
		return "", err
	}
	if !canSee(role, userID, deptID, employeeID, employeeDeptID) {
		return "", apperr.NotFound("Attachment not found")
	}

	u, err := store.presign.PresignedGetObject(ctx, store.bucket, att.StorageKey,
		60*time.Second, url.Values{})
	if err != nil {
		return "", apperr.Internal(err)
	}
	return u.String(), nil
}

// Scope mirrors listing visibility (docs/02).
func canSee(role string, userID, deptID, employeeID, employeeDeptID uuid.UUID) bool {
	switch role {
	case "finance", "admin":
		return true
	case "manager":
		return userID == employeeID || (deptID != uuid.Nil && deptID == employeeDeptID)
	default:
		return userID == employeeID
	}
}

// sanitizeFilename keeps the basename only; no paths, no separators.
func sanitizeFilename(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 0 {
			return -1
		}
		return r
	}, name)
	if name == "" {
		name = "receipt"
	}
	if len(name) > 255 {
		name = name[len(name)-255:]
	}
	return name
}
