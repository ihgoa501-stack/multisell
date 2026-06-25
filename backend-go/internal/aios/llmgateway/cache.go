package llmgateway

import (
	"context"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Cache interface
// ---------------------------------------------------------------------------

// Cache defines the response caching contract for the LLM Gateway.
// Implementations must be thread-safe.
type Cache interface {
	// Get returns a cached response. Returns nil and false if not found or expired.
	Get(ctx context.Context, key string) (*Response, bool)

	// Set stores a response with the given TTL.
	Set(ctx context.Context, key string, resp *Response, ttl time.Duration)

	// Invalidate removes all entries whose key contains the pattern string.
	// An empty pattern clears the entire cache.
	Invalidate(pattern string)

	// DefaultTTL returns the default TTL used when no explicit TTL is provided.
	DefaultTTL() time.Duration
}

// ---------------------------------------------------------------------------
// MemoryCache
// ---------------------------------------------------------------------------

// cacheEntry holds a cached response with its expiry metadata.
type cacheEntry struct {
	Response  *Response
	TTL       time.Duration
	CreatedAt time.Time
}

// MemoryCache is a thread-safe, in-memory implementation of the Cache interface.
// Entries are evicted lazily on read — expired entries are not returned and
// are cleaned up on subsequent Get calls.
type MemoryCache struct {
	mu         sync.RWMutex
	entries    map[string]cacheEntry
	defaultTTL time.Duration
}

// NewMemoryCache creates a MemoryCache with the given default TTL.
// Default TTL is used when Set is called without a specific TTL (zero value).
func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}
	return &MemoryCache{
		entries:    make(map[string]cacheEntry),
		defaultTTL: defaultTTL,
	}
}

// Get returns a cached response if the key exists and the entry has not
// expired. Returns nil and false otherwise. Expired entries are pruned
// on access to prevent memory leaks.
func (m *MemoryCache) Get(_ context.Context, key string) (*Response, bool) {
	m.mu.RLock()
	entry, ok := m.entries[key]
	m.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Check TTL expiry.
	if time.Since(entry.CreatedAt) >= entry.TTL {
		// Lazily evict.
		m.mu.Lock()
		delete(m.entries, key)
		m.mu.Unlock()
		return nil, false
	}

	return entry.Response, true
}

// Set stores a response under the given key. If ttl is zero, the cache's
// default TTL is used.
func (m *MemoryCache) Set(_ context.Context, key string, resp *Response, ttl time.Duration) {
	if ttl <= 0 {
		ttl = m.defaultTTL
	}

	m.mu.Lock()
	m.entries[key] = cacheEntry{
		Response:  resp,
		TTL:       ttl,
		CreatedAt: time.Now(),
	}
	m.mu.Unlock()
}

// Invalidate removes all entries whose key contains the pattern string.
// An empty pattern removes all entries.
func (m *MemoryCache) Invalidate(pattern string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pattern == "" {
		m.entries = make(map[string]cacheEntry)
		return
	}

	for key := range m.entries {
		if contains(key, pattern) {
			delete(m.entries, key)
		}
	}
}

// DefaultTTL returns the configured default TTL for this cache.
func (m *MemoryCache) DefaultTTL() time.Duration {
	return m.defaultTTL
}

// Len returns the number of entries currently in the cache (for testing).
func (m *MemoryCache) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// contains reports whether s contains substr.
func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	// Simple substring match; not rune-aware but sufficient for cache key patterns.
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
