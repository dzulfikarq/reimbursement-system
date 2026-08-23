// Package main seeds demo departments + one user per role. Idempotent
// (upsert on unique keys). Demo credentials only — never real secrets.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	"github.com/mumtaz/reimbursement-system/backend/internal/database"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/password"
)

var seedDepartments = []string{"Engineering", "Finance", "Human Resources"}

var seedUsers = []struct {
	Name, Email, Password, Role, Dept string
}{
	{"Ayu Admin", "admin@mumtaz.test", "Admin#12345", "admin", ""},
	{"Rina Finance", "finance@mumtaz.test", "Finance#12345", "finance", "Finance"},
	{"Budi Manager", "manager.eng@mumtaz.test", "Manager#12345", "manager", "Engineering"},
	{"Sari Employee", "employee.eng@mumtaz.test", "Employee#12345", "employee", "Engineering"},
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

	deptIDs := map[string]string{}
	for _, name := range seedDepartments {
		var id string
		err := db.WithContext(ctx).Raw(`
			INSERT INTO departments (name) VALUES (?)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, name).Scan(&id).Error
		if err != nil {
			fatal(logger, fmt.Errorf("seed department %s: %w", name, err))
		}
		deptIDs[name] = id
	}

	for _, u := range seedUsers {
		hash, err := password.Hash(u.Password)
		if err != nil {
			fatal(logger, fmt.Errorf("hash password for %s: %w", u.Email, err))
		}
		var deptArg any
		if u.Dept != "" {
			deptArg = deptIDs[u.Dept]
		}
		err = db.WithContext(ctx).Exec(`
			INSERT INTO users (name, email, password_hash, role, department_id)
			VALUES (?, ?, ?, ?::user_role, ?)
			ON CONFLICT (email) DO UPDATE SET
				password_hash = EXCLUDED.password_hash,
				role = EXCLUDED.role,
				department_id = EXCLUDED.department_id,
				is_active = TRUE`,
			u.Name, u.Email, hash, u.Role, deptArg).Error
		if err != nil {
			fatal(logger, fmt.Errorf("seed user %s: %w", u.Email, err))
		}
		logger.Info("user_seeded", "email", u.Email, "role", u.Role)
	}

	logger.Info("seed_complete")
}

func fatal(logger *slog.Logger, err error) {
	logger.Error("seed_failed", "error", err)
	os.Exit(1)
}
