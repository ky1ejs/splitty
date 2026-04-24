package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kylejs/splitty/backend/internal/db"
)

func testKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return key
}

func testTokenService(t *testing.T) *TokenService {
	t.Helper()
	return &TokenService{privateKey: testKey(t)}
}

func encodePEM(key *rsa.PrivateKey, pkcs8 bool) (string, error) {
	var der []byte
	var blockType string
	if pkcs8 {
		var err error
		der, err = x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return "", err
		}
		blockType = "PRIVATE KEY"
	} else {
		der = x509.MarshalPKCS1PrivateKey(key)
		blockType = "RSA PRIVATE KEY"
	}
	block := &pem.Block{Type: blockType, Bytes: der}
	return string(pem.EncodeToMemory(block)), nil
}

// --- Access token tests (pure, no DB) ---

func TestGenerateAndValidateAccessToken(t *testing.T) {
	svc := testTokenService(t)
	userID := uuid.NewString()
	email := "test@example.com"

	tokenStr, err := svc.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := svc.ValidateAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %q, want %q", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("Email = %q, want %q", claims.Email, email)
	}
	if claims.Subject != userID {
		t.Errorf("Subject = %q, want %q", claims.Subject, userID)
	}
	if claims.Issuer != issuer {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, issuer)
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	svc := testTokenService(t)

	userID := uuid.NewString()
	claims := Claims{
		UserID: userID,
		Email:  "expired@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			Issuer:    issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(svc.privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = svc.ValidateAccessToken(signed)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateAccessToken_WrongKey(t *testing.T) {
	svc1 := testTokenService(t)
	svc2 := testTokenService(t)

	tokenStr, err := svc1.GenerateAccessToken(uuid.NewString(), "a@b.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	_, err = svc2.ValidateAccessToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong key, got nil")
	}
}

func TestValidateAccessToken_WrongAlgorithm(t *testing.T) {
	svc := testTokenService(t)

	// Create an HMAC-signed token
	claims := jwt.RegisteredClaims{
		Subject:   uuid.NewString(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign HMAC token: %v", err)
	}

	_, err = svc.ValidateAccessToken(signed)
	if err == nil {
		t.Fatal("expected error for wrong algorithm, got nil")
	}
}

func TestNewTokenService_PEMFormats(t *testing.T) {
	for _, tc := range []struct {
		name  string
		pkcs8 bool
	}{
		{"PKCS8", true},
		{"PKCS1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := testKey(t)
			pemStr, err := encodePEM(key, tc.pkcs8)
			if err != nil {
				t.Fatalf("encode PEM: %v", err)
			}

			svc, err := NewTokenService(pemStr, nil)
			if err != nil {
				t.Fatalf("NewTokenService: %v", err)
			}

			tokenStr, err := svc.GenerateAccessToken("user-1", "test@example.com")
			if err != nil {
				t.Fatalf("GenerateAccessToken: %v", err)
			}
			if _, err := svc.ValidateAccessToken(tokenStr); err != nil {
				t.Fatalf("ValidateAccessToken: %v", err)
			}
		})
	}
}

func TestNewTokenService_InvalidPEM(t *testing.T) {
	_, err := NewTokenService("not-a-pem", nil)
	if err == nil {
		t.Fatal("expected error for invalid PEM, got nil")
	}
}

func TestHashToken(t *testing.T) {
	// SHA-256 of empty string is a known constant
	got := hashToken("")
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("hashToken(\"\") = %q, want %q", got, want)
	}

	// Same input produces same output
	token := uuid.NewString()
	if hashToken(token) != hashToken(token) {
		t.Error("hashToken is not deterministic")
	}
}

// --- Refresh token tests (integration, require DATABASE_URL) ---

func testTokenServiceWithDB(t *testing.T) (*TokenService, *pgxpool.Pool) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	key := testKey(t)
	return &TokenService{privateKey: key, pool: pool}, pool
}

func createTestUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()
	userID := uuid.NewString()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, display_name) VALUES ($1, $2)`,
		userID, "test-user",
	)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
			t.Errorf("cleanup delete user %q: %v", userID, err)
		}
	})
	return userID
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	svc, pool := testTokenServiceWithDB(t)
	userID := createTestUser(t, pool)
	ctx := context.Background()

	rawToken, err := svc.GenerateRefreshToken(ctx, userID)
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	gotUserID, err := svc.ValidateRefreshToken(ctx, rawToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken: %v", err)
	}
	if gotUserID != userID {
		t.Errorf("userID = %q, want %q", gotUserID, userID)
	}
}

func TestValidateRefreshToken_Invalid(t *testing.T) {
	svc, _ := testTokenServiceWithDB(t)
	ctx := context.Background()

	_, err := svc.ValidateRefreshToken(ctx, uuid.NewString())
	if err == nil {
		t.Fatal("expected error for invalid refresh token, got nil")
	}
}

func TestValidateRefreshToken_Expired(t *testing.T) {
	svc, pool := testTokenServiceWithDB(t)
	userID := createTestUser(t, pool)
	ctx := context.Background()

	// Insert a token that expired an hour ago
	rawToken := uuid.NewString()
	tokenHash := hashToken(rawToken)
	_, err := pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, time.Now().Add(-1*time.Hour),
	)
	if err != nil {
		t.Fatalf("insert expired token: %v", err)
	}

	_, err = svc.ValidateRefreshToken(ctx, rawToken)
	if err == nil {
		t.Fatal("expected error for expired refresh token, got nil")
	}
}

func TestRotateRefreshToken(t *testing.T) {
	svc, pool := testTokenServiceWithDB(t)
	userID := createTestUser(t, pool)
	ctx := context.Background()

	oldToken, err := svc.GenerateRefreshToken(ctx, userID)
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	newToken, err := svc.RotateRefreshToken(ctx, oldToken, userID)
	if err != nil {
		t.Fatalf("RotateRefreshToken: %v", err)
	}

	// Old token should be invalid
	_, err = svc.ValidateRefreshToken(ctx, oldToken)
	if err == nil {
		t.Error("expected old token to be invalid after rotation")
	}

	// New token should be valid
	gotUserID, err := svc.ValidateRefreshToken(ctx, newToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken(new): %v", err)
	}
	if gotUserID != userID {
		t.Errorf("userID = %q, want %q", gotUserID, userID)
	}
}

func TestRotateRefreshToken_ExpiredOldToken(t *testing.T) {
	svc, pool := testTokenServiceWithDB(t)
	userID := createTestUser(t, pool)
	ctx := context.Background()

	oldToken := uuid.NewString()
	tokenHash := hashToken(oldToken)
	_, err := pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, time.Now().Add(-1*time.Hour),
	)
	if err != nil {
		t.Fatalf("insert expired token: %v", err)
	}

	_, err = svc.RotateRefreshToken(ctx, oldToken, userID)
	if err == nil {
		t.Fatal("expected error when rotating expired refresh token, got nil")
	}
}

func TestRotateRefreshToken_InvalidOldToken(t *testing.T) {
	svc, _ := testTokenServiceWithDB(t)
	ctx := context.Background()

	_, err := svc.RotateRefreshToken(ctx, uuid.NewString(), uuid.NewString())
	if err == nil {
		t.Fatal("expected error for invalid old token, got nil")
	}
}

func TestRotateRefreshToken_WrongUser(t *testing.T) {
	svc, pool := testTokenServiceWithDB(t)
	userA := createTestUser(t, pool)
	userB := createTestUser(t, pool)
	ctx := context.Background()

	token, err := svc.GenerateRefreshToken(ctx, userA)
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	// Try to rotate user A's token as user B
	_, err = svc.RotateRefreshToken(ctx, token, userB)
	if err == nil {
		t.Fatal("expected error when rotating with wrong user, got nil")
	}

	// Original token should still be valid (rotation failed)
	gotUserID, err := svc.ValidateRefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("original token should still be valid: %v", err)
	}
	if gotUserID != userA {
		t.Errorf("userID = %q, want %q", gotUserID, userA)
	}
}
