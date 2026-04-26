// Package email provides a transactional email sender.
//
// MailgunSender talks to Mailgun's HTTP API directly to keep the dependency
// surface small. LogSender writes messages to the standard logger and is the
// default in development so local work doesn't need Mailgun credentials.
package email

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Sender delivers a transactional email.
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// LogSender writes the message to slog instead of sending it. Used in
// development and tests.
type LogSender struct{}

func (LogSender) Send(_ context.Context, to, subject, body string) error {
	slog.Info("email (dev)", "to", to, "subject", subject, "body", body)
	return nil
}

// MailgunSender sends email via the Mailgun HTTP API.
//
// API: POST https://api.mailgun.net/v3/{domain}/messages
// Auth: HTTP basic, username "api", password = API key.
type MailgunSender struct {
	APIKey string
	Domain string
	From   string
	HTTP   *http.Client
}

// NewMailgun returns a MailgunSender with a 10s default HTTP timeout.
func NewMailgun(apiKey, domain, from string) *MailgunSender {
	return &MailgunSender{
		APIKey: apiKey,
		Domain: domain,
		From:   from,
		HTTP:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *MailgunSender) Send(ctx context.Context, to, subject, body string) error {
	form := url.Values{}
	form.Set("from", s.From)
	form.Set("to", to)
	form.Set("subject", subject)
	form.Set("text", body)

	endpoint := fmt.Sprintf("https://api.mailgun.net/v3/%s/messages", s.Domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("mailgun: build request: %w", err)
	}
	req.SetBasicAuth("api", s.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("mailgun: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("mailgun: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
}
