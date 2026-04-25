package group

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgGroupStore implements group and transaction persistence using a pgx connection pool.
type PgGroupStore struct {
	pool *pgxpool.Pool
}

// NewPgGroupStore creates a PgGroupStore backed by the given pool.
func NewPgGroupStore(pool *pgxpool.Pool) *PgGroupStore {
	return &PgGroupStore{pool: pool}
}

// CreateGroup inserts a new group and adds the creator as the first member.
func (s *PgGroupStore) CreateGroup(ctx context.Context, name, creatorUserID string) (*GroupRecord, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("create group: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var g GroupRecord
	err = tx.QueryRow(ctx,
		`INSERT INTO groups (name, created_by) VALUES ($1, $2)
		 RETURNING id, name, created_by, created_at`,
		name, creatorUserID,
	).Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create group: insert: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)`,
		g.ID, creatorUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("create group: add creator as member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("create group: commit: %w", err)
	}
	return &g, nil
}

// GetByID returns the group with the given ID, or ErrNotFound.
func (s *PgGroupStore) GetByID(ctx context.Context, groupID string) (*GroupRecord, error) {
	var g GroupRecord
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, created_by, created_at FROM groups WHERE id = $1`,
		groupID,
	).Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get group by id: %w", err)
	}
	return &g, nil
}

// ListByUser returns all groups the user belongs to, ordered by creation date descending.
func (s *PgGroupStore) ListByUser(ctx context.Context, userID string) ([]*GroupRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT g.id, g.name, g.created_by, g.created_at
		 FROM groups g
		 JOIN group_members gm ON g.id = gm.group_id
		 WHERE gm.user_id = $1
		 ORDER BY g.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list groups by user: %w", err)
	}
	defer rows.Close()

	var groups []*GroupRecord
	for rows.Next() {
		var g GroupRecord
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedBy, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("list groups by user: scan: %w", err)
		}
		groups = append(groups, &g)
	}
	return groups, rows.Err()
}

// IsMember returns true if the user is a member of the group.
func (s *PgGroupStore) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)`,
		groupID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return exists, nil
}

// AreMembers checks whether all given user IDs are members of the group.
// Returns the set of IDs that are NOT members.
func (s *PgGroupStore) AreMembers(ctx context.Context, groupID string, userIDs []string) (nonMembers []string, err error) {
	rows, err := s.pool.Query(ctx,
		`SELECT user_id FROM group_members WHERE group_id = $1 AND user_id = ANY($2)`,
		groupID, userIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("check memberships: %w", err)
	}
	defer rows.Close()

	found := make(map[string]bool, len(userIDs))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("check memberships: scan: %w", err)
		}
		found[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("check memberships: %w", err)
	}

	for _, id := range userIDs {
		if !found[id] {
			nonMembers = append(nonMembers, id)
		}
	}
	return nonMembers, nil
}

// AddMember adds a user to a group. Returns ErrAlreadyMember if they are already a member.
func (s *PgGroupStore) AddMember(ctx context.Context, groupID, userID string) error {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadyMember
	}
	return nil
}

// GetMembers returns the user IDs of all members of the group.
func (s *PgGroupStore) GetMembers(ctx context.Context, groupID string) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT user_id FROM group_members WHERE group_id = $1 ORDER BY added_at`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("get members: scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CreateTransaction inserts a transaction and its splits within a database transaction.
// All splitBetween users must be members of the group.
func (s *PgGroupStore) CreateTransaction(ctx context.Context, groupID, description string, amount int64, paidBy string, splitBetween []string) (*TransactionRecord, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("create transaction: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var t TransactionRecord
	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (group_id, description, amount, paid_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, group_id, description, amount, paid_by, created_at`,
		groupID, description, amount, paidBy,
	).Scan(&t.ID, &t.GroupID, &t.Description, &t.Amount, &t.PaidBy, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create transaction: insert: %w", err)
	}

	for _, userID := range splitBetween {
		_, err = tx.Exec(ctx,
			`INSERT INTO transaction_splits (transaction_id, user_id) VALUES ($1, $2)`,
			t.ID, userID,
		)
		if err != nil {
			return nil, fmt.Errorf("create transaction: insert split for user %s: %w", userID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("create transaction: commit: %w", err)
	}
	return &t, nil
}

// GetTransaction returns a single transaction by ID, or ErrNotFound.
func (s *PgGroupStore) GetTransaction(ctx context.Context, transactionID string) (*TransactionRecord, error) {
	var t TransactionRecord
	err := s.pool.QueryRow(ctx,
		`SELECT id, group_id, description, amount, paid_by, created_at
		 FROM transactions WHERE id = $1`,
		transactionID,
	).Scan(&t.ID, &t.GroupID, &t.Description, &t.Amount, &t.PaidBy, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}
	return &t, nil
}

// ListByGroup returns all transactions for a group, ordered by creation date descending.
func (s *PgGroupStore) ListByGroup(ctx context.Context, groupID string) ([]*TransactionRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, group_id, description, amount, paid_by, created_at
		 FROM transactions WHERE group_id = $1
		 ORDER BY created_at DESC`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("list transactions by group: %w", err)
	}
	defer rows.Close()

	var txns []*TransactionRecord
	for rows.Next() {
		var t TransactionRecord
		if err := rows.Scan(&t.ID, &t.GroupID, &t.Description, &t.Amount, &t.PaidBy, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("list transactions by group: scan: %w", err)
		}
		txns = append(txns, &t)
	}
	return txns, rows.Err()
}

// DeleteTransaction deletes a transaction by ID. Cascade deletes its splits.
func (s *PgGroupStore) DeleteTransaction(ctx context.Context, transactionID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM transactions WHERE id = $1`,
		transactionID,
	)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetSplitUserIDs returns the user IDs in a transaction's split.
func (s *PgGroupStore) GetSplitUserIDs(ctx context.Context, transactionID string) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT user_id FROM transaction_splits WHERE transaction_id = $1`,
		transactionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get split user ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("get split user ids: scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
