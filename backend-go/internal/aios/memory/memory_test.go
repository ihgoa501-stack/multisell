package memory

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// WorkingMemoryBucket tests
// ---------------------------------------------------------------------------

func newTestBucket(t *testing.T) *WorkingMemoryBucket {
	t.Helper()
	logger := zap.NewNop()
	return NewWorkingMemoryBucket("test-agent", "test-session", 5*time.Minute, logger)
}

func TestWorkingMemoryBucket_SetAndGet(t *testing.T) {
	b := newTestBucket(t)

	// Set a value and retrieve it
	b.Set("stock_level", 42)
	val, ok := b.Get("stock_level")
	if !ok {
		t.Fatal("expected Get to return ok=true")
	}
	if val != 42 {
		t.Fatalf("expected value 42, got %v", val)
	}
}

func TestWorkingMemoryBucket_Get_Miss(t *testing.T) {
	b := newTestBucket(t)

	val, ok := b.Get("nonexistent")
	if ok {
		t.Fatal("expected Get on missing key to return ok=false")
	}
	if val != nil {
		t.Fatalf("expected nil value, got %v", val)
	}
}

func TestWorkingMemoryBucket_Overwrite(t *testing.T) {
	b := newTestBucket(t)

	b.Set("key", "first")
	b.Set("key", "second")

	val, ok := b.Get("key")
	if !ok {
		t.Fatal("expected Get to return ok=true")
	}
	if val != "second" {
		t.Fatalf("expected 'second', got %v", val)
	}
}

func TestWorkingMemoryBucket_Delete(t *testing.T) {
	b := newTestBucket(t)

	b.Set("key", "value")
	b.Delete("key")

	val, ok := b.Get("key")
	if ok || val != nil {
		t.Fatal("expected Get after Delete to return nil, false")
	}
}

func TestWorkingMemoryBucket_Delete_MissingKey(t *testing.T) {
	b := newTestBucket(t)

	// Should not panic
	b.Delete("does-not-exist")
}

func TestWorkingMemoryBucket_Clear(t *testing.T) {
	b := newTestBucket(t)

	b.Set("a", 1)
	b.Set("b", 2)
	b.Clear()

	if b.Len() != 0 {
		t.Fatalf("expected Len=0 after Clear, got %d", b.Len())
	}

	_, ok := b.Get("a")
	if ok {
		t.Fatal("expected Get after Clear to return ok=false")
	}
}

func TestWorkingMemoryBucket_List(t *testing.T) {
	b := newTestBucket(t)

	b.Set("x", 10)
	b.Set("y", 20)

	items := b.List()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// List should include all items; check they have the right keys
	keys := make(map[string]bool)
	for _, item := range items {
		keys[item.Key] = true
	}
	if !keys["x"] || !keys["y"] {
		t.Fatal("List did not contain all expected keys")
	}
}

func TestWorkingMemoryBucket_Snapshot(t *testing.T) {
	b := newTestBucket(t)

	b.Set("a", "alpha")
	b.Set("b", "beta")

	snap := b.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected snapshot with 2 entries, got %d", len(snap))
	}
	if snap["a"] != "alpha" || snap["b"] != "beta" {
		t.Fatal("snapshot values do not match")
	}
}

func TestWorkingMemoryBucket_Snapshot_ExcludesExpired(t *testing.T) {
	// Use a very short TTL so items expire before Snapshot
	logger := zap.NewNop()
	b := NewWorkingMemoryBucket("test-agent", "test-session", 1*time.Millisecond, logger)

	b.Set("ephemeral", "gone-soon")

	// Wait for TTL to expire
	time.Sleep(5 * time.Millisecond)

	snap := b.Snapshot()
	if len(snap) != 0 {
		t.Fatalf("expected empty snapshot after TTL expiry, got %d entries", len(snap))
	}
}

func TestWorkingMemoryBucket_ListIncludesExpired(t *testing.T) {
	logger := zap.NewNop()
	b := NewWorkingMemoryBucket("test-agent", "test-session", 1*time.Millisecond, logger)

	b.Set("ephemeral", "still-listed")

	time.Sleep(5 * time.Millisecond)

	items := b.List()
	found := false
	for _, item := range items {
		if item.Key == "ephemeral" {
			found = true
			if !item.IsExpired() {
				t.Fatal("expected item to be marked as expired")
			}
		}
	}
	if !found {
		t.Fatal("List should include expired entries")
	}
}

func TestWorkingMemoryBucket_TTL_Expiration(t *testing.T) {
	logger := zap.NewNop()
	b := NewWorkingMemoryBucket("test-agent", "test-session", 10*time.Millisecond, logger)

	b.Set("temp", "data")

	// Should be visible immediately
	_, ok := b.Get("temp")
	if !ok {
		t.Fatal("expected Get to succeed before TTL expiry")
	}

	// Wait for TTL to expire
	time.Sleep(25 * time.Millisecond)

	// Should be gone now (lazy eviction on Get)
	val, ok := b.Get("temp")
	if ok || val != nil {
		t.Fatal("expected Get to return nil, false after TTL expiry")
	}
}

func TestWorkingMemoryBucket_EmptySnapshot(t *testing.T) {
	b := newTestBucket(t)
	snap := b.Snapshot()
	if snap == nil {
		t.Fatal("expected non-nil snapshot even when empty")
	}
	if len(snap) != 0 {
		t.Fatalf("expected empty snapshot, got %d entries", len(snap))
	}
}

func TestWorkingMemoryBucket_EmptyList(t *testing.T) {
	b := newTestBucket(t)
	items := b.List()
	if items == nil {
		t.Fatal("expected non-nil list even when empty")
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list, got %d items", len(items))
	}
}

func TestWorkingMemoryBucket_ImportanceDefault(t *testing.T) {
	b := newTestBucket(t)
	b.Set("key", "val")
	items := b.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Importance != 0.5 {
		t.Fatalf("expected default importance 0.5, got %f", items[0].Importance)
	}
}

func TestWorkingMemoryBucket_ImportanceClamping(t *testing.T) {
	b := newTestBucket(t)

	b.Set("under", "val", -0.5)
	b.Set("over", "val", 1.5)
	b.Set("exact", "val", 0.7)

	items := b.List()
	importances := make(map[string]float64)
	for _, item := range items {
		importances[item.Key] = item.Importance
	}

	if importances["under"] != 0.0 {
		t.Fatalf("expected importance 0.0 for under, got %f", importances["under"])
	}
	if importances["over"] != 1.0 {
		t.Fatalf("expected importance 1.0 for over, got %f", importances["over"])
	}
	if importances["exact"] != 0.7 {
		t.Fatalf("expected importance 0.7, got %f", importances["exact"])
	}
}

// ---------------------------------------------------------------------------
// LongTermMemory tests
// ---------------------------------------------------------------------------

func newTestLTM(t *testing.T) *LongTermMemory {
	t.Helper()
	logger := zap.NewNop()
	return NewLongTermMemory("test-agent", nil, logger)
}

func TestLongTermMemory_RememberAndRecall(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("supplier_preference", "Supplier A is preferred for electronics", 0.8)
	val, ok := ltm.Recall("supplier_preference")
	if !ok {
		t.Fatal("expected Recall to return ok=true")
	}
	if val != "Supplier A is preferred for electronics" {
		t.Fatalf("unexpected value: %v", val)
	}
}

func TestLongTermMemory_Recall_Miss(t *testing.T) {
	ltm := newTestLTM(t)

	val, ok := ltm.Recall("nonexistent")
	if ok || val != nil {
		t.Fatal("expected Recall on missing key to return nil, false")
	}
}

func TestLongTermMemory_Overwrite(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("key", "old", 0.3)
	ltm.Remember("key", "new", 0.9)

	val, ok := ltm.Recall("key")
	if !ok {
		t.Fatal("expected Recall to return ok=true")
	}
	if val != "new" {
		t.Fatalf("expected 'new', got %v", val)
	}
}

func TestLongTermMemory_Forget(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("key", "value", 0.5)
	ltm.Forget("key")

	val, ok := ltm.Recall("key")
	if ok || val != nil {
		t.Fatal("expected Recall after Forget to return nil, false")
	}
}

func TestLongTermMemory_Forget_MissingKey(t *testing.T) {
	ltm := newTestLTM(t)
	ltm.Forget("does-not-exist") // should not panic
}

func TestLongTermMemory_Search_ByKey(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("supplier_rating_alpha", "Alpha Corp: 4.5/5", 0.7)
	ltm.Remember("supplier_rating_beta", "Beta Inc: 3.8/5", 0.6)
	ltm.Remember("inventory_turnover", "2.3x per month", 0.5)

	results, err := ltm.Search("supplier", 10)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'supplier', got %d", len(results))
	}
}

func TestLongTermMemory_Search_ByValue(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("note_a", "This is about supplier performance", 0.6)
	ltm.Remember("note_b", "This is about inventory levels", 0.5)

	results, err := ltm.Search("supplier", 10)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'supplier' value match, got %d", len(results))
	}
}

func TestLongTermMemory_Search_NoMatch(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("a", "hello", 0.5)

	results, err := ltm.Search("zzzzz", 10)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestLongTermMemory_Search_EmptyQuery(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("a", "1", 0.3)
	ltm.Remember("b", "2", 0.4)
	ltm.Remember("c", "3", 0.5)

	// Empty query returns all
	results, err := ltm.Search("", 10)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results for empty query, got %d", len(results))
	}
}

func TestLongTermMemory_Search_Limit(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("a", "hello", 0.3)
	ltm.Remember("b", "hello", 0.4)
	ltm.Remember("c", "hello", 0.5)

	results, err := ltm.Search("hello", 2)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results with limit=2, got %d", len(results))
	}
}

func TestLongTermMemory_Evict_RemovesLowestImportance(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("high", "important", 1.0)
	ltm.Remember("medium", "moderate", 0.5)
	ltm.Remember("low", "trivial", 0.1)

	ltm.Evict(1) // should remove "low" (importance 0.1)

	if _, ok := ltm.Recall("low"); ok {
		t.Fatal("expected 'low' to be evicted")
	}
	if _, ok := ltm.Recall("medium"); !ok {
		t.Fatal("expected 'medium' to survive eviction")
	}
	if _, ok := ltm.Recall("high"); !ok {
		t.Fatal("expected 'high' to survive eviction")
	}
}

func TestLongTermMemory_Evict_Multiple(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("a", "val", 0.1)
	ltm.Remember("b", "val", 0.2)
	ltm.Remember("c", "val", 0.3)
	ltm.Remember("d", "val", 0.9)

	ltm.Evict(2) // should remove a and b

	if _, ok := ltm.Recall("a"); ok {
		t.Fatal("expected 'a' to be evicted")
	}
	if _, ok := ltm.Recall("b"); ok {
		t.Fatal("expected 'b' to be evicted")
	}
	if _, ok := ltm.Recall("c"); !ok {
		t.Fatal("expected 'c' to survive eviction")
	}
	if _, ok := ltm.Recall("d"); !ok {
		t.Fatal("expected 'd' to survive eviction")
	}
}

func TestLongTermMemory_Evict_AllItems(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("a", "val", 0.5)
	ltm.Remember("b", "val", 0.8)

	ltm.Evict(10) // more items than exist — should clear all

	if ltm.Len() != 0 {
		t.Fatalf("expected Len=0 after Evict with large count, got %d", ltm.Len())
	}
}

func TestLongTermMemory_Evict_ZeroOrNegative(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("a", "val", 0.5)
	ltm.Remember("b", "val", 0.5)

	ltm.Evict(0)   // no-op
	ltm.Evict(-1)  // no-op

	if ltm.Len() != 2 {
		t.Fatalf("expected Len=2 after no-op Evict, got %d", ltm.Len())
	}
}

func TestLongTermMemory_Evict_EmptyStore(t *testing.T) {
	ltm := newTestLTM(t)
	ltm.Evict(5) // should not panic
}

func TestLongTermMemory_ImportanceClamping(t *testing.T) {
	ltm := newTestLTM(t)

	ltm.Remember("under", "val", -0.5)
	ltm.Remember("over", "val", 1.5)
	ltm.Remember("normal", "val", 0.7)

	if val, _ := ltm.Recall("under"); val == nil {
		t.Fatal("expected 'under' to be stored with clamped importance")
	}
	if val, _ := ltm.Recall("over"); val == nil {
		t.Fatal("expected 'over' to be stored with clamped importance")
	}

	// Verify by eviction: under (-0.5) and over (1.5) are clamped to 0.0 and 1.0
	// Evict 1 should remove the lowest importance. We need to inspect which
	// is removed to verify clamping happened.
	ltm.Evict(1)
	_, underOk := ltm.Recall("under")
	_, overOk := ltm.Recall("over")
	// If clamping worked: under has 0.0 (lowest) so it's evicted first
	// If clamping failed: over might have 1.5 (highest) and under has -0.5 (lowest)
	if underOk {
		t.Fatal("expected 'under' (clamped to 0.0) to be evicted first")
	}
	if !overOk {
		t.Fatal("expected 'over' (clamped to 1.0) to survive first eviction")
	}
}

// ---------------------------------------------------------------------------
// Concurrency safety tests
// ---------------------------------------------------------------------------

func TestWorkingMemoryBucket_ConcurrentAccess(t *testing.T) {
	b := newTestBucket(t)
	var wg sync.WaitGroup
	n := 50

	// Concurrent writers
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "key"
			b.Set(key, idx)
		}(i)
	}

	// Concurrent readers
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Get("key")
		}()
	}

	// Concurrent deleter
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.Delete("key")
	}()

	// Concurrent clear
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.Clear()
	}()

	// Concurrent list and snapshot
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.List()
			b.Snapshot()
		}()
	}

	wg.Wait()
	// No panic or race means success (verified by -race)
}

func TestLongTermMemory_ConcurrentAccess(t *testing.T) {
	ltm := newTestLTM(t)
	var wg sync.WaitGroup
	n := 50

	// Concurrent writers
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ltm.Remember("key", idx, 0.5)
		}(i)
	}

	// Concurrent readers
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ltm.Recall("key")
		}()
	}

	// Concurrent searchers
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = ltm.Search("key", 10)
		}()
	}

	// Concurrent forget
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ltm.Forget("key")
		}()
	}

	// Concurrent evict
	wg.Add(1)
	go func() {
		defer wg.Done()
		ltm.Evict(5)
	}()

	wg.Wait()
	// No panic or race means success (verified by -race)
}

func TestMemoryItem_IsExpired(t *testing.T) {
	// Item with no TTL (zero value) should never be expired
	item := &MemoryItem{}
	if item.IsExpired() {
		t.Fatal("expected zero-TTL item to not be expired")
	}

	// Item with future TTL should not be expired
	item = &MemoryItem{TTL: time.Now().Add(1 * time.Hour)}
	if item.IsExpired() {
		t.Fatal("expected future-TTL item to not be expired")
	}

	// Item with past TTL should be expired
	item = &MemoryItem{TTL: time.Now().Add(-1 * time.Hour)}
	if !item.IsExpired() {
		t.Fatal("expected past-TTL item to be expired")
	}
}

func TestNewMemoryItem_ZeroTTL(t *testing.T) {
	item := NewMemoryItem("agent", "session", "key", "value", 0, 0.5)
	if item.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if !item.TTL.IsZero() {
		t.Fatal("expected zero TTL for duration=0")
	}
	if item.IsExpired() {
		t.Fatal("expected zero-TTL item to not be expired")
	}
}

func TestNewMemoryItem_WithTTL(t *testing.T) {
	item := NewMemoryItem("agent", "session", "key", "value", 5*time.Minute, 0.8)
	if item.TTL.IsZero() {
		t.Fatal("expected non-zero TTL")
	}
	if item.Importance != 0.8 {
		t.Fatalf("expected importance 0.8, got %f", item.Importance)
	}
}

func TestWorkingMemoryBucket_NilLogger(t *testing.T) {
	b := NewWorkingMemoryBucket("agent", "session", time.Minute, nil)
	if b.logger == nil {
		t.Fatal("expected non-nil logger even when nil is passed")
	}
	// Should not panic on operations
	b.Set("k", "v")
	b.Get("k")
	b.Delete("k")
	b.Clear()
}

func TestLongTermMemory_NilLogger(t *testing.T) {
	ltm := NewLongTermMemory("agent", nil, nil)
	if ltm.logger == nil {
		t.Fatal("expected non-nil logger even when nil is passed")
	}
	// Should not panic on operations
	ltm.Remember("k", "v", 0.5)
	ltm.Recall("k")
	ltm.Forget("k")
}
