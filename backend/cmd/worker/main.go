// Worker consumes asynq jobs: email:send → SMTP, report:generate → CSV to
// MinIO. Docs/06: exponential retry (3 attempts), structured failure logs.
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	"github.com/mumtaz/reimbursement-system/backend/internal/modules/tasks"
)

const exportBucket = "exports"

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN()), &gorm.Config{})
	if err != nil {
		logger.Error("db_connect_failed", "error", err)
		os.Exit(1)
	}
	mc, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		logger.Error("minio_client_failed", "error", err)
		os.Exit(1)
	}
	if err := mc.MakeBucket(context.Background(), exportBucket, minio.MakeBucketOptions{}); err != nil {
		exists, e := mc.BucketExists(context.Background(), exportBucket)
		if !exists || e != nil {
			logger.Error("ensure_export_bucket_failed", "error", err)
			os.Exit(1)
		}
	}
	rdb := goredis.NewClient(&goredis.Options{Addr: cfg.RedisAddr})

	h := handlers{cfg: cfg, db: db, mc: mc, rdb: rdb}
	srv := asynq.NewServer(asynq.RedisClientOpt{Addr: cfg.RedisAddr}, asynq.Config{Concurrency: 5})
	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeEmailSend, h.handleEmail)
	mux.HandleFunc(tasks.TypeReportGen, h.handleReport)
	logger.Info("worker_started", "redis", cfg.RedisAddr)
	if err := srv.Run(mux); err != nil {
		logger.Error("worker_fatal", "error", err)
		os.Exit(1)
	}
}

type handlers struct {
	cfg *config.Config
	db  *gorm.DB
	mc  *minio.Client
	rdb *goredis.Client
}

type emailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (h *handlers) handleEmail(ctx context.Context, t *asynq.Task) error {
	var p emailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("bad payload: %w", err)
	}
	msg := strings.Join([]string{
		"From: reimbursement@mumtaz.test",
		"To: " + p.To,
		"Subject: " + p.Subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		p.Body,
	}, "\r\n")
	addr := h.cfg.SMTPAddr
	if addr == "" {
		addr = "mailhog:1025"
	}
	return smtp.SendMail(addr, nil, "reimbursement@mumtaz.test", []string{p.To}, []byte(msg))
}

type reportPayload struct {
	JobID       string `json:"job_id"`
	Month       string `json:"month"`
	Status      string `json:"status"`
	RequestedBy string `json:"requested_by"`
}

func (h *handlers) handleReport(ctx context.Context, t *asynq.Task) error {
	var p reportPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("bad payload: %w", err)
	}
	setJob(ctx, h.rdb, p.JobID, "processing", "")

	rows, err := queryClaims(h.db, p.Month, p.Status)
	if err != nil {
		setJob(ctx, h.rdb, p.JobID, "failed", err.Error())
		return fmt.Errorf("query claims: %w", err)
	}

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "employee", "department", "category", "title", "expense_date", "amount", "status"})
	for _, r := range rows {
		_ = w.Write(r[:])
	}
	w.Flush()

	key := time.Now().Format("2006/01") + "/" + p.JobID + ".csv"
	if _, err = h.mc.PutObject(ctx, exportBucket, key, strings.NewReader(buf.String()), int64(buf.Len()),
		minio.PutObjectOptions{ContentType: "text/csv"}); err != nil {
		setJob(ctx, h.rdb, p.JobID, "failed", err.Error())
		return fmt.Errorf("upload csv: %w", err)
	}
	h.rdb.HSet(ctx, jobKey(p.JobID), "file_key", key)
	setJob(ctx, h.rdb, p.JobID, "done", "")
	return nil
}

func jobKey(id string) string { return "exportjob:" + id }

func setJob(ctx context.Context, rdb *goredis.Client, id, status, errMsg string) {
	fields := map[string]any{"status": status}
	if errMsg != "" {
		fields["error"] = errMsg
	}
	rdb.HSet(ctx, jobKey(id), fields)
	rdb.Expire(ctx, jobKey(id), 24*time.Hour)
}

// queryClaims returns CSV-ready rows for the month (+optional status filter).
func queryClaims(db *gorm.DB, month, status string) ([][8]string, error) {
	type row struct {
		ID         string
		Employee   string
		Department string
		Category   string
		Title      string
		ExpenseDate string
		Amount     string
		Status     string
	}
	q := `SELECT r.id AS id,
	             u.name AS employee,
	             COALESCE(d.name, '') AS department,
	             c.name AS category,
	             r.title AS title,
	             to_char(r.expense_date, 'YYYY-MM-DD') AS expense_date,
	             r.amount::text AS amount,
	             r.status::text AS status
	      FROM reimbursements r
	      JOIN users u ON u.id = r.employee_id
	      LEFT JOIN departments d ON d.id = u.department_id
	      JOIN categories c ON c.id = r.category_id
	      WHERE r.deleted_at IS NULL AND date_trunc('month', r.expense_date) = ?::date`
	args := []any{month + "-01"}
	switch status {
	case "", "all":
	default:
		q += ` AND r.status = ?::reimb_status`
		args = append(args, status)
	}
	q += ` ORDER BY r.created_at`

	var rows []row
	if err := db.Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([][8]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, [8]string{r.ID, r.Employee, r.Department, r.Category, r.Title, r.ExpenseDate, r.Amount, r.Status})
	}
	return out, nil
}
