package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- mocks ---

type mockUserStore struct {
	upsertFn func(ctx context.Context, email string) (*UserRecord, error)
}

func (m *mockUserStore) UpsertByEmail(ctx context.Context, email string) (*UserRecord, error) {
	return m.upsertFn(ctx, email)
}

type mockTokenIssuer struct {
	issueFn func(ctx context.Context, userID, email string) (string, string, error)
}

func (m *mockTokenIssuer) IssueTokens(ctx context.Context, userID, email string) (string, string, error) {
	return m.issueFn(ctx, userID, email)
}

type passcodeRow struct {
	codeHash   string
	expiresAt  time.Time
	consumedAt time.Time
	createdAt  time.Time
}

type memPasscodeStore struct {
	mu      sync.Mutex
	byEmail map[string][]*passcodeRow
}

func newMemStore() *memPasscodeStore {
	return &memPasscodeStore{byEmail: map[string][]*passcodeRow{}}
}

func (s *memPasscodeStore) Create(_ context.Context, email, codeHash string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byEmail[email] = append(s.byEmail[email], &passcodeRow{
		codeHash:  codeHash,
		expiresAt: expiresAt,
		createdAt: time.Now(),
	})
	return nil
}

func (s *memPasscodeStore) LastIssuedAt(_ context.Context, email string) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.byEmail[email]
	if len(rows) == 0 {
		return time.Time{}, nil
	}
	return rows[len(rows)-1].createdAt, nil
}

func (s *memPasscodeStore) ConsumeMatching(_ context.Context, email, codeHash string, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.byEmail[email]
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		if r.codeHash == codeHash && r.consumedAt.IsZero() && r.expiresAt.After(now) {
			r.consumedAt = now
			return true, nil
		}
	}
	return false, nil
}

type captureSender struct {
	to, subject, body string
	err               error
}

func (c *captureSender) Send(_ context.Context, to, subject, body string) error {
	c.to, c.subject, c.body = to, subject, body
	return c.err
}

// --- helpers ---

var testUser = &UserRecord{
	ID:          "user-123",
	Email:       "alice@example.com",
	DisplayName: "alice@example.com",
}

func happyStore() *mockUserStore {
	return &mockUserStore{
		upsertFn: func(_ context.Context, _ string) (*UserRecord, error) {
			return testUser, nil
		},
	}
}

func happyTokens() *mockTokenIssuer {
	return &mockTokenIssuer{
		issueFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "access-tok", "refresh-tok", nil
		},
	}
}

func newSvc(users UserStore, tokens TokenIssuer, store PasscodeStore, sender *captureSender) *PasscodeService {
	if sender == nil {
		sender = &captureSender{}
	}
	return NewPasscodeService(users, tokens, store, sender)
}

// extractCode pulls the 6-digit code out of the email body our service writes.
func extractCode(body string) string {
	const marker = "code is "
	i := strings.Index(body, marker)
	if i < 0 {
		return ""
	}
	rest := body[i+len(marker):]
	if len(rest) < 6 {
		return ""
	}
	return rest[:6]
}

// --- SendPasscode tests ---

func TestSendPasscode_OK(t *testing.T) {
	sender := &captureSender{}
	svc := newSvc(happyStore(), happyTokens(), newMemStore(), sender)
	if err := svc.SendPasscode(context.Background(), "alice@example.com"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if sender.to != "alice@example.com" {
		t.Errorf("expected to=alice@example.com, got %q", sender.to)
	}
	if extractCode(sender.body) == "" {
		t.Errorf("expected 6-digit code in body, got %q", sender.body)
	}
}

func TestSendPasscode_EmptyEmail(t *testing.T) {
	svc := newSvc(happyStore(), happyTokens(), newMemStore(), nil)
	err := svc.SendPasscode(context.Background(), "")
	if !errors.Is(err, ErrEmailRequired) {
		t.Errorf("expected ErrEmailRequired, got %v", err)
	}
}

func TestSendPasscode_InvalidEmail(t *testing.T) {
	svc := newSvc(happyStore(), happyTokens(), newMemStore(), nil)
	err := svc.SendPasscode(context.Background(), "not-an-email")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestSendPasscode_RateLimited(t *testing.T) {
	store := newMemStore()
	svc := newSvc(happyStore(), happyTokens(), store, nil)
	if err := svc.SendPasscode(context.Background(), "alice@example.com"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	err := svc.SendPasscode(context.Background(), "alice@example.com")
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestSendPasscode_AfterRateLimit(t *testing.T) {
	store := newMemStore()
	svc := newSvc(happyStore(), happyTokens(), store, nil)
	now := time.Now()
	svc.now = func() time.Time { return now }
	if err := svc.SendPasscode(context.Background(), "alice@example.com"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	// Override stored createdAt to be in the past.
	store.byEmail["alice@example.com"][0].createdAt = now.Add(-2 * passcodeRateLimit)
	if err := svc.SendPasscode(context.Background(), "alice@example.com"); err != nil {
		t.Fatalf("second send: %v", err)
	}
}

// --- VerifyPasscode tests ---

func TestVerifyPasscode_OK(t *testing.T) {
	store := newMemStore()
	sender := &captureSender{}
	svc := newSvc(happyStore(), happyTokens(), store, sender)
	if err := svc.SendPasscode(context.Background(), "alice@example.com"); err != nil {
		t.Fatalf("send: %v", err)
	}
	code := extractCode(sender.body)
	result, err := svc.VerifyPasscode(context.Background(), "alice@example.com", code)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if result.AccessToken != "access-tok" {
		t.Errorf("expected access token, got %q", result.AccessToken)
	}
	if result.User == nil || result.User.ID != "user-123" {
		t.Errorf("unexpected user: %+v", result.User)
	}
}

func TestVerifyPasscode_BadCode(t *testing.T) {
	store := newMemStore()
	sender := &captureSender{}
	svc := newSvc(happyStore(), happyTokens(), store, sender)
	_ = svc.SendPasscode(context.Background(), "alice@example.com")

	_, err := svc.VerifyPasscode(context.Background(), "alice@example.com", "999999")
	if !errors.Is(err, ErrInvalidCode) {
		t.Errorf("expected ErrInvalidCode, got %v", err)
	}
}

func TestVerifyPasscode_Reuse(t *testing.T) {
	store := newMemStore()
	sender := &captureSender{}
	svc := newSvc(happyStore(), happyTokens(), store, sender)
	_ = svc.SendPasscode(context.Background(), "alice@example.com")
	code := extractCode(sender.body)

	if _, err := svc.VerifyPasscode(context.Background(), "alice@example.com", code); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	if _, err := svc.VerifyPasscode(context.Background(), "alice@example.com", code); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("expected ErrInvalidCode on reuse, got %v", err)
	}
}

func TestVerifyPasscode_Expired(t *testing.T) {
	store := newMemStore()
	sender := &captureSender{}
	svc := newSvc(happyStore(), happyTokens(), store, sender)
	now := time.Now()
	svc.now = func() time.Time { return now }
	_ = svc.SendPasscode(context.Background(), "alice@example.com")
	code := extractCode(sender.body)

	// Jump past TTL.
	svc.now = func() time.Time { return now.Add(passcodeTTL + time.Second) }
	if _, err := svc.VerifyPasscode(context.Background(), "alice@example.com", code); !errors.Is(err, ErrInvalidCode) {
		t.Errorf("expected ErrInvalidCode for expired code, got %v", err)
	}
}

func TestVerifyPasscode_EmptyEmail(t *testing.T) {
	svc := newSvc(happyStore(), happyTokens(), newMemStore(), nil)
	_, err := svc.VerifyPasscode(context.Background(), "", "123456")
	if !errors.Is(err, ErrEmailRequired) {
		t.Errorf("expected ErrEmailRequired, got %v", err)
	}
}

func TestVerifyPasscode_EmptyCode(t *testing.T) {
	svc := newSvc(happyStore(), happyTokens(), newMemStore(), nil)
	_, err := svc.VerifyPasscode(context.Background(), "alice@example.com", "")
	if !errors.Is(err, ErrCodeRequired) {
		t.Errorf("expected ErrCodeRequired, got %v", err)
	}
}

func TestVerifyPasscode_InvalidEmail(t *testing.T) {
	svc := newSvc(happyStore(), happyTokens(), newMemStore(), nil)
	_, err := svc.VerifyPasscode(context.Background(), "bad", "123456")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestVerifyPasscode_StoreError(t *testing.T) {
	store := newMemStore()
	sender := &captureSender{}
	users := &mockUserStore{
		upsertFn: func(_ context.Context, _ string) (*UserRecord, error) {
			return nil, errors.New("db down")
		},
	}
	svc := newSvc(users, happyTokens(), store, sender)
	_ = svc.SendPasscode(context.Background(), "alice@example.com")
	code := extractCode(sender.body)
	if _, err := svc.VerifyPasscode(context.Background(), "alice@example.com", code); err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyPasscode_TokenError(t *testing.T) {
	store := newMemStore()
	sender := &captureSender{}
	tokens := &mockTokenIssuer{
		issueFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", errors.New("token failure")
		},
	}
	svc := newSvc(happyStore(), tokens, store, sender)
	_ = svc.SendPasscode(context.Background(), "alice@example.com")
	code := extractCode(sender.body)
	if _, err := svc.VerifyPasscode(context.Background(), "alice@example.com", code); err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyPasscode_EmailNormalization(t *testing.T) {
	store := newMemStore()
	sender := &captureSender{}
	var captured string
	users := &mockUserStore{
		upsertFn: func(_ context.Context, email string) (*UserRecord, error) {
			captured = email
			return testUser, nil
		},
	}
	svc := newSvc(users, happyTokens(), store, sender)
	if err := svc.SendPasscode(context.Background(), "  Alice@Example.COM  "); err != nil {
		t.Fatalf("send: %v", err)
	}
	code := extractCode(sender.body)
	if _, err := svc.VerifyPasscode(context.Background(), "  Alice@Example.COM  ", code); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if captured != "alice@example.com" {
		t.Errorf("expected normalized email, got %q", captured)
	}
}
