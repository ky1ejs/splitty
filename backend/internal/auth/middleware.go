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

var publicMutations = map[string]bool{
	"signInWithApple": true,
	"sendPasscode":    true,
	"verifyPasscode":  true,
	"refreshToken":    true,
}

// Middleware returns HTTP middleware that validates JWT bearer tokens.
// Public mutations (signInWithApple, sendPasscode, verifyPasscode,
// refreshToken) are allowed through without authentication.
func Middleware(ts *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, hasToken := extractBearerToken(r)

			if hasToken {
				claims, err := ts.ValidateAccessToken(token)
				if err != nil {
					writeUnauthorized(w, "invalid token")
					return
				}
				ctx := withUserID(r.Context(), claims.Subject)
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

func isPublicOperation(r *http.Request) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var req struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.Query == "" {
		return false
	}

	doc, parseErr := parser.ParseQuery(&ast.Source{Input: req.Query})
	if parseErr != nil {
		return false
	}

	if len(doc.Operations) == 0 {
		return false
	}

	for _, op := range doc.Operations {
		if op.Operation != ast.Mutation {
			return false
		}
		for _, sel := range op.SelectionSet {
			field, ok := sel.(*ast.Field)
			if !ok {
				return false
			}
			if !publicMutations[field.Name] {
				return false
			}
		}
	}

	return true
}
