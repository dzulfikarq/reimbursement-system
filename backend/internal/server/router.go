// Package server wires every module into a Gin engine. Shared by cmd/api and
// the integration test harness so tests exercise the real route stack.
package server

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	goredis "github.com/redis/go-redis/v9"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/time/rate"
	"gorm.io/gorm"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/mumtaz/reimbursement-system/backend/docs"
	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	healthmod "github.com/mumtaz/reimbursement-system/backend/internal/modules/health"
	authmod "github.com/mumtaz/reimbursement-system/backend/internal/modules/auth"
	catmod "github.com/mumtaz/reimbursement-system/backend/internal/modules/categories"
	dashmod "github.com/mumtaz/reimbursement-system/backend/internal/modules/dashboard"
	reportmod "github.com/mumtaz/reimbursement-system/backend/internal/modules/reports"
	usermod "github.com/mumtaz/reimbursement-system/backend/internal/modules/users"
	reimbmod "github.com/mumtaz/reimbursement-system/backend/internal/modules/reimbursements"
	"github.com/mumtaz/reimbursement-system/backend/internal/middleware"
)

// New builds the full HTTP router. mc may be nil in tests that never touch
// attachments; attachment routes will simply fail if called.
func New(cfg *config.Config, db *gorm.DB, rdb *goredis.Client, mc *minio.Client) *gin.Engine {
	if cfg.Env != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.ErrorHandler(),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.FrontendURL),
		middleware.Recover(slog.Default()),
		middleware.Logger(slog.Default()),
		middleware.CSRF(cfg.AppSecret),
	)

	v1 := r.Group("/api/v1")
	authn := middleware.AuthN(cfg.AppSecret)
	adminOnly := middleware.RequireRole("admin")

	// Rate limits (docs/08 M8): strict on credential endpoints, moderate
	// global budget for everything else. Integration tests share one IP, so
	// they run with effectively unlimited budgets.
	loginGuard := middleware.RateLimit(rate.Every(6*time.Second), 5)
	globalGuard := middleware.RateLimit(rate.Every(time.Second), 40)
	if cfg.Env == "test" {
		loginGuard = middleware.RateLimit(rate.Inf, 1<<20)
		globalGuard = middleware.RateLimit(rate.Inf, 1<<20)
	}

	authRepo := authmod.NewRepository(db)
	authSvc := authmod.NewService(cfg, authRepo, authmod.NewSessionStore(rdb, cfg.RefreshTTL))
	authmod.NewHandler(cfg, authSvc).RegisterRoutes(v1, loginGuard)

	v1.Use(globalGuard)

	catmod.RegisterRoutes(v1, catmod.NewHandler(catmod.NewService(catmod.NewRepository(db))), authn, adminOnly)
	usermod.RegisterRoutes(v1, usermod.NewHandler(usermod.NewService(usermod.NewRepository(db))), authn, adminOnly)

	var store *reimbmod.AttachmentStore
	if mc != nil {
		store = reimbmod.NewAttachmentStore(mc, PresignClient(cfg, slog.Default()), cfg.MinioBucket)
	}
	reimbRepo := reimbmod.NewRepository(db)
	reimbSvc := reimbmod.NewService(reimbRepo)

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})

	reimbmod.RegisterRoutes(v1,
		reimbmod.NewHandler(reimbSvc, reimbmod.NewWorkflowService(cfg, reimbRepo, db, asynqClient), store), authn)

	dashmod.RegisterRoutes(v1, dashmod.NewHandler(dashmod.NewService(dashmod.NewRepository(db), rdb)), authn)

	reportmod.RegisterRoutes(v1,
		reportmod.NewHandler(reportmod.NewService(asynqClient, rdb), PresignClient(cfg, slog.Default()), "exports"),
		authn, middleware.RequireRole("finance", "admin"))

	r.GET("/healthz", healthmod.Handler(healthmod.Deps{
		DB:          db,
		Redis:       rdb,
		Minio:       mc,
		MinioBucket: cfg.MinioBucket,
	}))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

// EnsureBucket creates the attachments bucket when missing.
func EnsureBucket(ctx context.Context, mc *minio.Client, cfg *config.Config, logger *slog.Logger) {
	exists, err := mc.BucketExists(ctx, cfg.MinioBucket)
	if err != nil {
		logger.Warn("minio_probe_failed", "error", err)
		return
	}
	if !exists {
		if err := mc.MakeBucket(ctx, cfg.MinioBucket, minio.MakeBucketOptions{}); err != nil {
			logger.Warn("bucket_create_failed", "bucket", cfg.MinioBucket, "error", err)
			return
		}
		logger.Info("bucket_created", "bucket", cfg.MinioBucket)
	}
}

// PresignClient signs URLs against the public-facing endpoint (defaults to the
// internal one when MINIO_PUBLIC_ENDPOINT is unset). Region pinned so the SDK
// skips its location probe — that would dial the public host from inside the
// container and fail.
func PresignClient(cfg *config.Config, logger *slog.Logger) *minio.Client {
	pc, err := minio.New(cfg.MinioPublicEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
		Region: "us-east-1",
	})
	if err != nil {
		logger.Error("minio_presign_client_failed", "error", err)
		os.Exit(1)
	}
	return pc
}
