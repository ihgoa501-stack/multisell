package toolbridge

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// mockDriver — a test double implementing ToolDriver
// ---------------------------------------------------------------------------

type mockDriver struct {
	name        string
	fetchPageFn func(ctx context.Context, url string) (*PageData, error)
	healthFn    func() (string, time.Duration)
	mu          sync.Mutex
}

func (m *mockDriver) FetchPage(ctx context.Context, url string) (*PageData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fetchPageFn != nil {
		return m.fetchPageFn(ctx, url)
	}
	return nil, errors.New("not implemented")
}

func (m *mockDriver) Health() (string, time.Duration) {
	if m.healthFn != nil {
		return m.healthFn()
	}
	return "offline", 0
}

func (m *mockDriver) Name() string {
	return m.name
}

// Helper constructors for the common cases.
func okDriver(name string, data *PageData) *mockDriver {
	return &mockDriver{
		name: name,
		fetchPageFn: func(_ context.Context, _ string) (*PageData, error) {
			return data, nil
		},
		healthFn: func() (string, time.Duration) {
			return "online", time.Millisecond
		},
	}
}

func failDriver(name string, err error) *mockDriver {
	return &mockDriver{
		name: name,
		fetchPageFn: func(_ context.Context, _ string) (*PageData, error) {
			return nil, err
		},
		healthFn: func() (string, time.Duration) {
			return "offline", 0
		},
	}
}

func ctxDoneDriver(name string) *mockDriver {
	return &mockDriver{
		name: name,
		fetchPageFn: func(ctx context.Context, _ string) (*PageData, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
		healthFn: func() (string, time.Duration) {
			return "offline", 0
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRoute_PrimarySucceeds(t *testing.T) {
	logger := zap.NewNop()
	expected := &PageData{Title: "Test Product", Price: 99.99}
	primary := okDriver("primary", expected)
	fallback := okDriver("fallback", &PageData{Title: "should-not-be-used"})

	bridge := New(primary, fallback, logger, 10)
	result, err := bridge.Route(context.Background(), "https://detail.1688.com/test")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Title != "Test Product" {
		t.Fatalf("expected title 'Test Product', got: %q", result.Title)
	}
	if result.Price != 99.99 {
		t.Fatalf("expected price 99.99, got: %f", result.Price)
	}
}

func TestRoute_FallbackUsedWhenPrimaryFails(t *testing.T) {
	logger := zap.NewNop()
	primary := failDriver("primary", errors.New("primary unavailable"))
	fallback := okDriver("fallback", &PageData{Title: "Fallback Product", Price: 49.99})

	bridge := New(primary, fallback, logger, 10)
	result, err := bridge.Route(context.Background(), "https://detail.1688.com/test")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Title != "Fallback Product" {
		t.Fatalf("expected title 'Fallback Product', got: %q", result.Title)
	}
	if result.Price != 49.99 {
		t.Fatalf("expected price 49.99, got: %f", result.Price)
	}
}

func TestRoute_AllDriversFail(t *testing.T) {
	logger := zap.NewNop()
	primary := failDriver("primary", errors.New("primary error"))
	fallback := failDriver("fallback", errors.New("fallback error"))

	bridge := New(primary, fallback, logger, 10)
	result, err := bridge.Route(context.Background(), "https://detail.1688.com/test")

	if result != nil {
		t.Fatalf("expected nil result, got: %v", result)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAllDriversFailed) {
		t.Fatalf("expected ErrAllDriversFailed, got: %v", err)
	}
}

func TestRoute_NoDriversRegistered(t *testing.T) {
	logger := zap.NewNop()
	bridge := New(nil, nil, logger, 10)

	_, err := bridge.Route(context.Background(), "https://detail.1688.com/test")
	if !errors.Is(err, ErrNoDriversRegistered) {
		t.Fatalf("expected ErrNoDriversRegistered, got: %v", err)
	}
}

func TestRoute_FallbackOnly(t *testing.T) {
	logger := zap.NewNop()
	fallback := okDriver("fallback", &PageData{Title: "Direct Fallback"})

	bridge := New(nil, fallback, logger, 10)
	result, err := bridge.Route(context.Background(), "https://detail.1688.com/test")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Title != "Direct Fallback" {
		t.Fatalf("expected title 'Direct Fallback', got: %q", result.Title)
	}
}

func TestRoute_PrimaryOnly(t *testing.T) {
	logger := zap.NewNop()
	primary := okDriver("primary", &PageData{Title: "Primary Only"})

	bridge := New(primary, nil, logger, 10)
	result, err := bridge.Route(context.Background(), "https://detail.1688.com/test")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Title != "Primary Only" {
		t.Fatalf("expected title 'Primary Only', got: %q", result.Title)
	}
}

func TestRoute_PrimaryContextTimeout(t *testing.T) {
	logger := zap.NewNop()
	primary := ctxDoneDriver("primary")
	fallback := okDriver("fallback", &PageData{Title: "Fallback After Timeout"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	// Give the context timeout a moment to fire.
	time.Sleep(5 * time.Millisecond)

	bridge := New(primary, fallback, logger, 10)
	result, err := bridge.Route(ctx, "https://detail.1688.com/test")

	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if result.Title != "Fallback After Timeout" {
		t.Fatalf("expected fallback title, got: %q", result.Title)
	}
}

func TestDLQ_EntryWrittenOnDoubleFailure(t *testing.T) {
	logger := zap.NewNop()
	primary := failDriver("primary", errors.New("primary error"))
	fallback := failDriver("fallback", errors.New("fallback error"))

	bridge := New(primary, fallback, logger, 10)

	_, err := bridge.Route(context.Background(), "https://detail.1688.com/fail")
	if err == nil {
		t.Fatal("expected error from Route")
	}

	entries := bridge.DLQEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 DLQ entry, got %d", len(entries))
	}
	if entries[0].SourceURL != "https://detail.1688.com/fail" {
		t.Fatalf("unexpected source URL: %q", entries[0].SourceURL)
	}
	if entries[0].PrimaryErr != "primary error" {
		t.Fatalf("unexpected primary error: %q", entries[0].PrimaryErr)
	}
	if entries[0].FallbackErr != "fallback error" {
		t.Fatalf("unexpected fallback error: %q", entries[0].FallbackErr)
	}
}

func TestDLQ_MaxSizeEnforced(t *testing.T) {
	logger := zap.NewNop()
	primary := failDriver("primary", errors.New("err"))
	fallback := failDriver("fallback", errors.New("err"))

	bridge := New(primary, fallback, logger, 3)

	for i := 0; i < 5; i++ {
		_, _ = bridge.Route(context.Background(), "https://test.com/page")
	}

	entries := bridge.DLQEntries()
	if len(entries) > 3 {
		t.Fatalf("expected max 3 DLQ entries, got %d", len(entries))
	}
}

func TestDLQ_NoEntryOnSuccess(t *testing.T) {
	logger := zap.NewNop()
	primary := okDriver("primary", &PageData{Title: "OK"})
	bridge := New(primary, nil, logger, 10)

	_, err := bridge.Route(context.Background(), "https://detail.1688.com/ok")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	entries := bridge.DLQEntries()
	if len(entries) != 0 {
		t.Fatalf("expected 0 DLQ entries on success, got %d", len(entries))
	}
}

func TestHealth_Aggregation(t *testing.T) {
	logger := zap.NewNop()
	primary := okDriver("plugin", &PageData{})
	fallback := okDriver("playwright", &PageData{})

	bridge := New(primary, fallback, logger, 10)
	health := bridge.Health()

	if len(health) != 2 {
		t.Fatalf("expected 2 health entries, got %d", len(health))
	}

	p, ok := health["plugin"]
	if !ok {
		t.Fatal("expected health entry for 'plugin'")
	}
	if p.Status != "online" {
		t.Fatalf("expected plugin status 'online', got: %q", p.Status)
	}
	if p.Latency != time.Millisecond {
		t.Fatalf("expected plugin latency 1ms, got: %v", p.Latency)
	}

	pw, ok := health["playwright"]
	if !ok {
		t.Fatal("expected health entry for 'playwright'")
	}
	if pw.Status != "online" {
		t.Fatalf("expected playwright status 'online', got: %q", pw.Status)
	}
}

func TestHealth_EmptyOnNoDrivers(t *testing.T) {
	logger := zap.NewNop()
	bridge := New(nil, nil, logger, 10)

	health := bridge.Health()
	if len(health) != 0 {
		t.Fatalf("expected empty health, got %d entries", len(health))
	}
}

func TestSetPrimary_SetFallback(t *testing.T) {
	logger := zap.NewNop()
	bridge := New(nil, nil, logger, 10)

	// Set drivers after construction.
	bridge.SetPrimary(okDriver("plugin", &PageData{Title: "Plugin Result"}))
	bridge.SetFallback(okDriver("playwright", &PageData{Title: "Playwright Result"}))

	// Should get primary result.
	result, err := bridge.Route(context.Background(), "https://test.com")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if result.Title != "Plugin Result" {
		t.Fatalf("expected 'Plugin Result', got: %q", result.Title)
	}
}
