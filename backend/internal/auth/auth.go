package auth

import "context"

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
