// Package memory provides the Memory System v1 for AIOS — short-term working
// memory and long-term memory for agent decision contexts.
//
// Memory is the subjective knowledge layer for agents, distinct from traces
// (objective execution logs). WorkingMemoryBucket holds per-session ephemeral
// context with TTL-based lazy expiration, while LongTermMemory provides
// cross-session persistent storage with importance-based eviction.
//
// Usage:
//
//	bucket := memory.NewWorkingMemoryBucket("A5", "sess_abc", 15*time.Minute, logger)
//	bucket.Set("last_check", result)
//	data, ok := bucket.Get("last_check")
//
//	ltm := memory.NewLongTermMemory("A5", logger)
//	ltm.Remember("supplier_performance", "Supplier B was 3 days late last time", 0.8)
//	results, _ := ltm.Search("supplier", 5)
package memory

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryItem is a single memory entry stored in either working or long-term memory.
// Each item is identified by its Key within a scope (agent+session for working,
// agent for long-term), carries an opaque Value, and is annotated with metadata
// for lifecycle and retrieval decisions.
type MemoryItem struct {
	// ID is a globally unique identifier for this memory entry.
	ID string `json:"id"`

	// AgentID identifies the agent that owns this memory.
	AgentID string `json:"agent_id"`

	// SessionID identifies the decision session (working memory only).
	SessionID string `json:"session_id,omitempty"`

	// Key is the logical name for this memory entry (e.g. "last_stock_check").
	Key string `json:"key"`

	// Value is the opaque payload. Can hold any Go value.
	Value interface{} `json:"value"`

	// TTL is the expiration time. If zero or in the past, the item is considered
	// expired. Used for working memory auto-expiration.
	TTL time.Time `json:"ttl"`

	// Importance is a 0.0–1.0 score used for eviction decisions in long-term
	// memory. Higher values are less likely to be evicted.
	Importance float64 `json:"importance"`

	// CreatedAt is when this memory entry was first stored.
	CreatedAt time.Time `json:"created_at"`

	// LastAccessed is the last time this item was retrieved via Get/Recall.
	// Updated atomically through touch() under write lock.
	LastAccessed time.Time `json:"last_accessed"`
}

// NewMemoryItem creates a new MemoryItem with the given parameters. It generates
// a UUID for ID and sets CreatedAt and LastAccessed to the current time. The
// TTL is computed as now + ttlDuration; pass 0 for no expiration (long-term).
func NewMemoryItem(agentID, sessionID, key string, value interface{}, ttlDuration time.Duration, importance float64) *MemoryItem {
	now := time.Now()
	var ttl time.Time
	if ttlDuration > 0 {
		ttl = now.Add(ttlDuration)
	}
	return &MemoryItem{
		ID:           uuid.New().String(),
		AgentID:      agentID,
		SessionID:    sessionID,
		Key:          key,
		Value:        value,
		TTL:          ttl,
		Importance:   importance,
		CreatedAt:    now,
		LastAccessed: now,
	}
}

// IsExpired returns true if the item has a non-zero TTL that is in the past.
// Items with a zero TTL (long-term memory) never expire via this check.
func (m *MemoryItem) IsExpired() bool {
	if m.TTL.IsZero() {
		return false
	}
	return time.Now().After(m.TTL)
}

// memoryStore is the shared internal storage primitive used by both
// WorkingMemoryBucket and LongTermMemory. It provides thread-safe
// CRUD operations on the underlying map.
//
// To avoid data races on the shared MemoryItem pointers, Get methods
// return a value copy rather than a pointer. Mutating operations
// (Set, Touch) are serialised under the write lock.
type memoryStore struct {
	mu    sync.RWMutex
	items map[string]*MemoryItem
}

// newMemoryStore creates an initialised memoryStore.
func newMemoryStore() *memoryStore {
	return &memoryStore{
		items: make(map[string]*MemoryItem),
	}
}

// set stores an item under its key, overwriting any existing entry.
func (s *memoryStore) set(item *MemoryItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.Key] = item
}

// getCopy returns a value copy of the item for the given key under read lock.
// Returns the zero value and false if the key does not exist.
// The returned copy shares the same Value interface but cannot race on
// struct-level writes with concurrent touch() calls.
func (s *memoryStore) getCopy(key string) (MemoryItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[key]
	if !ok {
		return MemoryItem{}, false
	}
	return *item, true
}

// touch acquires a write lock and updates LastAccessed on the item with the
// given key. It is a no-op if the key does not exist. This is called after
// getCopy to atomically update the access timestamp.
func (s *memoryStore) touch(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.items[key]; ok {
		item.LastAccessed = time.Now()
	}
}

// del removes the item with the given key. It is a no-op if the key does not exist.
func (s *memoryStore) del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
}

// clear removes all items from the store.
func (s *memoryStore) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[string]*MemoryItem)
}

// len returns the number of items in the store.
func (s *memoryStore) len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// all returns a slice of value copies of all items under read lock.
// Mutating the returned slice or its elements does not affect the store.
func (s *memoryStore) all() []MemoryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]MemoryItem, 0, len(s.items))
	for _, item := range s.items {
		result = append(result, *item)
	}
	return result
}

// allMap returns a shallow copy of the underlying map under read lock.
// The Value pointers are still shared; callers must not mutate returned items.
func (s *memoryStore) allMap() map[string]*MemoryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*MemoryItem, len(s.items))
	for k, v := range s.items {
		result[k] = v
	}
	return result
}
