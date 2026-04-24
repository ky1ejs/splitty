package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testOrigin = "http://localhost:5173"

func newHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestPreflightReturns204(t *testing.T) {
	handler := Middleware(testOrigin)(newHandler())

	req := httptest.NewRequest(http.MethodOptions, "/query", nil)
	req.Header.Set("Origin", testOrigin)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Fatalf("expected Allow-Origin %q, got %q", testOrigin, got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("expected Allow-Methods, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Fatalf("expected Allow-Headers, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected Allow-Credentials true, got %q", got)
	}
}

func TestActualRequestSetsCORSHeaders(t *testing.T) {
	handler := Middleware(testOrigin)(newHandler())

	req := httptest.NewRequest(http.MethodPost, "/query", nil)
	req.Header.Set("Origin", testOrigin)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Fatalf("expected Allow-Origin %q, got %q", testOrigin, got)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body 'ok', got %q", rec.Body.String())
	}
}

func TestNoOriginHeader(t *testing.T) {
	handler := Middleware(testOrigin)(newHandler())

	req := httptest.NewRequest(http.MethodPost, "/query", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin header, got %q", got)
	}
}

func TestWrongOrigin(t *testing.T) {
	handler := Middleware(testOrigin)(newHandler())

	req := httptest.NewRequest(http.MethodPost, "/query", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin header, got %q", got)
	}
}

func TestEmptyAllowedOriginSkipsCORS(t *testing.T) {
	handler := Middleware("")(newHandler())

	req := httptest.NewRequest(http.MethodPost, "/query", nil)
	req.Header.Set("Origin", testOrigin)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin header, got %q", got)
	}
}
