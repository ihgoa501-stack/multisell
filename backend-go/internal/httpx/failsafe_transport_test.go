package httpx

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

type mockRoundTripper struct {
	called bool
	req    *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.called = true
	m.req = req
	return &http.Response{
		StatusCode: http.StatusOK,
	}, nil
}

func TestFailSafeRoundTripper_Blocked(t *testing.T) {
	mock := &mockRoundTripper{}
	transport := NewFailSafeRoundTripper(mock, "development")
	client := &http.Client{Transport: transport}

	_, err := client.Get("https://api.ozon.ru/v1/products")
	if err == nil {
		t.Fatal("Expected outbound request to api.ozon.ru to be blocked in development mode, but it succeeded")
	}
	if !errors.Is(err, ErrOutboundBlocked) && !strings.Contains(err.Error(), "blocked outbound request") {
		t.Fatalf("Expected blocked outbound request error, got: %v", err)
	}
	if mock.called {
		t.Fatal("Expected mock transport to not be called for blocked request")
	}
}

func TestFailSafeRoundTripper_AllowedPrivate(t *testing.T) {
	mock := &mockRoundTripper{}
	transport := NewFailSafeRoundTripper(mock, "development")
	client := &http.Client{Transport: transport}

	_, err := client.Get("http://127.0.0.1:8080/api/health")
	if err != nil {
		t.Fatalf("Expected loopback to bypass gate, but got error: %v", err)
	}
	if !mock.called {
		t.Fatal("Expected mock transport to be called for loopback address")
	}
}

func TestFailSafeRoundTripper_AllowedHosts(t *testing.T) {
	mock := &mockRoundTripper{}
	transport := NewFailSafeRoundTripper(mock, "development")
	transport.SetAllowedHosts([]string{"api.ozon.ru"})
	client := &http.Client{Transport: transport}

	_, err := client.Get("https://api.ozon.ru/v1/products")
	if err != nil {
		t.Fatalf("Expected allowed host to bypass gate, but got error: %v", err)
	}
	if !mock.called {
		t.Fatal("Expected mock transport to be called for allowed host")
	}
}

func TestFailSafeRoundTripper_AllowedSuffixes(t *testing.T) {
	mock := &mockRoundTripper{}
	transport := NewFailSafeRoundTripper(mock, "development")
	transport.SetAllowedHosts([]string{"*.ozon.ru"})
	client := &http.Client{Transport: transport}

	_, err := client.Get("https://api.ozon.ru/v1/products")
	if err != nil {
		t.Fatalf("Expected allowed suffix host to bypass gate, but got error: %v", err)
	}
	if !mock.called {
		t.Fatal("Expected mock transport to be called for allowed suffix host")
	}
}

func TestFailSafeRoundTripper_ProductionBypass(t *testing.T) {
	mock := &mockRoundTripper{}
	transport := NewFailSafeRoundTripper(mock, "production")
	client := &http.Client{Transport: transport}

	_, err := client.Get("https://api.ozon.ru/v1/products")
	if err != nil {
		t.Fatalf("Expected production environment to bypass gate, but got error: %v", err)
	}
	if !mock.called {
		t.Fatal("Expected mock transport to be called in production environment")
	}
}
