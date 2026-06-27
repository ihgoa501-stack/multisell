package costcontrol

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// fakeDB implements a minimal stub for DB reads/writes in tests.
// ponytail: no GORM test helper needed — budget tests are pure logic.
type fakeDB struct {
	mu    sync.Mutex
	spend float64
}

func (f *fakeDB) DailySpend() (*CostLogSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &CostLogSummary{TotalCost: f.spend}, nil
}

func (f *fakeDB) AddCost(c float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.spend += c
}

func TestAllow_Unlimited(t *testing.T) {
	logger := zap.NewNop()
	c := NewController(nil, logger, 0, 5*time.Minute, 3)
	res, err := c.Allow(context.Background(), AllowInput{AgentID: "test", Model: "sonnet", Tokens: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Action != ActionAllow {
		t.Fatalf("expected allow, got %s", res.Action)
	}
}

func TestRecord_NoPanic(t *testing.T) {
	logger := zap.NewNop()
	// With unlimited budget, record should not write to DB (nil db).
	c := NewController(nil, logger, 0, 5*time.Minute, 3)
	rec := RecordInput{AgentID: "test", Model: "claude-haiku-4", TokensIn: 100, TokensOut: 50, CostUSD: 0.0001}
	c.Record(context.Background(), rec) // no panic
}
