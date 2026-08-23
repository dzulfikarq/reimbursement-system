package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Connect opens the GORM pool with conservative production-safe settings.
func Connect(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	logger.Info("postgres_connected", "host", cfg.DBHost, "db", cfg.DBName)
	return db, nil
}

// Ping verifies DB reachability with a bounded timeout (health checks).
func Ping(ctx context.Context, db *gorm.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// MigrateUp applies all pending embedded migrations. Used on API boot
// (MIGRATE_ON_START) and by cmd/migrate.
func MigrateUp(dsn string) error {
	m, err := newMigrator(dsn)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// MigrateDown rolls back N steps (0 = all).
func MigrateDown(dsn string, steps int) error {
	m, err := newMigrator(dsn)
	if err != nil {
		return err
	}
	if steps <= 0 {
		return m.Down()
	}
	return m.Steps(-steps)
}

func newMigrator(dsn string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Version exposes current schema version for diagnostics.
func Version(dsn string) (uint, bool, error) {
	m, err := newMigrator(dsn)
	if err != nil {
		return 0, false, err
	}
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	var zero uint
	if v == zero && !dirty {
		return v, dirty, sql.ErrNoRows
	}
	return v, dirty, err
}
