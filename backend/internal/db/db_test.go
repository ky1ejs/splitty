package db

import (
	"context"
	"io/fs"
	"testing"
)

func TestMigrationsEmbedded(t *testing.T) {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		t.Fatalf("failed to read embedded migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one migration file")
	}
	found := false
	for _, e := range entries {
		if e.Name() == "001_initial.sql" {
			found = true
		}
	}
	if !found {
		t.Error("expected 001_initial.sql in embedded migrations")
	}
}

func TestConnectInvalidURL(t *testing.T) {
	_, err := Connect(context.Background(), "not-a-valid-url")
	if err == nil {
		t.Error("expected error for invalid database URL")
	}
}
