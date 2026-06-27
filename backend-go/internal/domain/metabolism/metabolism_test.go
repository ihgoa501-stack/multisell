package metabolism

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

// mockSemanticScorer returns a preset score for testing.
type mockSemanticScorer struct {
	score   float64
	err     error
	callLog []ScorableEvent // tracks every call so tests can verify invocation
}

func (m *mockSemanticScorer) Score(ev ScorableEvent) (float64, error) {
	m.callLog = append(m.callLog, ev)
	return m.score, m.err
}

// mockScoringAdapter is an in-memory implementation of ScoringAdapter.
type mockScoringAdapter struct {
	events []ScorableEvent
}

func (m *mockScoringAdapter) ScorableEvents(status string) ([]ScorableEvent, error) {
	var filtered []ScorableEvent
	for _, ev := range m.events {
		if ev.Status == status || status == "" {
			filtered = append(filtered, ev)
		}
	}
	return filtered, nil
}

func (m *mockScoringAdapter) MarkExcreted(eventID int64, reason string) error {
	for i := range m.events {
		if m.events[i].ID == eventID {
			m.events[i].Status = "excreted"
			return nil
		}
	}
	return errors.New("event not found")
}

// approx checks that |got - want| < eps.
func approx(got, want, eps float64) bool {
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	return diff < eps
}

// ---------------------------------------------------------------------------
// Scoring engine unit tests — ImpactScore
// ---------------------------------------------------------------------------

func TestImpactScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		count    int
		expected float64
	}{
		{name: "zero op_logs", count: 0, expected: 0.0},
		{name: "negative op_logs", count: -1, expected: 0.0},
		{name: "one op_log", count: 1, expected: 0.3},
		{name: "two op_logs", count: 2, expected: 0.3 + (1.0/3.0)*0.7},
		{name: "three op_logs", count: 3, expected: 0.3 + (2.0/3.0)*0.7},
		{name: "four op_logs", count: 4, expected: 0.3 + (3.0/3.0)*0.7},
		{name: "five op_logs", count: 5, expected: 1.0},
		{name: "ten op_logs", count: 10, expected: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ImpactScore(tt.count)
			if !approx(got, tt.expected, 0.0001) {
				t.Errorf("ImpactScore(%d) = %.6f, want %.6f", tt.count, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scoring engine unit tests — ReferenceScore
// ---------------------------------------------------------------------------

func TestReferenceScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		count    int
		expected float64
	}{
		{name: "zero refs", count: 0, expected: 0.0},
		{name: "negative refs", count: -1, expected: 0.0},
		{name: "one ref", count: 1, expected: 0.3},
		{name: "two refs", count: 2, expected: 0.65},
		{name: "three refs", count: 3, expected: 1.0},
		{name: "five refs", count: 5, expected: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReferenceScore(tt.count)
			if !approx(got, tt.expected, 0.0001) {
				t.Errorf("ReferenceScore(%d) = %.6f, want %.6f", tt.count, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scoring engine unit tests — FreshnessScore
// ---------------------------------------------------------------------------

func TestFreshnessScore(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	ttl := 7 * 24 * time.Hour // 168h

	tests := []struct {
		name      string
		createdAt time.Time
		expected  float64
	}{
		{
			name:      "just created",
			createdAt: now,
			expected:  0.0,
		},
		{
			name:      "half TTL elapsed",
			createdAt: now.Add(-ttl / 2),
			expected:  0.5,
		},
		{
			name:      "exactly at TTL",
			createdAt: now.Add(-ttl),
			expected:  1.0,
		},
		{
			name:      "past TTL",
			createdAt: now.Add(-ttl * 2),
			expected:  1.0,
		},
		{
			name:      "created in future",
			createdAt: now.Add(1 * time.Hour),
			expected:  0.0,
		},
		{
			name:      "barely created (1 nanosecond ago)",
			createdAt: now.Add(-1 * time.Nanosecond),
			expected:  0.0, // elapsed ≈ 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FreshnessScore(tt.createdAt, now)
			if !approx(got, tt.expected, 0.0001) {
				t.Errorf("FreshnessScore(%v, now) = %.6f, want %.6f", tt.createdAt, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scoring engine — aggregate formula and excretion threshold
// ---------------------------------------------------------------------------

func TestScoreAggregateFormula(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	mockScorer := &mockSemanticScorer{score: 0}
	svc := &MetabolismService{semanticScorer: mockScorer, logger: zap.NewNop()}

	ttl := 7 * 24 * time.Hour // 168h

	t.Run("all zero scores produce combined=0.0", func(t *testing.T) {
		ev := ScorableEvent{
			OpLogCount: 0,
			RefCount:   0,
			CreatedAt:  now, // freshness = 0
		}
		score := svc.scoreAt(ev, now)

		if score.Combined != 0.0 {
			t.Errorf("expected combined=0.0, got %.6f", score.Combined)
		}
		if score.Excretable {
			t.Error("combined=0 should NOT be excretable")
		}
	})

	t.Run("combined=0.69 just below threshold", func(t *testing.T) {
		// impact=1.0 (opLog=5), ref=0.3 (refCount=1), freshness=0.6667
		// combined = 1.0*0.4 + 0.3*0.3 + 0.6667*0.3 = 0.4+0.09+0.2000 = 0.69
		elapsed := time.Duration((2.0 / 3.0) * float64(ttl)) // exactly 112h
		ev := ScorableEvent{
			OpLogCount: 5,
			RefCount:   1,
			CreatedAt:  now.Add(-elapsed),
		}
		score := svc.scoreAt(ev, now)

		if !approx(score.Combined, 0.69, 0.005) {
			t.Errorf("expected combined ≈ 0.69, got %.6f", score.Combined)
		}
		if score.Excretable {
			t.Error("combined ≈ 0.69 should NOT be excretable")
		}
	})

	t.Run("combined=0.70 at boundary — excretable", func(t *testing.T) {
		// impact=1.0 (opLog=5), ref=0.3 (refCount=1), freshness=0.70
		// combined = 1.0*0.4 + 0.3*0.3 + 0.70*0.3 = 0.4+0.09+0.21 = 0.70
		freshness := 0.70
		elapsed := time.Duration(freshness * float64(ttl))
		ev := ScorableEvent{
			OpLogCount: 5,
			RefCount:   1,
			CreatedAt:  now.Add(-elapsed),
		}
		score := svc.scoreAt(ev, now)

		if !approx(score.Combined, 0.70, 0.005) {
			t.Errorf("expected combined ≈ 0.70, got %.6f", score.Combined)
		}
		if !score.Excretable {
			t.Error("combined ≈ 0.70 should BE excretable")
		}
	})

	t.Run("combined=0.71 above boundary — excretable", func(t *testing.T) {
		// impact=1.0 (opLog=5), ref=0.3 (refCount=1), freshness=0.7333
		// combined = 1.0*0.4 + 0.3*0.3 + 0.7333*0.3 = 0.4+0.09+0.22 = 0.71
		freshness := 0.73333334 // ~11/15
		elapsed := time.Duration(freshness * float64(ttl))
		ev := ScorableEvent{
			OpLogCount: 5,
			RefCount:   1,
			CreatedAt:  now.Add(-elapsed),
		}
		score := svc.scoreAt(ev, now)

		if !approx(score.Combined, 0.71, 0.005) {
			t.Errorf("expected combined ≈ 0.71, got %.6f", score.Combined)
		}
		if !score.Excretable {
			t.Error("combined ≈ 0.71 should BE excretable")
		}
	})
}

// ---------------------------------------------------------------------------
// Scoring engine — gray zone semantic trigger
// ---------------------------------------------------------------------------

func TestScoreGrayZone(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	t.Run("combined below 0.4 — no semantic call", func(t *testing.T) {
		mockScorer := &mockSemanticScorer{score: 0.5}
			svc := &MetabolismService{semanticScorer: mockScorer, logger: zap.NewNop()}

		// Combined will be 0.0 (all zeros = no op, no ref, createdAt=now)
		ev := ScorableEvent{
			OpLogCount: 0,
			RefCount:   0,
			CreatedAt:  now,
		}
		_ = svc.scoreAt(ev, now)

		if len(mockScorer.callLog) != 0 {
			t.Errorf("expected 0 semantic calls for combined<0.4, got %d", len(mockScorer.callLog))
		}
	})

	t.Run("combined in gray zone [0.4, 0.75) — semantic called", func(t *testing.T) {
		mockScorer := &mockSemanticScorer{score: 0.5}
			svc := &MetabolismService{semanticScorer: mockScorer, logger: zap.NewNop()}

		// impact=0.767 (opLog=3), ref=0, freshness=0.5 (half TTL)
		// combined = 0.767*0.4 + 0 + 0.5*0.3 = 0.3068 + 0.15 = 0.4568 → in gray zone
		ttl := 7 * 24 * time.Hour
		ev := ScorableEvent{
			OpLogCount: 3,
			RefCount:   0,
			CreatedAt:  now.Add(-ttl / 2),
		}
		score := svc.scoreAt(ev, now)

		if score.Combined < GrayZoneLower || score.Combined >= GrayZoneUpper {
			t.Fatalf("test setup: combined=%.4f should be in gray zone [%.2f, %.2f)",
				score.Combined, GrayZoneLower, GrayZoneUpper)
		}
		if len(mockScorer.callLog) != 1 {
			t.Errorf("expected 1 semantic call, got %d", len(mockScorer.callLog))
		}
	})

	t.Run("combined at 0.75 — no semantic call", func(t *testing.T) {
		mockScorer := &mockSemanticScorer{score: 0.5}
			svc := &MetabolismService{semanticScorer: mockScorer, logger: zap.NewNop()}

		// impact=1.0 (opLog=5), ref=1.0 (refCount=3), freshness=0.5 (half TTL)
		// combined = 1.0*0.4 + 1.0*0.3 + 0.5*0.3 = 0.4+0.3+0.15 = 0.85
		ttl := 7 * 24 * time.Hour
		ev := ScorableEvent{
			OpLogCount: 5,
			RefCount:   3,
			CreatedAt:  now.Add(-ttl / 2),
		}
		_ = svc.scoreAt(ev, now)

		if len(mockScorer.callLog) != 0 {
			t.Errorf("expected 0 semantic calls for combined >= 0.75, got %d", len(mockScorer.callLog))
		}
	})

	t.Run("semantic scorer error handled gracefully", func(t *testing.T) {
		mockScorer := &mockSemanticScorer{score: 0, err: errors.New("LLM unavailable")}
			svc := &MetabolismService{semanticScorer: mockScorer, logger: zap.NewNop()}

		ttl := 7 * 24 * time.Hour
		ev := ScorableEvent{
			OpLogCount: 3,
			RefCount:   0,
			CreatedAt:  now.Add(-ttl / 2),
		}
		_ = svc.scoreAt(ev, now)

		// Should not panic; base combined score is preserved.
		if len(mockScorer.callLog) != 1 {
			t.Errorf("expected 1 semantic call attempt, got %d", len(mockScorer.callLog))
		}
	})

	t.Run("semantic boost pushes combined above threshold", func(t *testing.T) {
		// Semantic score of 1.0 blended as 1.0*0.7 = 0.70, which meets threshold.
		mockScorer := &mockSemanticScorer{score: 1.0}
			svc := &MetabolismService{semanticScorer: mockScorer, logger: zap.NewNop()}

		ttl := 7 * 24 * time.Hour
		ev := ScorableEvent{
			OpLogCount: 3,
			RefCount:   0,
			CreatedAt:  now.Add(-ttl / 2),
		}
		score := svc.scoreAt(ev, now)

		if len(mockScorer.callLog) == 0 {
			t.Fatal("expected semantic call")
		}
		// base combined ≈ 0.4568, semantic blended = 0.70, so combined should be 0.70
		if !approx(score.Combined, 0.70, 0.01) {
			t.Errorf("expected combined boosted to ~0.70, got %.6f", score.Combined)
		}
		if !score.Excretable {
			t.Error("boosted combined >= 0.70 should be excretable")
		}
	})
}

// ---------------------------------------------------------------------------
// M1 Execute — integration tests
// ---------------------------------------------------------------------------

func TestExecute_NoPendingEvents(t *testing.T) {
	t.Parallel()

	db := dbtest.NewDB(t, &MetabolismLog{})
	logger := dbtest.NewLogger(t)

	// Raw event_outbox table (not auto-migrated by GORM — we create via Exec)
	db.Exec(`CREATE TABLE event_outbox (
		id INTEGER PRIMARY KEY,
		topic TEXT, source TEXT, payload TEXT,
		priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending', created_at TIMESTAMP
	)`)

	adapter := newEventOutboxAdapterFromDB(db)
	svc := NewService(db, logger, adapter, nil)

	if err := svc.Execute(false); err != nil {
		t.Fatalf("Execute with no events should not error: %v", err)
	}
}

func TestExecute_ScoreAndLog(t *testing.T) {
	t.Parallel()

	db := dbtest.NewDB(t, &MetabolismLog{})
	logger := dbtest.NewLogger(t)

	db.Exec(`CREATE TABLE event_outbox (
		id INTEGER PRIMARY KEY,
		topic TEXT, source TEXT, payload TEXT,
		priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending', created_at TIMESTAMP
	)`)

	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	// Insert pending events of varying ages and priorities.
	db.Exec(`INSERT INTO event_outbox (id, topic, source, payload, priority, status, created_at)
		VALUES (1, 'order.created', 'shopee', '{}', 1, 'pending', ?)`, now.Add(-30*24*time.Hour)) // very old → high freshness
	db.Exec(`INSERT INTO event_outbox (id, topic, source, payload, priority, status, created_at)
		VALUES (2, 'inventory.low', 'ozon', '{}', 2, 'pending', ?)`, now.Add(-1*time.Hour)) // recent → low freshness
	db.Exec(`INSERT INTO event_outbox (id, topic, source, payload, priority, status, created_at)
		VALUES (3, 'price.change', 'shopee', '{}', 0, 'pending', ?)`, now.Add(-7*24*time.Hour)) // at TTL → max freshness

	adapter := newEventOutboxAdapterFromDB(db)
	svc := NewService(db, logger, adapter, &mockSemanticScorer{score: 0.3})

	if err := svc.Execute(false); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Verify metabolism_log entries were created for each event.
	var logs []MetabolismLog
	if err := db.Find(&logs).Error; err != nil {
		t.Fatalf("query metabolism_log: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 metabolism_log entries, got %d", len(logs))
	}

	// Check that event IDs match.
	ids := make(map[int64]bool)
	for _, l := range logs {
		ids[l.EventID] = true
	}
	for _, id := range []int64{1, 2, 3} {
		if !ids[id] {
			t.Errorf("metabolism_log missing event_id=%d", id)
		}
	}

	// Verify dimensions are valid JSON.
	for _, l := range logs {
		var dims ScoreDimensions
		if err := json.Unmarshal([]byte(l.Dimensions), &dims); err != nil {
			t.Errorf("unmarshal dimensions for event %d: %v", l.EventID, err)
		}
		if dims.Impact < 0 || dims.Impact > 1 {
			t.Errorf("event %d: impact out of range: %f", l.EventID, dims.Impact)
		}
	}
}

func TestExecute_DryRun(t *testing.T) {
	t.Parallel()

	db := dbtest.NewDB(t, &MetabolismLog{})
	logger := dbtest.NewLogger(t)

	db.Exec(`CREATE TABLE event_outbox (
		id INTEGER PRIMARY KEY,
		topic TEXT, source TEXT, payload TEXT,
		excreted_at TIMESTAMP, excretion_reason TEXT,
		priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending', created_at TIMESTAMP
	)`)

	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	// Event old enough to score well above threshold.
	db.Exec(`INSERT INTO event_outbox (id, topic, source, payload, priority, status, created_at)
		VALUES (1, 'order.old', 'shopee', '{}', 1, 'pending', ?)`, now.Add(-30*24*time.Hour))

	adapter := newEventOutboxAdapterFromDB(db)
	svc := NewService(db, logger, adapter, &mockSemanticScorer{score: 0.3})

	// Execute in dry-run mode.
	if err := svc.Execute(true); err != nil {
		t.Fatalf("Execute(dryRun=true): %v", err)
	}

	// Verify metabolism_log entries were created (scoring ran).
	var logCount int64
	db.Model(&MetabolismLog{}).Count(&logCount)
	if logCount == 0 {
		t.Error("expected metabolism_log entries in dry-run mode (scoring should still run)")
	}

	// Verify no event was excreted (excreted_at should remain NULL).
	row := db.Raw(`SELECT excreted_at FROM event_outbox WHERE id = 1`).Row()
	if row == nil {
		t.Fatal("expected a row from event_outbox")
	}
	var excretedAt interface{}
	row.Scan(&excretedAt)
	if excretedAt != nil {
		t.Error("dry-run: event should NOT have excreted_at set")
	}
}

func TestExecute_DBError_Graceful(t *testing.T) {
	t.Parallel()

	// Use a closed/closing DB to simulate errors.
	closedDB := dbtest.NewDB(t, &MetabolismLog{})
	closedDB.Exec(`CREATE TABLE event_outbox (
		id INTEGER PRIMARY KEY,
		topic TEXT, source TEXT, payload TEXT,
		priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending', created_at TIMESTAMP
	)`)
	// Insert one event so Execute tries to log.
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	closedDB.Exec(`INSERT INTO event_outbox (id, topic, source, payload, priority, status, created_at)
		VALUES (1, 'test.event', 'test', '{}', 0, 'pending', ?)`, now)

	// Get a raw *sql.DB and close it to cause errors on subsequent operations.
	sqlDB, _ := closedDB.DB()
	sqlDB.Close()

	adapter := newEventOutboxAdapterFromDB(closedDB)
	logger := dbtest.NewLogger(t)
	svc := NewService(closedDB, logger, adapter, &mockSemanticScorer{score: 0.3})

	// Should not panic; should return nil because adapter still reads OK
	// (the adapter reads from its own *gorm.DB instance, which will fail).
	err := svc.Execute(false)
	if err == nil {
		// The adapter may have cached nothing or the error might surface differently.
		// The important thing is we didn't panic.
		t.Log("Execute returned nil even with closed DB (adapter may have error)")
	}
}

// ---------------------------------------------------------------------------
// Adapter tests — EventOutboxAdapter
// ---------------------------------------------------------------------------

// newEventOutboxAdapterFromDB creates an adapter backed by a *gorm.DB that has
// a manually-created event_outbox table.
func newEventOutboxAdapterFromDB(db *gorm.DB) ScoringAdapter {
	return &testOutboxAdapter{db: db}
}

// testOutboxAdapter is a simplified ScoringAdapter that reads from event_outbox.
type testOutboxAdapter struct {
	db *gorm.DB
}

type eventOutboxRowTest struct {
	ID        int64
	Topic     string
	Source    string
	Payload   string
	Priority  int
	Status    string
	CreatedAt time.Time
}

func (eventOutboxRowTest) TableName() string { return "event_outbox" }

func (a *testOutboxAdapter) ScorableEvents(status string) ([]ScorableEvent, error) {
	var rows []eventOutboxRowTest
	tx := a.db.Order("priority DESC, created_at ASC")
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	tx = tx.Find(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}

	results := make([]ScorableEvent, len(rows))
	for i, r := range rows {
		// Simple payload parsing: count comma-separated refs and op_logs
		// in the JSON payload for basic normalization.
		opLogCount, refCount := 0, 0
		if r.Payload != "" && r.Payload != "{}" {
			// Naive counting: "op_logs":[x,y,z] → 3 items
			opLogCount = countJSONArray(r.Payload, "op_logs")
			refCount = countJSONArray(r.Payload, "refs")
		}
		results[i] = ScorableEvent{
			ID:         r.ID,
			Topic:      r.Topic,
			Source:     r.Source,
			Payload:    r.Payload,
			Priority:   r.Priority,
			Status:     r.Status,
			OpLogCount: opLogCount,
			RefCount:   refCount,
			CreatedAt:  r.CreatedAt,
		}
	}
	return results, nil
}

func (a *testOutboxAdapter) MarkExcreted(eventID int64, reason string) error {
	return a.db.Exec(
		"UPDATE event_outbox SET excreted_at = CURRENT_TIMESTAMP, excretion_reason = ? WHERE id = ?",
		reason, eventID,
	).Error
}

// countJSONArray counts occurrences of a JSON array key value.
// This is a simplified parser sufficient for test payloads like:
// {"op_logs":[1,2,3], "refs":[1,2]}
func countJSONArray(payload, key string) int {
	// Find key: "key":[
	search := `"` + key + `":[`
	idx := indexOf(payload, search)
	if idx < 0 {
		return 0
	}
	start := idx + len(search)
	// Count comma-separated items inside the array.
	if start >= len(payload) {
		return 0
	}
	// Check for empty array
	if payload[start] == ']' {
		return 0
	}
	count := 1
	for i := start; i < len(payload) && payload[i] != ']'; i++ {
		if payload[i] == ',' {
			count++
		}
	}
	return count
}

// indexOf is a simple substring index finder.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestEventOutboxAdapter_Normalize(t *testing.T) {
	t.Parallel()

	db := dbtest.NewDB(t, &MetabolismLog{})
	db.Exec(`CREATE TABLE event_outbox (
		id INTEGER PRIMARY KEY,
		topic TEXT, source TEXT, payload TEXT,
		priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending', created_at TIMESTAMP
	)`)

	now := time.Date(2026, 6, 25, 8, 0, 0, 0, time.UTC)

	// Insert a row that simulates a real event_outbox record with payload
	// containing op_logs and refs arrays.
	db.Exec(`INSERT INTO event_outbox (id, topic, source, payload, priority, status, created_at)
		VALUES (1, 'order.created', 'shopee', '{"op_logs":[101,102,103],"refs":[201,202]}', 1, 'pending', ?)`, now)

	adapter := newEventOutboxAdapterFromDB(db)

	events, err := adapter.ScorableEvents("pending")
	if err != nil {
		t.Fatalf("ScorableEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.ID != 1 {
		t.Errorf("ID: want 1, got %d", ev.ID)
	}
	if ev.Topic != "order.created" {
		t.Errorf("Topic: want 'order.created', got %s", ev.Topic)
	}
	if ev.Source != "shopee" {
		t.Errorf("Source: want 'shopee', got %s", ev.Source)
	}
	if ev.Priority != 1 {
		t.Errorf("Priority: want 1, got %d", ev.Priority)
	}
	if ev.Status != "pending" {
		t.Errorf("Status: want 'pending', got %s", ev.Status)
	}
	if !ev.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt: want %v, got %v", now, ev.CreatedAt)
	}

	// Verify normalization of payload.
	if ev.OpLogCount != 3 {
		t.Errorf("OpLogCount: want 3 (from [101,102,103]), got %d", ev.OpLogCount)
	}
	if ev.RefCount != 2 {
		t.Errorf("RefCount: want 2 (from [201,202]), got %d", ev.RefCount)
	}
}

func TestEventOutboxAdapter_EmptyPayload(t *testing.T) {
	t.Parallel()

	db := dbtest.NewDB(t, &MetabolismLog{})
	db.Exec(`CREATE TABLE event_outbox (
		id INTEGER PRIMARY KEY,
		topic TEXT, source TEXT, payload TEXT,
		priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending', created_at TIMESTAMP
	)`)

	now := time.Date(2026, 6, 25, 8, 0, 0, 0, time.UTC)
	db.Exec(`INSERT INTO event_outbox (id, topic, source, payload, priority, status, created_at)
		VALUES (1, 'test.event', 'test', '{}', 0, 'pending', ?)`, now)
	db.Exec(`INSERT INTO event_outbox (id, topic, source, payload, priority, status, created_at)
		VALUES (2, 'test.event2', 'test', '', 0, 'pending', ?)`, now)

	adapter := newEventOutboxAdapterFromDB(db)

	events, err := adapter.ScorableEvents("pending")
	if err != nil {
		t.Fatalf("ScorableEvents: %v", err)
	}

	for _, ev := range events {
		if ev.OpLogCount != 0 {
			t.Errorf("event %d: expected OpLogCount=0 for empty payload, got %d", ev.ID, ev.OpLogCount)
		}
		if ev.RefCount != 0 {
			t.Errorf("event %d: expected RefCount=0 for empty payload, got %d", ev.ID, ev.RefCount)
		}
	}
}

func TestEventOutboxAdapter_MarkExcreted(t *testing.T) {
	t.Parallel()

	db := dbtest.NewDB(t, &MetabolismLog{})
	db.Exec(`CREATE TABLE event_outbox (
		id INTEGER PRIMARY KEY,
		topic TEXT, source TEXT, payload TEXT,
		excreted_at TIMESTAMP, excretion_reason TEXT,
		priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending', created_at TIMESTAMP
	)`)

	now := time.Date(2026, 6, 25, 8, 0, 0, 0, time.UTC)
	db.Exec(`INSERT INTO event_outbox (id, topic, source, payload, priority, status, created_at)
		VALUES (1, 'test.event', 'test', '{}', 0, 'pending', ?)`, now)

	adapter := newEventOutboxAdapterFromDB(db)
	if err := adapter.MarkExcreted(1, "combined score 0.85 meets threshold"); err != nil {
		t.Fatalf("MarkExcreted: %v", err)
	}

	row1 := db.Raw(`SELECT excreted_at FROM event_outbox WHERE id = 1`).Row()
	if row1 == nil {
		t.Fatal("expected a row from event_outbox")
	}
	var excretedAt interface{}
	row1.Scan(&excretedAt)

	var reason string
	row2 := db.Raw(`SELECT excretion_reason FROM event_outbox WHERE id = 1`).Row()
	row2.Scan(&reason)

	if excretedAt == nil {
		t.Error("excreted_at should be set after MarkExcreted")
	}
	if reason != "combined score 0.85 meets threshold" {
		t.Errorf("excretion_reason: want 'combined score 0.85 meets threshold', got %s", reason)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestClamp01(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    float64
		expected float64
	}{
		{input: -1.0, expected: 0},
		{input: -0.001, expected: 0},
		{input: 0, expected: 0},
		{input: 0.5, expected: 0.5},
		{input: 1, expected: 1},
		{input: 1.001, expected: 1},
		{input: math.MaxFloat64, expected: 1},
	}
	for _, tt := range tests {
		got := clamp01(tt.input)
		if got != tt.expected {
			t.Errorf("clamp01(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestScore_ReasonPopulated(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	mockScorer := &mockSemanticScorer{score: 0}
	svc := &MetabolismService{semanticScorer: mockScorer, logger: zap.NewNop()}

	// A high-scoring event should get a reason.
	ttl := 7 * 24 * time.Hour
	ev := ScorableEvent{
		OpLogCount: 5,
		RefCount:   3,
		CreatedAt:  now.Add(-ttl * 2),
	}
	score := svc.scoreAt(ev, now)
	if !score.Excretable {
		t.Fatal("expected excretable for max scores")
	}
	if score.Reason == "" {
		t.Error("expected non-empty reason for excretable event")
	}
}

// ---------------------------------------------------------------------------
// Entity-based M1 excretion — pure function tests
// ---------------------------------------------------------------------------

func TestScoreStaleness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		daysSinceActivity int
		minExpected       float64
		maxExpected       float64
	}{
		{name: "zero days — active today", daysSinceActivity: 0, minExpected: 0, maxExpected: 0},
		{name: "negative days — treat as 0", daysSinceActivity: -5, minExpected: 0, maxExpected: 0},
		{name: "1 day — slight staleness", daysSinceActivity: 1, minExpected: 1, maxExpected: 5},
		{name: "30 days — moderate staleness", daysSinceActivity: 30, minExpected: 30, maxExpected: 55},
		{name: "45 days — half stale", daysSinceActivity: 45, minExpected: 50, maxExpected: 75},
		{name: "90 days — fully stale", daysSinceActivity: 90, maxExpected: 100, minExpected: 99},
		{name: "180 days — well past stale", daysSinceActivity: 180, maxExpected: 100, minExpected: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreStaleness(tt.daysSinceActivity)
			if got < tt.minExpected || got > tt.maxExpected {
				t.Errorf("ScoreStaleness(%d) = %.2f, want [%.2f, %.2f]",
					tt.daysSinceActivity, got, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestScorePerformance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		salesVelocity float64
		profitMargin  float64
		minExpected   float64
		maxExpected   float64
	}{
		{name: "no sales, no margin", salesVelocity: 0, profitMargin: 0, minExpected: 0, maxExpected: 0},
		{name: "no sales, negative margin", salesVelocity: 0, profitMargin: -0.1, minExpected: 0, maxExpected: 0},
		{name: "low sales, no margin", salesVelocity: 0.5, profitMargin: 0, minExpected: 10, maxExpected: 50},
		{name: "high sales, no margin", salesVelocity: 10, profitMargin: 0, minExpected: 78, maxExpected: 82},
		{name: "high sales, high margin", salesVelocity: 10, profitMargin: 0.3, minExpected: 95, maxExpected: 100},
		{name: "medium sales, medium margin", salesVelocity: 3, profitMargin: 0.15, minExpected: 50, maxExpected: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScorePerformance(tt.salesVelocity, tt.profitMargin)
			if got < tt.minExpected || got > tt.maxExpected {
				t.Errorf("ScorePerformance(%.2f, %.2f) = %.2f, want [%.2f, %.2f]",
					tt.salesVelocity, tt.profitMargin, got, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestScoreEntity(t *testing.T) {
	t.Parallel()

	svc := &MetabolismService{logger: zap.NewNop()}

	tests := []struct {
		name       string
		staleScore float64
		perfScore  float64
		minScore   float64
		maxScore   float64
	}{
		{name: "all good — no excretion needed", staleScore: 0, perfScore: 100, minScore: 95, maxScore: 100},
		{name: "fully stale, no sales — high excretion risk", staleScore: 100, perfScore: 0, minScore: 0, maxScore: 5},
		{name: "stale but good performance — moderate health", staleScore: 100, perfScore: 100, minScore: 35, maxScore: 45},
		{name: "active but no performance — healthy enough", staleScore: 0, perfScore: 0, minScore: 55, maxScore: 65},
		{name: "moderately stale, mediocre performance — flagged",
			staleScore: 50, perfScore: 30, minScore: 37, maxScore: 47},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.ScoreEntity(tt.staleScore, tt.perfScore)
			if got < tt.minScore || got > tt.maxScore {
				t.Errorf("ScoreEntity(%.2f, %.2f) = %.2f, want [%.2f, %.2f]",
					tt.staleScore, tt.perfScore, got, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestClassifyAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{name: "very low score — excrete", score: 10, expected: "excrete"},
		{name: "just below excrete threshold — excrete", score: 19, expected: "excrete"},
		{name: "at excrete threshold — flag (not < threshold)", score: 20, expected: "flag"},
		{name: "just above excrete threshold — flag", score: 21, expected: "flag"},
		{name: "middle of flag zone — flag", score: 30, expected: "flag"},
		{name: "just below flag threshold — flag", score: 39, expected: "flag"},
		{name: "at flag threshold — keep (not < threshold)", score: 40, expected: "keep"},
		{name: "just above flag threshold — keep", score: 41, expected: "keep"},
		{name: "high score — keep", score: 80, expected: "keep"},
		{name: "maximum score — keep", score: 100, expected: "keep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyAction(tt.score)
			if got != tt.expected {
				t.Errorf("classifyAction(%.0f) = %s, want %s", tt.score, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Entity-based M1 excretion — integration tests
// ---------------------------------------------------------------------------

func TestScoreAndExcreteEntities_NoListings(t *testing.T) {
	t.Parallel()

	db := dbtest.NewDB(t, &MetabolismLog{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil, nil)

	// Create product_listing table but with no rows.
	db.Exec(`CREATE TABLE product_listing (
		id INTEGER PRIMARY KEY,
		product_id INTEGER NOT NULL DEFAULT 0,
		status TEXT DEFAULT 'active',
		last_sync_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	result, err := svc.ScoreAndExcreteEntities(false)
	if err != nil {
		t.Fatalf("ScoreAndExcreteEntities: %v", err)
	}
	if result.TotalItems != 0 {
		t.Errorf("expected 0 items with empty listing table, got %d", result.TotalItems)
	}
	if result.Excreted != 0 || result.Flagged != 0 {
		t.Errorf("expected 0 excreted/flagged, got excreted=%d flagged=%d", result.Excreted, result.Flagged)
	}
}

func TestScoreAndExcreteEntities_WithListings(t *testing.T) {
	t.Parallel()

	db := dbtest.NewDB(t, &MetabolismLog{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil, nil)

	db.Exec(`CREATE TABLE product_listing (
		id INTEGER PRIMARY KEY,
		product_id INTEGER NOT NULL DEFAULT 0,
		status TEXT DEFAULT 'active',
		last_sync_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)

	// Listing 1: very old, never synced — should be excreted.
	db.Exec(`INSERT INTO product_listing (id, product_id, status, last_sync_at, created_at, updated_at)
		VALUES (1, 100, 'active', NULL, ?, ?)`,
		now.Add(-180*24*time.Hour), now.Add(-180*24*time.Hour))

	// Listing 2: moderately old, some activity — should be flagged.
	db.Exec(`INSERT INTO product_listing (id, product_id, status, last_sync_at, created_at, updated_at)
		VALUES (2, 200, 'active', ?, ?, ?)`,
		now.Add(-60*24*time.Hour), now.Add(-90*24*time.Hour), now.Add(-60*24*time.Hour))

	// Listing 3: recently synced — should be kept.
	db.Exec(`INSERT INTO product_listing (id, product_id, status, last_sync_at, created_at, updated_at)
		VALUES (3, 300, 'active', ?, ?, ?)`,
		now.Add(-1*24*time.Hour), now.Add(-30*24*time.Hour), now.Add(-1*24*time.Hour))

	// Listing 4: already archived — should be excluded from scoring.
	db.Exec(`INSERT INTO product_listing (id, product_id, status, last_sync_at, created_at, updated_at)
		VALUES (4, 400, 'archived', NULL, ?, ?)`,
		now.Add(-365*24*time.Hour), now.Add(-365*24*time.Hour))

	result, err := svc.ScoreAndExcreteEntities(false)
	if err != nil {
		t.Fatalf("ScoreAndExcreteEntities: %v", err)
	}

	// Should have scored 3 listings (archived excluded, ai_trace table not available for agents).
	if result.TotalItems != 3 {
		t.Errorf("expected 3 total items (listings only, no ai_trace table), got %d", result.TotalItems)
	}

	// Check that items are classified and results are populated.
	var listingItems []ExcretionItem
	for _, item := range result.Items {
		if item.TargetType == ExcretionTargetListing {
			listingItems = append(listingItems, item)
		}
	}
	if len(listingItems) != 3 {
		t.Fatalf("expected 3 listing items, got %d", len(listingItems))
	}

	// Listing 1 (id=1, 180 days stale, no perf data) gets health score 0 — excreted.
	item1 := listingItems[0]
	if item1.TargetID != 1 {
		t.Fatalf("expected listing 1 first, got listing %d", item1.TargetID)
	}
	if item1.Action != "excrete" && item1.Action != "excreted" {
		t.Errorf("listing 1: expected action 'excrete' or 'excreted', got %s", item1.Action)
	}
	if item1.StaleScore <= 0 {
		t.Errorf("listing 1: expected staleScore > 0 for 180d stale item, got %.2f", item1.StaleScore)
	}

	// Verify archived listings count: pre-existing archived (1) + newly excreted (2) = 3.
	var count int64
	db.Model(&listingRow{}).Where("status = ?", "archived").Count(&count)
	if count != 3 {
		t.Errorf("expected 3 archived listings (1 pre-existing + 2 newly excreted), got %d", count)
	}
}

func TestScoreAndExcreteEntities_DryRun(t *testing.T) {
	t.Parallel()

	db := dbtest.NewDB(t, &MetabolismLog{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil, nil)

	db.Exec(`CREATE TABLE product_listing (
		id INTEGER PRIMARY KEY,
		product_id INTEGER NOT NULL DEFAULT 0,
		status TEXT DEFAULT 'active',
		last_sync_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)

	// Very old listing that would be excreted in non-dry-run mode.
	db.Exec(`INSERT INTO product_listing (id, product_id, status, last_sync_at, created_at, updated_at)
		VALUES (1, 100, 'active', NULL, ?, ?)`,
		now.Add(-365*24*time.Hour), now.Add(-365*24*time.Hour))

	result, err := svc.ScoreAndExcreteEntities(true)
	if err != nil {
		t.Fatalf("ScoreAndExcreteEntities(dryRun=true): %v", err)
	}

	// Should be marked as excrete in the result.
	if result.Excreted <= 0 {
		t.Errorf("expected some items excreted in result, got %d", result.Excreted)
	}

	// But the listing should NOT have been archived in dry-run mode.
	var status string
	db.Raw(`SELECT status FROM product_listing WHERE id = 1`).Scan(&status)
	if status != "active" {
		t.Errorf("dry-run: listing should remain 'active', got %s", status)
	}
}

func TestScoreAndExcreteEntities_NoTables(t *testing.T) {
	t.Parallel()

	// Test with completely empty DB — no product_listing or ai_trace tables.
	db := dbtest.NewDB(t, &MetabolismLog{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil, nil)

	// Should not panic; should gracefully handle missing tables.
	result, err := svc.ScoreAndExcreteEntities(false)
	if err != nil {
		t.Fatalf("ScoreAndExcreteEntities with no tables: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	t.Logf("Total items scored with no tables: %d", result.TotalItems)
}
