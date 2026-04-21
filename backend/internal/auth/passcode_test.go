package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/kylejs/splitty/backend/internal/config"
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

// --- SendPasscode tests ---

func TestSendPasscode_Dev_OK(t *testing.T) {
	svc := NewPasscodeService(config.EnvDevelopment, happyStore(), happyTokens())
	err := svc.SendPasscode(context.Background(), "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendPasscode_Prod_Unavailable(t *testing.T) {
	svc := NewPasscodeService(config.EnvProduction, happyStore(), happyTokens())
	err := svc.SendPasscode(context.Background(), "alice@example.com")
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("expected ErrUnavailable, got %v", err)
	}
}

func TestSendPasscode_UnknownEnv_Unavailable(t *testing.T) {
	svc := NewPasscodeService("staging", happyStore(), happyTokens())
	err := svc.SendPasscode(context.Background(), "alice@example.com")
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("expected ErrUnavailable, got %v", err)
	}
}

func TestSendPasscode_EmptyEmail(t *testing.T) {
	svc := NewPasscodeService(config.EnvDevelopment, happyStore(), happyTokens())
	err := svc.SendPasscode(context.Background(), "")
	if !errors.Is(err, ErrEmailRequired) {
		t.Errorf("expected ErrEmailRequired, got %v", err)
	}
}

func TestSendPasscode_InvalidEmail(t *testing.T) {
	svc := NewPasscodeService(config.EnvDevelopment, happyStore(), happyTokens())
	err := svc.SendPasscode(context.Background(), "not-an-email")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

// --- VerifyPasscode tests ---

func TestVerifyPasscode_Dev_OK(t *testing.T) {
	svc := NewPasscodeService(config.EnvDevelopment, happyStore(), happyTokens())
	result, err := svc.VerifyPasscode(context.Background(), "alice@example.com", "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken != "access-tok" {
		t.Errorf("expected access token %q, got %q", "access-tok", result.AccessToken)
	}
	if result.RefreshToken != "refresh-tok" {
		t.Errorf("expected refresh token %q, got %q", "refresh-tok", result.RefreshToken)
	}
	if result.User == nil {
		t.Fatal("expected non-nil user")
	}
	if result.User.ID != "user-123" {
		t.Errorf("expected user id %q, got %q", "user-123", result.User.ID)
	}
	if result.User.Email != "alice@example.com" {
		t.Errorf("expected user email %q, got %q", "alice@example.com", result.User.Email)
	}
}

func TestVerifyPasscode_Prod_Unavailable(t *testing.T) {
	svc := NewPasscodeService(config.EnvProduction, happyStore(), happyTokens())
	_, err := svc.VerifyPasscode(context.Background(), "alice@example.com", "123456")
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("expected ErrUnavailable, got %v", err)
	}
}

func TestVerifyPasscode_EmptyEmail(t *testing.T) {
	svc := NewPasscodeService(config.EnvDevelopment, happyStore(), happyTokens())
	_, err := svc.VerifyPasscode(context.Background(), "", "123456")
	if !errors.Is(err, ErrEmailRequired) {
		t.Errorf("expected ErrEmailRequired, got %v", err)
	}
}

func TestVerifyPasscode_EmptyCode(t *testing.T) {
	svc := NewPasscodeService(config.EnvDevelopment, happyStore(), happyTokens())
	_, err := svc.VerifyPasscode(context.Background(), "alice@example.com", "")
	if !errors.Is(err, ErrCodeRequired) {
		t.Errorf("expected ErrCodeRequired, got %v", err)
	}
}

func TestVerifyPasscode_InvalidEmail(t *testing.T) {
	svc := NewPasscodeService(config.EnvDevelopment, happyStore(), happyTokens())
	_, err := svc.VerifyPasscode(context.Background(), "bad", "123456")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestVerifyPasscode_StoreError(t *testing.T) {
	store := &mockUserStore{
		upsertFn: func(_ context.Context, _ string) (*UserRecord, error) {
			return nil, errors.New("db down")
		},
	}
	svc := NewPasscodeService(config.EnvDevelopment, store, happyTokens())
	_, err := svc.VerifyPasscode(context.Background(), "alice@example.com", "123456")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyPasscode_TokenError(t *testing.T) {
	tokens := &mockTokenIssuer{
		issueFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", errors.New("token failure")
		},
	}
	svc := NewPasscodeService(config.EnvDevelopment, happyStore(), tokens)
	_, err := svc.VerifyPasscode(context.Background(), "alice@example.com", "123456")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyPasscode_EmailNormalization(t *testing.T) {
	var captured string
	store := &mockUserStore{
		upsertFn: func(_ context.Context, email string) (*UserRecord, error) {
			captured = email
			return testUser, nil
		},
	}
	svc := NewPasscodeService(config.EnvDevelopment, store, happyTokens())
	_, err := svc.VerifyPasscode(context.Background(), "  Alice@Example.COM  ", "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured != "alice@example.com" {
		t.Errorf("expected normalized email %q, got %q", "alice@example.com", captured)
	}
}
