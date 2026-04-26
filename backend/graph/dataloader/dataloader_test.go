package dataloader

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kylejs/splitty/backend/internal/auth"
	"github.com/kylejs/splitty/backend/internal/db"
	"github.com/kylejs/splitty/backend/internal/group"
)

type testEnv struct {
	loaders    *Loaders
	userStore  *auth.PgUserStore
	groupStore *group.PgGroupStore
	pool       *pgxpool.Pool
}

func setup(t *testing.T) *testEnv {
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

	userStore := auth.NewPgUserStore(pool)
	groupStore := group.NewPgGroupStore(pool)
	return &testEnv{
		loaders:    NewLoaders(userStore, groupStore),
		userStore:  userStore,
		groupStore: groupStore,
		pool:       pool,
	}
}

func (e *testEnv) createUser(t *testing.T) *auth.UserRecord {
	t.Helper()
	email := fmt.Sprintf("dl-test-%s@example.com", uuid.NewString())
	u, err := e.userStore.UpsertByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	t.Cleanup(func() {
		e.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, u.ID)
	})
	return u
}

func (e *testEnv) createGroup(t *testing.T, name string, creatorID string) *group.GroupRecord {
	t.Helper()
	g, err := e.groupStore.CreateGroup(context.Background(), name, creatorID)
	if err != nil {
		t.Fatalf("create test group: %v", err)
	}
	t.Cleanup(func() {
		e.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})
	return g
}

func TestUserLoader_Load(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	user := env.createUser(t)

	got, err := env.loaders.UserLoader.Load(ctx, user.ID)
	if err != nil {
		t.Fatalf("UserLoader.Load: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("ID = %q, want %q", got.ID, user.ID)
	}
	if got.Email != user.Email {
		t.Errorf("Email = %q, want %q", got.Email, user.Email)
	}
}

func TestUserLoader_Load_NotFound(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.loaders.UserLoader.Load(ctx, uuid.NewString())
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
}

func TestUserLoader_LoadAll(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	u1 := env.createUser(t)
	u2 := env.createUser(t)

	results, err := env.loaders.UserLoader.LoadAll(ctx, []string{u1.ID, u2.ID})
	if err != nil {
		t.Fatalf("UserLoader.LoadAll: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != u1.ID {
		t.Errorf("first result ID = %q, want %q", results[0].ID, u1.ID)
	}
	if results[1].ID != u2.ID {
		t.Errorf("second result ID = %q, want %q", results[1].ID, u2.ID)
	}
}

func TestGroupLoader_Load(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	user := env.createUser(t)
	g := env.createGroup(t, "Test Group", user.ID)

	got, err := env.loaders.GroupLoader.Load(ctx, g.ID)
	if err != nil {
		t.Fatalf("GroupLoader.Load: %v", err)
	}
	if got.ID != g.ID {
		t.Errorf("ID = %q, want %q", got.ID, g.ID)
	}
	if got.Name != "Test Group" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Group")
	}
}

func TestGroupLoader_Load_NotFound(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.loaders.GroupLoader.Load(ctx, uuid.NewString())
	if err == nil {
		t.Fatal("expected error for non-existent group, got nil")
	}
}

func TestGroupMembersLoader_Load(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	creator := env.createUser(t)
	member := env.createUser(t)

	g := env.createGroup(t, "Test Group", creator.ID)

	if err := env.groupStore.AddMember(ctx, g.ID, member.ID); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	ids, err := env.loaders.GroupMembersLoader.Load(ctx, g.ID)
	if err != nil {
		t.Fatalf("GroupMembersLoader.Load: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 member IDs, got %d", len(ids))
	}
}

func TestTransactionSplitsLoader_Load(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	creator := env.createUser(t)
	member := env.createUser(t)

	g := env.createGroup(t, "Test Group", creator.ID)

	if err := env.groupStore.AddMember(ctx, g.ID, member.ID); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	txn, err := env.groupStore.CreateTransaction(ctx, g.ID, "Dinner", 5000, creator.ID, []string{creator.ID, member.ID})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}

	ids, err := env.loaders.TransactionSplitsLoader.Load(ctx, txn.ID)
	if err != nil {
		t.Fatalf("TransactionSplitsLoader.Load: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 split user IDs, got %d", len(ids))
	}
}
