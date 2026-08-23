package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration, sourced exclusively from environment
// variables (12-factor). Defaults target docker compose service names.
type Config struct {
	Env  string
	Port string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	MinioEndpoint string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket   string
	MinioUseSSL   bool

	MigrateOnStart bool

	FrontendURL string

	AppSecret   string
	AccessTTL   time.Duration
	RefreshTTL  time.Duration
	CookieSecure bool
}

func Load() *Config {
	return &Config{
		Env:  getEnv("APP_ENV", "development"),
		Port: getEnv("PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "postgres"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "reimbursement"),
		DBPassword: getEnv("DB_PASSWORD", "reimbursement"),
		DBName:     getEnv("DB_NAME", "reimbursement"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RedisAddr:     getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "minio:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    getEnv("MINIO_BUCKET", "receipts"),
		MinioUseSSL:    getEnvBool("MINIO_USE_SSL", false),

		MigrateOnStart: getEnvBool("MIGRATE_ON_START", true),

		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),

		AppSecret: getEnv("APP_SECRET", "dev-only-secret-change-me"),
		AccessTTL: 15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
		CookieSecure: os.Getenv("COOKIE_SECURE") == "true",
	}
}

// PostgresDSN returns a lib/pq style DSN for golang-migrate.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
