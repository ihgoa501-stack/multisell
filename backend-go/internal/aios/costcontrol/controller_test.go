package costcontrol

import (
	"context"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

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

func TestAllow_DailyBudgetExceeded(t *testing.T) {
	db := dbtest.NewDB(t, &CostLog{})
	logger, _ := zap.NewDevelopment()

	// Seed a cost log above the daily cap.
	db.Create(&CostLog{
		WindowDate: time.Now().UTC(),
		CostUSD:    5.00,
	})

	c := NewController(db, logger, 1.00, 5*time.Minute, 3) // $1 daily cap, already spent $5
	res, err := c.Allow(context.Background(), AllowInput{AgentID: "test", Model: "sonnet", Tokens: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Action != ActionBlock {
		t.Fatalf("expected block, got %s", res.Action)
	}
	if res.Reason == "" {
		t.Fatal("expected non-empty reason for block")
	}
}

func TestAllow_DailyBudgetRemaining(t *testing.T) {
	db := dbtest.NewDB(t, &CostLog{})
	logger, _ := zap.NewDevelopment()

	// Seed a small cost log, well under the cap.
	db.Create(&CostLog{
		WindowDate: time.Now().UTC(),
		CostUSD:    0.50,
	})

	c := NewController(db, logger, 10.00, 5*time.Minute, 3) // $10 daily cap, $0.50 spent
	res, err := c.Allow(context.Background(), AllowInput{AgentID: "test", Model: "sonnet", Tokens: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Action != ActionAllow {
		t.Fatalf("expected allow (under budget), got %s", res.Action)
	}
}
