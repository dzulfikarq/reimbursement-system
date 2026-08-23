package main

import (
	"flag"
	"log"
	"os"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	"github.com/mumtaz/reimbursement-system/backend/internal/database"
)

// cmd/migrate runs schema migrations manually (CI, ops). The API also
// migrates on boot when MIGRATE_ON_START=true.
//
// Usage:
//
//	go run ./cmd/migrate -direction up
//	go run ./cmd/migrate -direction down [-steps 1]
func main() {
	direction := flag.String("direction", "up", "up|down")
	steps := flag.Int("steps", 1, "steps to roll back (down only, 0 = all)")
	flag.Parse()

	cfg := config.Load()
	dsn := cfg.PostgresDSN()

	var err error
	switch *direction {
	case "up":
		err = database.MigrateUp(dsn)
	case "down":
		err = database.MigrateDown(dsn, *steps)
	default:
		log.Fatalf("unknown direction %q (use up|down)", *direction)
	}
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	v, dirty, _ := database.Version(dsn)
	log.Printf("migrations %s ok — version=%d dirty=%v\n", *direction, v, dirty)
	os.Exit(0)
}
