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

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	goredis "github.com/redis/go-redis/v9"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	"github.com/mumtaz/reimbursement-system/backend/internal/database"
	"github.com/mumtaz/reimbursement-system/backend/internal/server"
)

// @title Reimbursement Management System API
// @version 1.0
// @description Take-home test, PT Mumtaz Teknologi Indonesia. Cookie-based auth: login via /auth/login, then use X-CSRF-Token header on mutations. Session cookies are HttpOnly — no bearer tokens.
// @BasePath /api/v1
//
// @securityDefinitions.apikey CSRF
// @in header
// @name X-CSRF-Token
func main() {
	cfg := config.Load()

	level := slog.LevelInfo
	if cfg.Env == "development" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

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
	server.EnsureBucket(context.Background(), mc, cfg, logger)
	router := server.New(cfg, db, rdb, mc)

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
