//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lee-econ/orca-core/internal/db"
)

func main() {
	cfg := db.Config{
		Host:     "localhost",
		Port:     5433,
		User:     "orca",
		Password: os.Getenv("ORCA_DB_PASSWORD"),
		Database: "orca_core",
		SSLMode:  "disable",
		PoolMax:  5,
		PoolMin:  1,
	}
	if cfg.Password == "" {
		cfg.Password = "change_me"
	}

	repo, err := db.NewRepository(cfg)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer repo.Close()

	seeder := db.NewSeeder(repo)
	ctx := context.Background()

	if err := seeder.Run(ctx, true); err != nil {
		log.Fatalf("seed: %v", err)
	}

	report, err := seeder.VerifyIntegrity(ctx)
	if err != nil {
		log.Fatalf("verify: %v", err)
	}

	fmt.Printf("Seed complete. Passed: %v\n", report.Passed)
	for _, c := range report.Checks {
		fmt.Printf("  %s: %s (%d rows)\n", c.Table, c.Status, c.Count)
	}
}
