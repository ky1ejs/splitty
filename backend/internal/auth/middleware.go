package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

const devAccessTokenPrefix = "dev-access-"

var publicMutations = map[string]bool{
	"signInWithApple": true,
	"sendPasscode":    true,
	"verifyPasscode":  true,
	"refreshToken":    true,
}

// tokenValidator extracts a user ID from a bearer token.
// Returns the user ID and true on success, or empty string and false on failure.
type tokenValidator func(token string) (userID string, ok bool)

// Middleware returns HTTP middleware that validates JWT bearer tokens.
// Public mutations (signInWithApple, sendPasscode, verifyPasscode,
// refreshToken) are allowed through without authentication.
func Middleware(ts *TokenService) func(http.Handler) http.Handler {
	return authMiddleware(func(token string) (string, bool) {
		claims, err := ts.ValidateAccessToken(token)
		if err != nil {
			return "", false
		}
		return claims.Subject, true
	})
}

// DevMiddleware returns HTTP middleware for development mode that validates
// dev tokens (format: "dev-access-<userID>-<hex>") without requiring JWT keys.
// Public mutations are allowed through without a token, same as production.
func DevMiddleware() func(http.Handler) http.Handler {
	return authMiddleware(parseDevAccessToken)
}

// authMiddleware creates HTTP middleware that validates bearer tokens using the
// given validator. If a token is present and valid, the user ID is injected into
// the request context. Public mutations are allowed without a token.
func authMiddleware(validate tokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, hasToken := extractBearerToken(r)

			if hasToken {
				userID, ok := validate(token)
				if !ok {
					writeUnauthorized(w, "invalid token")
					return
				}
				ctx := withUserID(r.Context(), userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if isPublicOperation(r) {
				next.ServeHTTP(w, r)
				return
			}

			writeUnauthorized(w, "authorization required")
		})
	}
}

// parseDevAccessToken extracts the user ID from a dev access token.
// Dev tokens have the format "dev-access-<userID>-<hex>" where the hex
// suffix is a 32-character random string.
func parseDevAccessToken(token string) (string, bool) {
	rest, ok := strings.CutPrefix(token, devAccessTokenPrefix)
	if !ok || rest == "" {
		return "", false
	}
	lastDash := strings.LastIndex(rest, "-")
	if lastDash <= 0 {
		return "", false
	}
	return rest[:lastDash], true
}

func extractBearerToken(r *http.Request) (string, bool) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return token, ok
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]string{{"message": msg}},
	})
}

// maxBodySize is the maximum request body size the middleware will read
// when checking for public operations (1 MB).
const maxBodySize = 1 << 20

func isPublicOperation(r *http.Request) bool {
	limited := io.LimitReader(r.Body, maxBodySize+1)
	body, err := io.ReadAll(limited)
	if err != nil || len(body) > maxBodySize {
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var req struct {
		Query         string `json:"query"`
		OperationName string `json:"operationName"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.Query == "" {
		return false
	}

	doc, parseErr := parser.ParseQuery(&ast.Source{Input: req.Query})
	if parseErr != nil {
		return false
	}

	op := findOperation(doc, req.OperationName)
	if op == nil || op.Operation != ast.Mutation {
		return false
	}

	return allSelectionsPublic(op.SelectionSet, doc)
}

func findOperation(doc *ast.QueryDocument, name string) *ast.OperationDefinition {
	if len(doc.Operations) == 0 {
		return nil
	}
	if name == "" {
		if len(doc.Operations) == 1 {
			return doc.Operations[0]
		}
		return nil
	}
	for _, op := range doc.Operations {
		if op.Name == name {
			return op
		}
	}
	return nil
}

func allSelectionsPublic(selections ast.SelectionSet, doc *ast.QueryDocument) bool {
	for _, sel := range selections {
		switch s := sel.(type) {
		case *ast.Field:
			if !publicMutations[s.Name] {
				return false
			}
		case *ast.InlineFragment:
			if !allSelectionsPublic(s.SelectionSet, doc) {
				return false
			}
		case *ast.FragmentSpread:
			frag := doc.Fragments.ForName(s.Name)
			if frag == nil || !allSelectionsPublic(frag.SelectionSet, doc) {
				return false
			}
		default:
			return false
		}
	}
	return len(selections) > 0
}
