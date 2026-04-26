package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PasscodeStore persists hashed passcodes for the email-passcode auth flow.
type PasscodeStore interface {
	// Create stores a new passcode for the email.
	Create(ctx context.Context, email, codeHash string, expiresAt time.Time) error
	// LastIssuedAt returns the created_at of the most recently issued passcode
	// for the email, or zero time if none exists. Used for rate limiting.
	LastIssuedAt(ctx context.Context, email string) (time.Time, error)
	// ConsumeMatching atomically finds an unconsumed, unexpired passcode for
	// the email whose hash matches and marks it consumed. Returns true on
	// match, false if no such row.
	ConsumeMatching(ctx context.Context, email, codeHash string, now time.Time) (bool, error)
}

// PgPasscodeStore implements PasscodeStore using a pgx pool.
type PgPasscodeStore struct {
	pool *pgxpool.Pool
}

// NewPgPasscodeStore creates a PgPasscodeStore.
func NewPgPasscodeStore(pool *pgxpool.Pool) *PgPasscodeStore {
	return &PgPasscodeStore{pool: pool}
}

func (s *PgPasscodeStore) Create(ctx context.Context, email, codeHash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO email_passcodes (email, code_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		email, codeHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create passcode: %w", err)
	}
	return nil
}

func (s *PgPasscodeStore) LastIssuedAt(ctx context.Context, email string) (time.Time, error) {
	var t time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT created_at FROM email_passcodes
		 WHERE email = $1
		 ORDER BY created_at DESC
		 LIMIT 1`,
		email,
	).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("last issued: %w", err)
	}
	return t, nil
}

func (s *PgPasscodeStore) ConsumeMatching(ctx context.Context, email, codeHash string, now time.Time) (bool, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`UPDATE email_passcodes
		 SET consumed_at = $4
		 WHERE id = (
		     SELECT id FROM email_passcodes
		     WHERE email = $1
		       AND code_hash = $2
		       AND consumed_at IS NULL
		       AND expires_at > $3
		     ORDER BY created_at DESC
		     LIMIT 1
		     FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id`,
		email, codeHash, now, now,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("consume passcode: %w", err)
	}
	return true, nil
}
