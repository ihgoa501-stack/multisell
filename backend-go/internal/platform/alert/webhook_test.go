package alert

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookSender_Send(t *testing.T) {
	var received struct {
		Title   string `json:"title"`
		Message string `json:"message"`
		Level   string `json:"level"`
		Source  string `json:"source"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := NewWebhookSender([]string{srv.URL})
	err := sender.Send(context.Background(), Payload{
		Title:   "Test Alert",
		Message: "Something went wrong",
		Level:   "error",
		Source:  "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Title != "Test Alert" {
		t.Errorf("expected title 'Test Alert', got %q", received.Title)
	}
	if received.Level != "error" {
		t.Errorf("expected level 'error', got %q", received.Level)
	}
}

func TestWebhookSender_NoURLs(t *testing.T) {
	sender := NewWebhookSender(nil)
	err := sender.Send(context.Background(), Payload{
		Title: "no-op",
	})
	if err != nil {
		t.Fatalf("expected nil for no URLs, got %v", err)
	}
}

func TestWebhookSender_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	sender := NewWebhookSender([]string{srv.URL})
	err := sender.Send(context.Background(), Payload{
		Title:   "Error test",
		Message: "server error expected",
		Level:   "error",
	})
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestSendHealthAlert(t *testing.T) {
	var got Payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := NewWebhookSender([]string{srv.URL})
	err := sender.SendHealthAlert(context.Background(), "backend", "service unreachable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Level != "critical" {
		t.Errorf("expected level critical for health alert, got %q", got.Level)
	}
}
