package auth

import (
	"context"
	"testing"
)

func TestUserIDFromContext(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		ctx := withUserID(context.Background(), "user-456")
		id, ok := UserIDFromContext(ctx)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if id != "user-456" {
			t.Errorf("got %q, want %q", id, "user-456")
		}
	})

	t.Run("absent", func(t *testing.T) {
		id, ok := UserIDFromContext(context.Background())
		if ok {
			t.Fatal("expected ok=false")
		}
		if id != "" {
			t.Errorf("got %q, want empty", id)
		}
	})
}
