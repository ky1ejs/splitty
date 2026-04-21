package auth

import (
	"context"
	"errors"
	"testing"

	splittyv1 "github.com/kylejs/splitty/backend/gen/splitty/v1"
	"github.com/kylejs/splitty/backend/internal/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func newTestServer(env string, users UserStore, tokens TokenIssuer) *AuthServer {
	return NewAuthServer(env, users, tokens)
}

func assertCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != want {
		t.Errorf("expected code %v, got %v: %s", want, st.Code(), st.Message())
	}
}

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
	srv := newTestServer(config.EnvDevelopment, happyStore(), happyTokens())
	resp, err := srv.SendPasscode(context.Background(), &splittyv1.SendPasscodeRequest{
		Email: "alice@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestSendPasscode_Prod_Unavailable(t *testing.T) {
	srv := newTestServer(config.EnvProduction, happyStore(), happyTokens())
	_, err := srv.SendPasscode(context.Background(), &splittyv1.SendPasscodeRequest{
		Email: "alice@example.com",
	})
	assertCode(t, err, codes.Unavailable)
}

func TestSendPasscode_UnknownEnv_Unavailable(t *testing.T) {
	srv := newTestServer("staging", happyStore(), happyTokens())
	_, err := srv.SendPasscode(context.Background(), &splittyv1.SendPasscodeRequest{
		Email: "alice@example.com",
	})
	assertCode(t, err, codes.Unavailable)
}

func TestSendPasscode_EmptyEmail(t *testing.T) {
	srv := newTestServer(config.EnvDevelopment, happyStore(), happyTokens())
	_, err := srv.SendPasscode(context.Background(), &splittyv1.SendPasscodeRequest{
		Email: "",
	})
	assertCode(t, err, codes.InvalidArgument)
}

func TestSendPasscode_InvalidEmail(t *testing.T) {
	srv := newTestServer(config.EnvDevelopment, happyStore(), happyTokens())
	_, err := srv.SendPasscode(context.Background(), &splittyv1.SendPasscodeRequest{
		Email: "not-an-email",
	})
	assertCode(t, err, codes.InvalidArgument)
}

// --- VerifyPasscode tests ---

func TestVerifyPasscode_Dev_OK(t *testing.T) {
	srv := newTestServer(config.EnvDevelopment, happyStore(), happyTokens())
	resp, err := srv.VerifyPasscode(context.Background(), &splittyv1.VerifyPasscodeRequest{
		Email: "alice@example.com",
		Code:  "123456",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "access-tok" {
		t.Errorf("expected access token %q, got %q", "access-tok", resp.AccessToken)
	}
	if resp.RefreshToken != "refresh-tok" {
		t.Errorf("expected refresh token %q, got %q", "refresh-tok", resp.RefreshToken)
	}
	if resp.User == nil {
		t.Fatal("expected non-nil user")
	}
	if resp.User.Id != "user-123" {
		t.Errorf("expected user id %q, got %q", "user-123", resp.User.Id)
	}
	if resp.User.Email != "alice@example.com" {
		t.Errorf("expected user email %q, got %q", "alice@example.com", resp.User.Email)
	}
}

func TestVerifyPasscode_Prod_Unavailable(t *testing.T) {
	srv := newTestServer(config.EnvProduction, happyStore(), happyTokens())
	_, err := srv.VerifyPasscode(context.Background(), &splittyv1.VerifyPasscodeRequest{
		Email: "alice@example.com",
		Code:  "123456",
	})
	assertCode(t, err, codes.Unavailable)
}

func TestVerifyPasscode_EmptyEmail(t *testing.T) {
	srv := newTestServer(config.EnvDevelopment, happyStore(), happyTokens())
	_, err := srv.VerifyPasscode(context.Background(), &splittyv1.VerifyPasscodeRequest{
		Email: "",
		Code:  "123456",
	})
	assertCode(t, err, codes.InvalidArgument)
}

func TestVerifyPasscode_EmptyCode(t *testing.T) {
	srv := newTestServer(config.EnvDevelopment, happyStore(), happyTokens())
	_, err := srv.VerifyPasscode(context.Background(), &splittyv1.VerifyPasscodeRequest{
		Email: "alice@example.com",
		Code:  "",
	})
	assertCode(t, err, codes.InvalidArgument)
}

func TestVerifyPasscode_InvalidEmail(t *testing.T) {
	srv := newTestServer(config.EnvDevelopment, happyStore(), happyTokens())
	_, err := srv.VerifyPasscode(context.Background(), &splittyv1.VerifyPasscodeRequest{
		Email: "bad",
		Code:  "123456",
	})
	assertCode(t, err, codes.InvalidArgument)
}

func TestVerifyPasscode_StoreError(t *testing.T) {
	store := &mockUserStore{
		upsertFn: func(_ context.Context, _ string) (*UserRecord, error) {
			return nil, errors.New("db down")
		},
	}
	srv := newTestServer(config.EnvDevelopment, store, happyTokens())
	_, err := srv.VerifyPasscode(context.Background(), &splittyv1.VerifyPasscodeRequest{
		Email: "alice@example.com",
		Code:  "123456",
	})
	assertCode(t, err, codes.Internal)
}

func TestVerifyPasscode_TokenError(t *testing.T) {
	tokens := &mockTokenIssuer{
		issueFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", errors.New("token failure")
		},
	}
	srv := newTestServer(config.EnvDevelopment, happyStore(), tokens)
	_, err := srv.VerifyPasscode(context.Background(), &splittyv1.VerifyPasscodeRequest{
		Email: "alice@example.com",
		Code:  "123456",
	})
	assertCode(t, err, codes.Internal)
}

func TestVerifyPasscode_EmailNormalization(t *testing.T) {
	var captured string
	store := &mockUserStore{
		upsertFn: func(_ context.Context, email string) (*UserRecord, error) {
			captured = email
			return testUser, nil
		},
	}
	srv := newTestServer(config.EnvDevelopment, store, happyTokens())
	_, err := srv.VerifyPasscode(context.Background(), &splittyv1.VerifyPasscodeRequest{
		Email: "  Alice@Example.COM  ",
		Code:  "123456",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured != "alice@example.com" {
		t.Errorf("expected normalized email %q, got %q", "alice@example.com", captured)
	}
}
