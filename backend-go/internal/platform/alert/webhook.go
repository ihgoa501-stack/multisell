// Package alert provides lightweight notification channels for system alerts.
// Currently supports webhook-based HTTP POST notifications.
//
// ponytail: single webhook sender, one URL list. Add retry/backoff if delivery
// reliability becomes a concern.
package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookSender sends alert payloads to configured HTTP endpoints.
type WebhookSender struct {
	client *http.Client
	urls   []string
}

// NewWebhookSender creates a sender that POSTs alerts to the given URLs.
func NewWebhookSender(urls []string) *WebhookSender {
	return &WebhookSender{
		client: &http.Client{Timeout: 10 * time.Second},
		urls:   urls,
	}
}

// Payload is the JSON structure sent to each webhook URL.
type Payload struct {
	Title     string `json:"title"`
	Message   string `json:"message"`
	Level     string `json:"level"` // "info" | "warning" | "error" | "critical"
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
}

// Send delivers an alert to every configured webhook URL.
// Errors from individual URLs are logged but not returned — the function
// succeeds if at least one URL was reached, or returns the last error.
func (s *WebhookSender) Send(ctx context.Context, p Payload) error {
	if len(s.urls) == 0 {
		return nil
	}
	if p.Timestamp == "" {
		p.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("alert: marshal payload: %w", err)
	}
	var lastErr error
	for _, url := range s.urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			lastErr = fmt.Errorf("alert: create request for %q: %w", url, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("alert: send to %q: %w", url, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("alert: %q returned status %d", url, resp.StatusCode)
		}
	}
	return lastErr
}

// SendHealthAlert is a convenience wrapper that sends a health-check-failure alert.
func (s *WebhookSender) SendHealthAlert(ctx context.Context, service, detail string) error {
	return s.Send(ctx, Payload{
		Title:   fmt.Sprintf("[%s] Health check failed", service),
		Message: detail,
		Level:   "critical",
		Source:  service,
	})
}
