package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/kylejs/splitty/backend/internal/email"
)

// Tunable constants for the email-passcode flow.
const (
	passcodeTTL       = 10 * time.Minute
	passcodeRateLimit = 60 * time.Second
)

// PasscodeService implements the email passcode auth flow.
//
// In all environments, codes are generated, hashed, and persisted with a
// short TTL. The Sender controls delivery: in development, a LogSender writes
// the code to logs; in production, MailgunSender delivers the email.
type PasscodeService struct {
	users  UserStore
	tokens TokenIssuer
	store  PasscodeStore
	sender email.Sender
	now    func() time.Time
}

// NewPasscodeService creates a PasscodeService.
func NewPasscodeService(users UserStore, tokens TokenIssuer, store PasscodeStore, sender email.Sender) *PasscodeService {
	return &PasscodeService{
		users:  users,
		tokens: tokens,
		store:  store,
		sender: sender,
		now:    time.Now,
	}
}

// SendPasscode generates a 6-digit code, persists its hash, and dispatches
// the plaintext code via the configured Sender.
func (s *PasscodeService) SendPasscode(ctx context.Context, rawEmail string) error {
	emailAddr, err := NormalizeEmail(rawEmail)
	if err != nil {
		return err
	}

	now := s.now()
	last, err := s.store.LastIssuedAt(ctx, emailAddr)
	if err != nil {
		return err
	}
	if !last.IsZero() && now.Sub(last) < passcodeRateLimit {
		return ErrRateLimited
	}

	code, err := generatePasscode()
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	if err := s.store.Create(ctx, emailAddr, hashCode(code), now.Add(passcodeTTL)); err != nil {
		return err
	}

	subject := "Your Splitty sign-in code"
	body := fmt.Sprintf("Your Splitty sign-in code is %s. It expires in %d minutes.", code, int(passcodeTTL.Minutes()))
	if err := s.sender.Send(ctx, emailAddr, subject, body); err != nil {
		slog.Error("send passcode email", "email", emailAddr, "err", err)
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// VerifyPasscode validates a code for the given email, consumes it, and
// issues an auth result.
func (s *PasscodeService) VerifyPasscode(ctx context.Context, rawEmail, code string) (*AuthResult, error) {
	emailAddr, err := NormalizeEmail(rawEmail)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(code) == "" {
		return nil, ErrCodeRequired
	}

	matched, err := s.store.ConsumeMatching(ctx, emailAddr, hashCode(code), s.now())
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, ErrInvalidCode
	}

	user, err := s.users.UpsertByEmail(ctx, emailAddr)
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

// generatePasscode returns a uniformly-random 6-digit numeric string.
func generatePasscode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(sum[:])
}
