package memory

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// WorkingMemoryBucket holds ephemeral, session-scoped context for a single
// agent decision session. Entries have a TTL and are lazily expired on Get.
// This is the "scratch pad" for an agent mid-decision.
//
// Concurrency: all exported methods are safe for concurrent use via
// sync.RWMutex. Multiple goroutines may read simultaneously; writes
// synchronise exclusively.
type WorkingMemoryBucket struct {
	store     *memoryStore
	agentID   string
	sessionID string
	ttl       time.Duration
	mu        sync.RWMutex // guards the owned fields (not the store, which has its own)
	logger    *zap.Logger
}

// NewWorkingMemoryBucket creates a WorkingMemoryBucket scoped to the given
// agent and session. Items written without an explicit importance will default
// to 0.5 (moderate importance). The ttl parameter sets the default expiration
// duration for items written via Set.
func NewWorkingMemoryBucket(agentID, sessionID string, ttl time.Duration, logger *zap.Logger) *WorkingMemoryBucket {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &WorkingMemoryBucket{
		store:     newMemoryStore(),
		agentID:   agentID,
		sessionID: sessionID,
		ttl:       ttl,
		logger:    logger,
	}
}

// Set writes a value into working memory under the given key. importance is
// optional (variadic): if provided, the first value is clamped to [0.0, 1.0];
// if omitted, defaults to 0.5.
//
// If the key already exists, it is overwritten. Old items are not explicitly
// expired beforehand — Set replaces unconditionally.
func (b *WorkingMemoryBucket) Set(key string, value interface{}, importance ...float64) {
	imp := 0.5 // default importance
	if len(importance) > 0 {
		imp = importance[0]
		if imp < 0.0 {
			imp = 0.0
		} else if imp > 1.0 {
			imp = 1.0
		}
	}

	item := NewMemoryItem(b.agentID, b.sessionID, key, value, b.ttl, imp)
	b.store.set(item)

	b.logger.Debug("working memory set",
		zap.String("agent_id", b.agentID),
		zap.String("session_id", b.sessionID),
		zap.String("key", key),
		zap.Float64("importance", imp),
	)
}

// Get retrieves a value by key. It performs lazy TTL expiration: if the item
// exists but its TTL has elapsed, the item is deleted and (nil, false) is
// returned. On a successful hit, LastAccessed is updated atomically.
func (b *WorkingMemoryBucket) Get(key string) (interface{}, bool) {
	item, ok := b.store.getCopy(key)
	if !ok {
		b.logger.Debug("working memory miss",
			zap.String("agent_id", b.agentID),
			zap.String("session_id", b.sessionID),
			zap.String("key", key),
		)
		return nil, false
	}

	if item.IsExpired() {
		b.store.del(key) // lazy eviction
		b.logger.Debug("working memory expired",
			zap.String("agent_id", b.agentID),
			zap.String("session_id", b.sessionID),
			zap.String("key", key),
		)
		return nil, false
	}

	// Update last accessed time under write lock (separate call, safe).
	b.store.touch(key)

	b.logger.Debug("working memory hit",
		zap.String("agent_id", b.agentID),
		zap.String("session_id", b.sessionID),
		zap.String("key", key),
	)
	return item.Value, true
}

// Delete removes the item with the given key. It is a no-op if the key does
// not exist.
func (b *WorkingMemoryBucket) Delete(key string) {
	b.store.del(key)
	b.logger.Debug("working memory deleted",
		zap.String("agent_id", b.agentID),
		zap.String("session_id", b.sessionID),
		zap.String("key", key),
	)
}

// Clear removes all entries from this working memory bucket. After clear,
// the bucket is empty but still usable for new Set calls.
func (b *WorkingMemoryBucket) Clear() {
	b.store.clear()
	b.logger.Debug("working memory cleared",
		zap.String("agent_id", b.agentID),
		zap.String("session_id", b.sessionID),
	)
}

// List returns all MemoryItem entries currently in the bucket, including
// expired ones. Callers can check item.IsExpired() to identify stale entries.
// The returned slice is a value copy; mutating it does not affect the bucket.
func (b *WorkingMemoryBucket) List() []MemoryItem {
	return b.store.all()
}

// Snapshot returns a map of all non-expired key-value pairs currently held
// in the bucket. This is a point-in-time copy intended for trace logging,
// session serialisation, or debugging. Expired entries are excluded.
func (b *WorkingMemoryBucket) Snapshot() map[string]interface{} {
	items := b.store.all()
	result := make(map[string]interface{}, len(items))
	for _, item := range items {
		if !item.IsExpired() {
			result[item.Key] = item.Value
		}
	}
	return result
}

// AgentID returns the agent ID this bucket is scoped to.
func (b *WorkingMemoryBucket) AgentID() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.agentID
}

// SessionID returns the session ID this bucket is scoped to.
func (b *WorkingMemoryBucket) SessionID() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.sessionID
}

// Len returns the number of items currently in the bucket (including expired).
func (b *WorkingMemoryBucket) Len() int {
	return b.store.len()
}
