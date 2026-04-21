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
