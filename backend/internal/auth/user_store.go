package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgUserStore implements UserStore using a pgx connection pool.
type PgUserStore struct {
	pool *pgxpool.Pool
}

// NewPgUserStore creates a PgUserStore backed by the given pool.
func NewPgUserStore(pool *pgxpool.Pool) *PgUserStore {
	return &PgUserStore{pool: pool}
}

// GetByID returns the user with the given ID, or an error if not found.
func (s *PgUserStore) GetByID(ctx context.Context, id string) (*UserRecord, error) {
	var u UserRecord
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, display_name FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.DisplayName)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

// GetByIDs returns users matching the given IDs.
func (s *PgUserStore) GetByIDs(ctx context.Context, ids []string) ([]*UserRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, email, display_name FROM users
		 WHERE id = ANY($1::uuid[])
		 ORDER BY array_position($1::uuid[], id)`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("get users by ids: %w", err)
	}
	defer rows.Close()

	var users []*UserRecord
	for rows.Next() {
		var u UserRecord
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName); err != nil {
			return nil, fmt.Errorf("get users by ids: scan: %w", err)
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

// GetByEmail returns the user with the given email, or an error if not found.
func (s *PgUserStore) GetByEmail(ctx context.Context, email string) (*UserRecord, error) {
	var u UserRecord
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, display_name FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.DisplayName)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

func (s *PgUserStore) UpsertByEmail(ctx context.Context, email string) (*UserRecord, error) {
	var u UserRecord
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, display_name)
		 VALUES ($1, $1)
		 ON CONFLICT (email) DO UPDATE SET updated_at = now()
		 RETURNING id, email, display_name`,
		email,
	).Scan(&u.ID, &u.Email, &u.DisplayName)
	if err != nil {
		return nil, fmt.Errorf("upsert user by email: %w", err)
	}
	return &u, nil
}
