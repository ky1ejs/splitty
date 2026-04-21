package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// DevTokenIssuer is a TokenIssuer that returns random placeholder tokens.
// It is intended for development mode only.
type DevTokenIssuer struct{}

func (d *DevTokenIssuer) IssueTokens(_ context.Context, userID, _ string) (string, string, error) {
	access, err := randomHex(16)
	if err != nil {
		return "", "", fmt.Errorf("generate dev access token: %w", err)
	}
	refresh, err := randomHex(16)
	if err != nil {
		return "", "", fmt.Errorf("generate dev refresh token: %w", err)
	}
	return "dev-access-" + userID + "-" + access, "dev-refresh-" + userID + "-" + refresh, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
