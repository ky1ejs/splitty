package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func generateTestKey(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return key, string(pemBytes)
}

func newTestTokenService(t *testing.T) *TokenService {
	t.Helper()
	_, pemKey := generateTestKey(t)
	ts, err := NewTokenService(pemKey, nil)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}
	return ts
}

func TestMiddleware(t *testing.T) {
	ts := newTestTokenService(t)
	validToken, err := ts.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	// Handler that records the user ID from context.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromContext(r.Context())
		if ok {
			w.Write([]byte(uid))
		} else {
			w.Write([]byte("no-user"))
		}
	})

	mw := Middleware(ts)(handler)

	tests := []struct {
		name       string
		auth       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid token on protected query",
			auth:       "Bearer " + validToken,
			body:       `{"query":"{ me { id } }"}`,
			wantStatus: http.StatusOK,
			wantBody:   "user-123",
		},
		{
			name:       "missing token on protected query",
			body:       `{"query":"{ me { id } }"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token on protected query",
			auth:       "Bearer invalid.jwt.here",
			body:       `{"query":"{ me { id } }"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing Bearer prefix",
			auth:       validToken,
			body:       `{"query":"{ me { id } }"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "sendPasscode without token",
			body:       `{"query":"mutation { sendPasscode(email: \"a@b.com\") { success } }"}`,
			wantStatus: http.StatusOK,
			wantBody:   "no-user",
		},
		{
			name:       "verifyPasscode without token",
			body:       `{"query":"mutation { verifyPasscode(email: \"a@b.com\", code: \"123456\") { accessToken } }"}`,
			wantStatus: http.StatusOK,
			wantBody:   "no-user",
		},
		{
			name:       "refreshToken without token",
			body:       `{"query":"mutation { refreshToken(refreshToken: \"tok\") { accessToken } }"}`,
			wantStatus: http.StatusOK,
			wantBody:   "no-user",
		},
		{
			name:       "signInWithApple without token",
			body:       `{"query":"mutation { signInWithApple(identityToken: \"tok\") { accessToken } }"}`,
			wantStatus: http.StatusOK,
			wantBody:   "no-user",
		},
		{
			name:       "non-auth mutation without token",
			body:       `{"query":"mutation { deleteAccount { success } }"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "mixed auth and non-auth mutations without token",
			body:       `{"query":"mutation { sendPasscode(email: \"a@b.com\") { success } deleteAccount { success } }"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "auth mutation with valid token still injects user",
			auth:       "Bearer " + validToken,
			body:       `{"query":"mutation { sendPasscode(email: \"a@b.com\") { success } }"}`,
			wantStatus: http.StatusOK,
			wantBody:   "user-123",
		},
		{
			name:       "multi-operation with operationName selecting public mutation",
			body:       `{"query":"mutation Public { sendPasscode(email: \"a@b.com\") { success } } query Private { me { id } }","operationName":"Public"}`,
			wantStatus: http.StatusOK,
			wantBody:   "no-user",
		},
		{
			name:       "multi-operation with operationName selecting protected query",
			body:       `{"query":"mutation Public { sendPasscode(email: \"a@b.com\") { success } } query Private { me { id } }","operationName":"Private"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "multi-operation without operationName",
			body:       `{"query":"mutation A { sendPasscode(email: \"a@b.com\") { success } } mutation B { refreshToken(refreshToken: \"t\") { accessToken } }"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "public mutation via inline fragment",
			body:       `{"query":"mutation { ... { sendPasscode(email: \"a@b.com\") { success } } }"}`,
			wantStatus: http.StatusOK,
			wantBody:   "no-user",
		},
		{
			name:       "public mutation via named fragment spread",
			body:       `{"query":"fragment F on Mutation { sendPasscode(email: \"a@b.com\") { success } } mutation { ...F }"}`,
			wantStatus: http.StatusOK,
			wantBody:   "no-user",
		},
		{
			name:       "protected mutation via named fragment spread",
			body:       `{"query":"fragment F on Mutation { deleteAccount { success } } mutation { ...F }"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "oversized body without token",
			body:       `{"query":"` + strings.Repeat("x", 2<<20) + `"}`,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}

			w := httptest.NewRecorder()
			mw.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				body, _ := io.ReadAll(w.Result().Body)
				t.Errorf("got status %d, want %d (body: %s)", w.Code, tt.wantStatus, body)
			}
			if tt.wantBody != "" {
				got := w.Body.String()
				if got != tt.wantBody {
					t.Errorf("got body %q, want %q", got, tt.wantBody)
				}
			}
		})
	}
}
