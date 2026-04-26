package group

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/kylejs/splitty/backend/internal/auth"
	"github.com/kylejs/splitty/backend/internal/db"
)

func testGroupStore(t *testing.T) (*PgGroupStore, *auth.PgUserStore) {
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

	return NewPgGroupStore(pool), auth.NewPgUserStore(pool)
}

func createTestUser(t *testing.T, store *PgGroupStore, userStore *auth.PgUserStore) string {
	t.Helper()
	email := fmt.Sprintf("test-%s@example.com", uuid.NewString())
	u, err := userStore.UpsertByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	t.Cleanup(func() {
		if _, err := store.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, u.ID); err != nil {
			t.Errorf("cleanup user %s: %v", u.ID, err)
		}
	})
	return u.ID
}

func TestCreateGroup_OK(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	userID := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Vacation", userID)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	if g.Name != "Vacation" {
		t.Errorf("Name = %q, want %q", g.Name, "Vacation")
	}
	if g.CreatedBy != userID {
		t.Errorf("CreatedBy = %q, want %q", g.CreatedBy, userID)
	}

	// Creator should be a member.
	isMember, err := store.IsMember(ctx, g.ID, userID)
	if err != nil {
		t.Fatalf("IsMember: %v", err)
	}
	if !isMember {
		t.Error("creator should be a member of the group")
	}
}

func TestGetByID_NotFound(t *testing.T) {
	store, _ := testGroupStore(t)
	ctx := context.Background()

	_, err := store.GetByID(ctx, uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListByUser_Empty(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	userID := createTestUser(t, store, userStore)

	groups, err := store.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}

func TestListByUser_Multiple(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	userID := createTestUser(t, store, userStore)

	g1, err := store.CreateGroup(ctx, "Group A", userID)
	if err != nil {
		t.Fatalf("CreateGroup A: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g1.ID)
	})

	g2, err := store.CreateGroup(ctx, "Group B", userID)
	if err != nil {
		t.Fatalf("CreateGroup B: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g2.ID)
	})

	groups, err := store.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestAddMember_OK(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)
	member := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Trip", creator)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	if err := store.AddMember(ctx, g.ID, member); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	isMember, err := store.IsMember(ctx, g.ID, member)
	if err != nil {
		t.Fatalf("IsMember: %v", err)
	}
	if !isMember {
		t.Error("added user should be a member")
	}
}

func TestAddMember_AlreadyMember(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Trip", creator)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	err = store.AddMember(ctx, g.ID, creator)
	if !errors.Is(err, ErrAlreadyMember) {
		t.Fatalf("expected ErrAlreadyMember, got %v", err)
	}
}

func TestIsMember_False(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)
	outsider := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Trip", creator)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	isMember, err := store.IsMember(ctx, g.ID, outsider)
	if err != nil {
		t.Fatalf("IsMember: %v", err)
	}
	if isMember {
		t.Error("outsider should not be a member")
	}
}

func TestGetMembers(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)
	member := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Trip", creator)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	if err := store.AddMember(ctx, g.ID, member); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	members, err := store.GetMembers(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetMembers: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

func TestCreateTransaction_OK(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)
	member := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Trip", creator)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	if err := store.AddMember(ctx, g.ID, member); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	txn, err := store.CreateTransaction(ctx, g.ID, "Dinner", 5000, creator, []string{creator, member})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}

	if txn.Description != "Dinner" {
		t.Errorf("Description = %q, want %q", txn.Description, "Dinner")
	}
	if txn.Amount != 5000 {
		t.Errorf("Amount = %d, want %d", txn.Amount, 5000)
	}
	if txn.PaidBy != creator {
		t.Errorf("PaidBy = %q, want %q", txn.PaidBy, creator)
	}

	splits, err := store.GetSplitUserIDs(ctx, txn.ID)
	if err != nil {
		t.Fatalf("GetSplitUserIDs: %v", err)
	}
	if len(splits) != 2 {
		t.Errorf("expected 2 splits, got %d", len(splits))
	}
}

func TestCreateTransaction_PayerIsMember(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)
	member := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Trip", creator)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	if err := store.AddMember(ctx, g.ID, member); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	txn, err := store.CreateTransaction(ctx, g.ID, "Cab", 1500, member, []string{creator, member})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if txn.PaidBy != member {
		t.Errorf("PaidBy = %q, want %q", txn.PaidBy, member)
	}
}

func TestAreMembers(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)
	member := createTestUser(t, store, userStore)
	outsider := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Trip", creator)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	if err := store.AddMember(ctx, g.ID, member); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	nonMembers, err := store.AreMembers(ctx, g.ID, []string{creator, member})
	if err != nil {
		t.Fatalf("AreMembers all-members: %v", err)
	}
	if len(nonMembers) != 0 {
		t.Errorf("expected no non-members, got %v", nonMembers)
	}

	nonMembers, err = store.AreMembers(ctx, g.ID, []string{creator, outsider})
	if err != nil {
		t.Fatalf("AreMembers with outsider: %v", err)
	}
	if len(nonMembers) != 1 || nonMembers[0] != outsider {
		t.Errorf("expected [%s] non-members, got %v", outsider, nonMembers)
	}
}

func TestDeleteTransaction_OK(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Trip", creator)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	txn, err := store.CreateTransaction(ctx, g.ID, "Lunch", 2500, creator, []string{creator})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}

	if err := store.DeleteTransaction(ctx, txn.ID); err != nil {
		t.Fatalf("DeleteTransaction: %v", err)
	}

	_, err = store.GetTransaction(ctx, txn.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteTransaction_NotFound(t *testing.T) {
	store, _ := testGroupStore(t)
	ctx := context.Background()

	err := store.DeleteTransaction(ctx, uuid.NewString())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByIDs(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)

	g1, err := store.CreateGroup(ctx, "Group A", creator)
	if err != nil {
		t.Fatalf("CreateGroup A: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g1.ID)
	})

	g2, err := store.CreateGroup(ctx, "Group B", creator)
	if err != nil {
		t.Fatalf("CreateGroup B: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g2.ID)
	})

	// Fetch both groups in reverse order to verify ordering.
	groups, err := store.GetByIDs(ctx, []string{g2.ID, g1.ID})
	if err != nil {
		t.Fatalf("GetByIDs: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].ID != g2.ID {
		t.Errorf("first group ID = %q, want %q", groups[0].ID, g2.ID)
	}
	if groups[1].ID != g1.ID {
		t.Errorf("second group ID = %q, want %q", groups[1].ID, g1.ID)
	}
}

func TestGetByIDs_Empty(t *testing.T) {
	store, _ := testGroupStore(t)
	ctx := context.Background()

	groups, err := store.GetByIDs(ctx, []string{})
	if err != nil {
		t.Fatalf("GetByIDs empty: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}

func TestGetMembersByGroupIDs(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)
	member := createTestUser(t, store, userStore)

	g1, err := store.CreateGroup(ctx, "Group A", creator)
	if err != nil {
		t.Fatalf("CreateGroup A: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g1.ID)
	})

	g2, err := store.CreateGroup(ctx, "Group B", creator)
	if err != nil {
		t.Fatalf("CreateGroup B: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g2.ID)
	})

	// Add member to group A only.
	if err := store.AddMember(ctx, g1.ID, member); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	result, err := store.GetMembersByGroupIDs(ctx, []string{g1.ID, g2.ID})
	if err != nil {
		t.Fatalf("GetMembersByGroupIDs: %v", err)
	}

	// Group A: creator + member.
	if len(result[g1.ID]) != 2 {
		t.Errorf("group A members = %d, want 2", len(result[g1.ID]))
	}
	// Group B: creator only.
	if len(result[g2.ID]) != 1 {
		t.Errorf("group B members = %d, want 1", len(result[g2.ID]))
	}
}

func TestGetMembersByGroupIDs_Empty(t *testing.T) {
	store, _ := testGroupStore(t)
	ctx := context.Background()

	result, err := store.GetMembersByGroupIDs(ctx, []string{})
	if err != nil {
		t.Fatalf("GetMembersByGroupIDs empty: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestGetSplitUserIDsByTransactionIDs(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)
	member := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Trip", creator)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	if err := store.AddMember(ctx, g.ID, member); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	txn1, err := store.CreateTransaction(ctx, g.ID, "Dinner", 5000, creator, []string{creator, member})
	if err != nil {
		t.Fatalf("CreateTransaction 1: %v", err)
	}

	txn2, err := store.CreateTransaction(ctx, g.ID, "Lunch", 2000, creator, []string{creator})
	if err != nil {
		t.Fatalf("CreateTransaction 2: %v", err)
	}

	result, err := store.GetSplitUserIDsByTransactionIDs(ctx, []string{txn1.ID, txn2.ID})
	if err != nil {
		t.Fatalf("GetSplitUserIDsByTransactionIDs: %v", err)
	}

	if len(result[txn1.ID]) != 2 {
		t.Errorf("txn1 splits = %d, want 2", len(result[txn1.ID]))
	}
	if len(result[txn2.ID]) != 1 {
		t.Errorf("txn2 splits = %d, want 1", len(result[txn2.ID]))
	}
}

func TestGetSplitUserIDsByTransactionIDs_Empty(t *testing.T) {
	store, _ := testGroupStore(t)
	ctx := context.Background()

	result, err := store.GetSplitUserIDsByTransactionIDs(ctx, []string{})
	if err != nil {
		t.Fatalf("GetSplitUserIDsByTransactionIDs empty: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestListByGroup(t *testing.T) {
	store, userStore := testGroupStore(t)
	ctx := context.Background()

	creator := createTestUser(t, store, userStore)

	g, err := store.CreateGroup(ctx, "Trip", creator)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), `DELETE FROM groups WHERE id = $1`, g.ID)
	})

	if _, err := store.CreateTransaction(ctx, g.ID, "Breakfast", 1000, creator, []string{creator}); err != nil {
		t.Fatalf("CreateTransaction 1: %v", err)
	}
	if _, err := store.CreateTransaction(ctx, g.ID, "Lunch", 2000, creator, []string{creator}); err != nil {
		t.Fatalf("CreateTransaction 2: %v", err)
	}

	txns, err := store.ListByGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("ListByGroup: %v", err)
	}
	if len(txns) != 2 {
		t.Errorf("expected 2 transactions, got %d", len(txns))
	}
}
