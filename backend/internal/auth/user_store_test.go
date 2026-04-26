package auth

import (
	"context"
	"fmt"
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

	email := fmt.Sprintf("getbyid-%s@example.com", uuid.NewString())
	created, err := store.UpsertByEmail(ctx, email)
	if err != nil {
		t.Fatalf("UpsertByEmail: %v", err)
	}
	t.Cleanup(func() {
		if _, err := store.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, created.ID); err != nil {
			t.Errorf("cleanup delete user %q: %v", created.ID, err)
		}
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

func TestGetByEmail_OK(t *testing.T) {
	store := testUserStore(t)
	ctx := context.Background()

	email := fmt.Sprintf("getbyemail-%s@example.com", uuid.NewString())
	created, err := store.UpsertByEmail(ctx, email)
	if err != nil {
		t.Fatalf("UpsertByEmail: %v", err)
	}
	t.Cleanup(func() {
		if _, err := store.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, created.ID); err != nil {
			t.Errorf("cleanup delete user %q: %v", created.ID, err)
		}
	})

	got, err := store.GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
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

func TestGetByEmail_NotFound(t *testing.T) {
	store := testUserStore(t)
	ctx := context.Background()

	_, err := store.GetByEmail(ctx, fmt.Sprintf("nonexistent-%s@example.com", uuid.NewString()))
	if err == nil {
		t.Fatal("expected error for non-existent email, got nil")
	}
}
