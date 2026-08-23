package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/gorm"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	"github.com/mumtaz/reimbursement-system/backend/internal/database"
	healthmod "github.com/mumtaz/reimbursement-system/backend/internal/modules/health"
	authmod "github.com/mumtaz/reimbursement-system/backend/internal/modules/auth"
	catmod "github.com/mumtaz/reimbursement-system/backend/internal/modules/categories"
	deptmod "github.com/mumtaz/reimbursement-system/backend/internal/modules/departments"
	usermod "github.com/mumtaz/reimbursement-system/backend/internal/modules/users"
	"github.com/mumtaz/reimbursement-system/backend/internal/middleware"
)

func main() {
	cfg := config.Load()

	level := slog.LevelInfo
	if cfg.Env == "development" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	if cfg.Env != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(cfg, logger)
	if err != nil {
		logger.Error("postgres_connect_failed", "error", err)
		os.Exit(1)
	}

	if cfg.MigrateOnStart {
		if err := database.MigrateUp(cfg.PostgresDSN()); err != nil {
			logger.Error("migration_failed", "error", err)
			os.Exit(1)
		}
		logger.Info("migrations_applied")
	}

	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer rdb.Close()

	mc, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		logger.Error("minio_client_failed", "error", err)
		os.Exit(1)
	}
	ensureBucket(context.Background(), mc, cfg, logger)
	router := buildRouter(cfg, db, rdb, mc)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		logger.Info("api_started", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server_failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting_down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	_ = database.Close(db)
}

func buildRouter(cfg *config.Config, db *gorm.DB, rdb *goredis.Client, mc *minio.Client) *gin.Engine {
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

	authRepo := authmod.NewRepository(db)
	authSvc := authmod.NewService(cfg, authRepo, authmod.NewSessionStore(rdb, cfg.RefreshTTL))
	authmod.NewHandler(cfg, authSvc).RegisterRoutes(v1)

	deptmod.RegisterRoutes(v1, deptmod.NewHandler(deptmod.NewService(deptmod.NewRepository(db))), authn, adminOnly)
	catmod.RegisterRoutes(v1, catmod.NewHandler(catmod.NewService(catmod.NewRepository(db))), authn, adminOnly)
	usermod.RegisterRoutes(v1, usermod.NewHandler(usermod.NewService(usermod.NewRepository(db))), authn, adminOnly)

	r.GET("/healthz", healthmod.Handler(healthmod.Deps{
		DB:          db,
		Redis:       rdb,
		Minio:       mc,
		MinioBucket: cfg.MinioBucket,
	}))

	return r
}

func ensureBucket(ctx context.Context, mc *minio.Client, cfg *config.Config, logger *slog.Logger) {
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
