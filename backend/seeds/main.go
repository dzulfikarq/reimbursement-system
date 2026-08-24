// Package main seeds demo users (one per role). Idempotent (upsert on unique
// email). Demo credentials only — never real secrets.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	"github.com/mumtaz/reimbursement-system/backend/internal/database"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/password"
)

var seedUsers = []struct {
	Name, Email, Password, Role string
}{
	{"Ayu Admin", "admin@mumtaz.test", "Admin#12345", "admin"},
	{"Rina Finance", "finance@mumtaz.test", "Finance#12345", "finance"},
	{"Budi Manager", "manager.eng@mumtaz.test", "Manager#12345", "manager"},
	{"Sari Employee", "employee.eng@mumtaz.test", "Employee#12345", "employee"},
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()
	db, err := database.Connect(cfg, logger)
	if err != nil {
		logger.Error("seed_connect_failed", "error", err)
		os.Exit(1)
	}
	defer database.Close(db)

	ctx := context.Background()

	for _, u := range seedUsers {
		hash, err := password.Hash(u.Password)
		if err != nil {
			fatal(logger, err)
		}
		err = db.WithContext(ctx).Exec(`
			INSERT INTO users (name, email, password_hash, role)
			VALUES (?, ?, ?, ?::user_role)
			ON CONFLICT (email) DO UPDATE SET
				password_hash = EXCLUDED.password_hash,
				role = EXCLUDED.role,
				is_active = TRUE`,
			u.Name, u.Email, hash, u.Role).Error
		if err != nil {
			logger.Error("seed_failed", "error", err, "email", u.Email)
			os.Exit(1)
		}
		logger.Info("user_seeded", "email", u.Email, "role", u.Role)
	}

	logger.Info("seed_complete")
}

func fatal(logger *slog.Logger, err error) {
	logger.Error("seed_failed", "error", err)
	os.Exit(1)
}
