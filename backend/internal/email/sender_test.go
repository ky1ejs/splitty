package email

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMailgunSender_Send_Success(t *testing.T) {
	var gotPath, gotAuth, gotForm string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotForm = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"<msg@example>","message":"Queued"}`))
	}))
	defer srv.Close()

	s := NewMailgun("KEY", "mg.example.com", "Splitty <noreply@mg.example.com>")
	s.HTTP = srv.Client()
	// Override endpoint by redirecting via a custom transport
	s.HTTP.Transport = &rewriteTransport{base: srv.Client().Transport, target: srv.URL}

	if err := s.Send(context.Background(), "alice@example.com", "Code", "123456"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.HasSuffix(gotPath, "/v3/mg.example.com/messages") {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Errorf("expected basic auth, got %q", gotAuth)
	}
	if !strings.Contains(gotForm, "to=alice%40example.com") {
		t.Errorf("missing to in form: %s", gotForm)
	}
	if !strings.Contains(gotForm, "text=123456") {
		t.Errorf("missing body in form: %s", gotForm)
	}
}

func TestMailgunSender_Send_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`forbidden`))
	}))
	defer srv.Close()

	s := NewMailgun("KEY", "mg.example.com", "noreply@mg.example.com")
	s.HTTP = srv.Client()
	s.HTTP.Transport = &rewriteTransport{base: srv.Client().Transport, target: srv.URL}

	err := s.Send(context.Background(), "alice@example.com", "Code", "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected status in error, got %v", err)
	}
}

// rewriteTransport rewrites all requests to point at target host (the test server),
// preserving the path and headers. This lets us exercise the real URL building.
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tgt, err := req.URL.Parse(rt.target)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = tgt.Scheme
	req.URL.Host = tgt.Host
	if rt.base == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return rt.base.RoundTrip(req)
}
