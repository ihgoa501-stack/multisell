package memory

import (
	"sort"
	"strings"

	"go.uber.org/zap"
)

// LongTermMemory provides persistent, cross-session knowledge storage for a
// single agent. Unlike WorkingMemoryBucket, items have no TTL — they survive
// until explicitly Forgotten or Evicted by importance-based ranking.
//
// Concurrency: all exported methods are safe for concurrent use. Reads use a
// read lock; writes and eviction use a write lock.
type LongTermMemory struct {
	store   *memoryStore
	agentID string

	// embedFn is an optional embedding function for semantic search (v2 feature).
	// When non-nil, items are embedded on Remember for vector-based retrieval.
	// In v1, this is unused — search is keyword-based.
	embedFn func(text string) ([]float32, error)
 // guards agentID (the store has its own lock)
	logger *zap.Logger
}

// NewLongTermMemory creates a LongTermMemory for the given agent. The embedFn
// may be nil; in v1 it is reserved for future semantic search. If logger is
// nil, a no-op logger is used.
func NewLongTermMemory(agentID string, embedFn func(string) ([]float32, error), logger *zap.Logger) *LongTermMemory {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &LongTermMemory{
		store:   newMemoryStore(),
		agentID: agentID,
		embedFn: embedFn,
		logger:  logger,
	}
}

// Remember stores a key-value pair in long-term memory with the given
// importance (0.0–1.0). Higher importance makes the entry less likely to be
// evicted. If the key already exists, it is overwritten.
func (m *LongTermMemory) Remember(key string, value interface{}, importance float64) {
	// Clamp importance to [0.0, 1.0]
	if importance < 0.0 {
		importance = 0.0
	} else if importance > 1.0 {
		importance = 1.0
	}

	// Long-term memory uses zero TTL (no expiration).
	item := NewMemoryItem(m.agentID, "", key, value, 0, importance)
	m.store.set(item)

	m.logger.Debug("long-term memory remember",
		zap.String("agent_id", m.agentID),
		zap.String("key", key),
		zap.Float64("importance", importance),
	)
}

// Recall retrieves a value by exact key match. Returns (nil, false) if the
// key is not found. Unlike working memory, long-term entries never expire,
// so no TTL check is performed. LastAccessed is updated atomically.
func (m *LongTermMemory) Recall(key string) (interface{}, bool) {
	item, ok := m.store.getCopy(key)
	if !ok {
		m.logger.Debug("long-term memory recall miss",
			zap.String("agent_id", m.agentID),
			zap.String("key", key),
		)
		return nil, false
	}

	// Update last accessed time under write lock.
	m.store.touch(key)

	m.logger.Debug("long-term memory recall hit",
		zap.String("agent_id", m.agentID),
		zap.String("key", key),
	)
	return item.Value, true
}

// Search performs a simple keyword-based search over all stored long-term
// memory entries. An entry matches if its Key or string-formatted Value
// contains the query substring (case-sensitive). Results are limited to
// at most limit entries (0 or negative means no limit).
//
// In v1, search uses strings.Contains. Future versions may add TF-IDF,
// embedding-based semantic search, or full-text indexing.
func (m *LongTermMemory) Search(query string, limit int) ([]MemoryItem, error) {
	items := m.store.all()

	if query == "" {
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}
		return items, nil
	}

	matched := make([]MemoryItem, 0)
	for _, item := range items {
		if strings.Contains(item.Key, query) {
			matched = append(matched, item)
			continue
		}
		// Also match against the string representation of value
		if str, ok := item.Value.(string); ok && strings.Contains(str, query) {
			matched = append(matched, item)
		}
	}

	// Apply limit
	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}

	return matched, nil
}

// Forget removes a single entry by key from long-term memory. It is a no-op
// if the key does not exist.
func (m *LongTermMemory) Forget(key string) {
	m.store.del(key)
	m.logger.Debug("long-term memory forget",
		zap.String("agent_id", m.agentID),
		zap.String("key", key),
	)
}

// Evict removes the lowest-importance entries until count items have been
// removed or the store is empty. Entries are sorted by Importance ascending,
// and the first count entries are deleted. This is useful for enforcing a
// capacity limit on long-term memory.
//
// If count is greater than or equal to the store length, all entries are
// removed. If count is <= 0, this is a no-op.
func (m *LongTermMemory) Evict(count int) {
	if count <= 0 {
		return
	}

	items := m.store.all()
	if len(items) == 0 {
		return
	}

	// Sort ascending by importance (lowest importance first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Importance < items[j].Importance
	})

	// Evict the first 'count' items (or all if count >= len)
	toRemove := count
	if toRemove > len(items) {
		toRemove = len(items)
	}

	for i := 0; i < toRemove; i++ {
		m.store.del(items[i].Key)
	}

	m.logger.Info("long-term memory evicted",
		zap.String("agent_id", m.agentID),
		zap.Int("evicted_count", toRemove),
		zap.Int("remaining", m.store.len()),
	)
}

// Len returns the number of entries currently stored.
func (m *LongTermMemory) Len() int {
	return m.store.len()
}
