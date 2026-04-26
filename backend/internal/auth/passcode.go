package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/kylejs/splitty/backend/internal/config"
)

// PasscodeService implements the email passcode auth flow.
// Behavior varies by environment: development accepts any code,
// non-development returns ErrUnavailable.
type PasscodeService struct {
	env    string
	users  UserStore
	tokens TokenIssuer
}

// NewPasscodeService creates a PasscodeService with the given dependencies.
func NewPasscodeService(env string, users UserStore, tokens TokenIssuer) *PasscodeService {
	return &PasscodeService{
		env:    env,
		users:  users,
		tokens: tokens,
	}
}

func (s *PasscodeService) SendPasscode(_ context.Context, email string) error {
	if s.env != config.EnvDevelopment {
		return ErrUnavailable
	}

	email, err := NormalizeEmail(email)
	if err != nil {
		return err
	}

	slog.Info("passcode requested (any code accepted in dev mode)", "email", email)
	return nil
}

func (s *PasscodeService) VerifyPasscode(ctx context.Context, email, code string) (*AuthResult, error) {
	if s.env != config.EnvDevelopment {
		return nil, ErrUnavailable
	}

	email, err := NormalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(code) == "" {
		return nil, ErrCodeRequired
	}

	user, err := s.users.UpsertByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	accessToken, refreshToken, err := s.tokens.IssueTokens(ctx, user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

// NormalizeEmail validates and normalizes an email address.
func NormalizeEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrEmailRequired
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", ErrInvalidEmail
	}
	return strings.ToLower(addr.Address), nil
}
