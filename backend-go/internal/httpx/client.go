package httpx

import (
	"net/http"
	"time"
)

// NewClient creates a standard HTTP Client pre-loaded with outbound safety barriers.
func NewClient(env string, timeout time.Duration) *http.Client {
	transport := NewFailSafeRoundTripper(http.DefaultTransport, env)
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}
