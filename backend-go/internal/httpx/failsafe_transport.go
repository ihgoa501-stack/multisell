package httpx

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

var ErrOutboundBlocked = errors.New("outbound network traffic blocked by FailSafeRoundTripper")

type FailSafeRoundTripper struct {
	BaseTransport   http.RoundTripper
	Environment     string
	AllowedHosts    map[string]bool
	AllowedSuffixes []string
	mu              sync.RWMutex
}

func NewFailSafeRoundTripper(base http.RoundTripper, env string) *FailSafeRoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &FailSafeRoundTripper{
		BaseTransport: base,
		Environment:   strings.ToLower(env),
		AllowedHosts: map[string]bool{
			"localhost": true,
			"127.0.0.1": true,
			"::1":       true,
		},
	}
}

func (rt *FailSafeRoundTripper) SetAllowedHosts(hosts []string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.AllowedHosts = map[string]bool{
		"localhost": true,
		"127.0.0.1": true,
		"::1":       true,
	}
	var suffixes []string
	for _, h := range hosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if strings.HasPrefix(h, "*.") {
			suffixes = append(suffixes, h[1:])
		} else {
			rt.AllowedHosts[h] = true
		}
	}
	rt.AllowedSuffixes = suffixes
}

func (rt *FailSafeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.Environment == "production" {
		return rt.BaseTransport.RoundTrip(req)
	}

	host := strings.ToLower(req.URL.Hostname())

	rt.mu.RLock()
	allowed := rt.AllowedHosts[host]
	suffixes := rt.AllowedSuffixes
	rt.mu.RUnlock()

	if !allowed {
		for _, suffix := range suffixes {
			if strings.HasSuffix(host, suffix) {
				allowed = true
				break
			}
		}
	}

	if !allowed {
		if ip := net.ParseIP(host); ip != nil {
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
				allowed = true
			}
		}
	}

	if !allowed {
		return nil, fmt.Errorf("%w: blocked outbound request to %q (env=%s)",
			ErrOutboundBlocked, host, rt.Environment)
	}

	return rt.BaseTransport.RoundTrip(req)
}
