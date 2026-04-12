package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kylejs/splitty/backend/internal/config"
	"github.com/kylejs/splitty/backend/internal/db"
)

func main() {
	cfg := config.Load()
	fmt.Printf("splitty server starting in %s mode\n", cfg.Env)

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	log.Println("database connected and migrations applied")
}
