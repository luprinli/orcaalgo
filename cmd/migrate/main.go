// Command migrate applies pending database migrations using the Go-managed
// migration runner (the single source of truth, shared with the server's
// startup path). Usage: `go run ./cmd/migrate` or `scripts/migrate.ps1`.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lee-econ/orca-core/internal/db"
)

func main() {
	cfg := db.DefaultConfig()
	repo, err := db.NewRepository(cfg)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer repo.Close()

	if err := repo.RunMigrations(context.Background()); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	pending, err := repo.ListPendingMigrations(context.Background(), migrationsDir())
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	fmt.Printf("Migrations complete. %d pending (should be 0).\n", len(pending))
	for _, p := range pending {
		fmt.Printf("  pending: %s\n", p)
	}
}

func migrationsDir() string {
	if d := os.Getenv("ORCA_MIGRATIONS_DIR"); d != "" {
		return d
	}
	return "internal/db/migrations"
}
