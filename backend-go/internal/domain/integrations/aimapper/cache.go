package aimapper

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"
)

// ── Deterministic mapping cache ──

// MappingCache is a size-bounded cache keyed by (platform, event type, payload hash).
type MappingCache struct {
	maxSize int
	data    map[string]*CacheEntry
}

// CacheEntry wraps a MapResult with its creation timestamp for LRU-ish eviction.
type CacheEntry struct {
	Result    *MapResult
	CreatedAt time.Time
}

// NewMappingCache creates a cache with the given max capacity (default 10000).
func NewMappingCache(maxSize int) *MappingCache {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &MappingCache{
		maxSize: maxSize,
		data:    make(map[string]*CacheEntry),
	}
}

// Get returns the cached result, or nil on miss.
func (c *MappingCache) Get(key string) *MapResult {
	entry, ok := c.data[key]
	if !ok {
		return nil
	}
	return entry.Result
}

// Set stores a result under the given key. If the cache exceeds maxSize
// the oldest 25 % of entries are evicted.
func (c *MappingCache) Set(key string, result *MapResult) {
	c.data[key] = &CacheEntry{
		Result:    result,
		CreatedAt: time.Now(),
	}
	if len(c.data) > c.maxSize {
		c.evict()
	}
}

// CacheKey builds a deterministic key from the triple (platformCode, eventType, rawPayload).
func CacheKey(platformCode, eventType string, rawPayload []byte) string {
	h := sha256.Sum256(rawPayload)
	return fmt.Sprintf("%s:%s:%x", platformCode, eventType, h)
}

// evict removes the oldest 25% of entries (at least 1) when the cache is over capacity.
func (c *MappingCache) evict() {
	if len(c.data) <= c.maxSize {
		return
	}
	type kv struct {
		key string
		t   time.Time
	}
	entries := make([]kv, 0, len(c.data))
	for k, v := range c.data {
		entries = append(entries, kv{k, v.CreatedAt})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].t.Before(entries[j].t)
	})
	remove := len(c.data) * 25 / 100
	if remove < 1 {
		remove = 1
	}
	for i := 0; i < remove && i < len(entries); i++ {
		delete(c.data, entries[i].key)
	}
}
