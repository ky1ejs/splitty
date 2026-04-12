package auth

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	issuer          = "splitty"
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 90 * 24 * time.Hour
)

// Claims represents the JWT claims for a Splitty access token.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// TokenService handles JWT access token and opaque refresh token operations.
type TokenService struct {
	privateKey *rsa.PrivateKey
	pool       *pgxpool.Pool
}

// NewTokenService creates a TokenService from a PEM-encoded RSA private key
// and a Postgres connection pool.
func NewTokenService(pemKey string, pool *pgxpool.Pool) (*TokenService, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, fmt.Errorf("auth: parse private key: no PEM block found")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		pkcs1Key, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if pkcs1Err != nil {
			return nil, fmt.Errorf("auth: parse private key: %w", err)
		}
		key = pkcs1Key
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("auth: parse private key: not an RSA key")
	}

	return &TokenService{
		privateKey: rsaKey,
		pool:       pool,
	}, nil
}

// GenerateAccessToken creates an RS256 JWT with a 15 minute TTL.
func (s *TokenService) GenerateAccessToken(userID, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			Issuer:    issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("auth: sign access token: %w", err)
	}
	return signed, nil
}

// ValidateAccessToken parses and validates an RS256 JWT, returning the claims.
func (s *TokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", token.Header["alg"])
		}
		return &s.privateKey.PublicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: validate access token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("auth: validate access token: token is not valid")
	}
	return claims, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// GenerateRefreshToken creates an opaque UUID refresh token, stores its
// SHA-256 hash in Postgres with a 90 day expiry, and returns the raw token.
func (s *TokenService) GenerateRefreshToken(ctx context.Context, userID string) (string, error) {
	rawToken := uuid.NewString()
	tokenHash := hashToken(rawToken)
	expiresAt := time.Now().Add(refreshTokenTTL)

	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("auth: store refresh token: %w", err)
	}
	return rawToken, nil
}

// ValidateRefreshToken looks up the SHA-256 hash of the given token in Postgres,
// checks expiry, and returns the owning user ID.
func (s *TokenService) ValidateRefreshToken(ctx context.Context, tokenString string) (string, error) {
	tokenHash := hashToken(tokenString)

	var userID string
	err := s.pool.QueryRow(ctx,
		`SELECT user_id FROM refresh_tokens WHERE token_hash = $1 AND expires_at > now()`,
		tokenHash,
	).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("auth: validate refresh token: %w", err)
	}

	return userID, nil
}

// RotateRefreshToken atomically deletes the old refresh token and inserts a
// new one in a single statement. Returns an error if the old token is not found
// or does not belong to the given user.
func (s *TokenService) RotateRefreshToken(ctx context.Context, oldToken, userID string) (string, error) {
	oldHash := hashToken(oldToken)
	newToken := uuid.NewString()
	newHash := hashToken(newToken)
	expiresAt := time.Now().Add(refreshTokenTTL)

	tag, err := s.pool.Exec(ctx,
		`WITH deleted AS (
			DELETE FROM refresh_tokens WHERE token_hash = $1 AND user_id = $2 RETURNING id
		)
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		SELECT $2, $3, $4 FROM deleted`,
		oldHash, userID, newHash, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("auth: rotate refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return "", fmt.Errorf("auth: rotate refresh token: old token not found")
	}

	return newToken, nil
}
