package auth

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/kylejs/splitty/backend/internal/db"
)

func testUserStore(t *testing.T) *PgUserStore {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	return NewPgUserStore(pool)
}

func TestGetByID_OK(t *testing.T) {
	store := testUserStore(t)
	ctx := context.Background()

	// Create a user via UpsertByEmail
	created, err := store.UpsertByEmail(ctx, "getbyid@example.com")
	if err != nil {
		t.Fatalf("UpsertByEmail: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, created.ID)
	})

	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Email != created.Email {
		t.Errorf("Email = %q, want %q", got.Email, created.Email)
	}
	if got.DisplayName != created.DisplayName {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, created.DisplayName)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	store := testUserStore(t)
	ctx := context.Background()

	_, err := store.GetByID(ctx, uuid.NewString())
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
}
