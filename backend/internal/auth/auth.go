package auth

import (
	"context"
	"errors"
)

var (
	ErrUnavailable    = errors.New("email passcode is not available")
	ErrInvalidEmail   = errors.New("invalid email address")
	ErrEmailRequired  = errors.New("email is required")
	ErrCodeRequired   = errors.New("code is required")
)

// TokenIssuer generates access and refresh tokens for a user.
type TokenIssuer interface {
	IssueTokens(ctx context.Context, userID, email string) (accessToken, refreshToken string, err error)
}

// UserStore manages user persistence for the auth package.
type UserStore interface {
	UpsertByEmail(ctx context.Context, email string) (*UserRecord, error)
}

// UserRecord is the domain model for a user returned from the store.
type UserRecord struct {
	ID          string
	Email       string
	DisplayName string
}

// AuthResult is the domain result of a successful authentication.
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User         *UserRecord
}
