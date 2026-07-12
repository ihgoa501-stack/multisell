package command

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func idempotentAction(key string) AgentAction {
	return AgentAction{ActionType: "stock_alert", AgentID: "A5", Actor: "system", RiskLevel: RiskLow, Mode: ModeProduction, IdempotencyKey: key}
}

func TestDispatchSafe_IdempotencyReplaysSuccessfulResult(t *testing.T) {
	db := dbtest.NewDB(t, &ActionExecution{})
	d := NewDispatcher(dbtest.NewLogger(t), WithIdempotencyStore(NewGormIdempotencyStore(db, time.Minute)))
	var calls atomic.Int32
	d.Register("stock_alert", func(context.Context, map[string]interface{}) (*Result, error) {
		calls.Add(1)
		return &Result{Success: true, BusinessID: "alert-1", AfterSnapshot: map[string]interface{}{"status": "created"}}, nil
	})
	action := idempotentAction("stock-alert:sku-1")

	first, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("handler calls = %d, want 1", calls.Load())
	}
	if first.BusinessID != "alert-1" || second.BusinessID != first.BusinessID || second.AfterSnapshot["status"] != "created" {
		t.Fatalf("replayed result mismatch: first=%+v second=%+v", first, second)
	}
}

func TestGormIdempotencyStore_ConcurrentClaimIsRejected(t *testing.T) {
	db := dbtest.NewDB(t, &ActionExecution{})
	store := NewGormIdempotencyStore(db, time.Minute)
	action := idempotentAction("stock-alert:sku-2")
	claim, err := store.Claim(context.Background(), action)
	if err != nil || !claim.Execute {
		t.Fatalf("first claim = %+v, %v", claim, err)
	}
	if _, err := store.Claim(context.Background(), action); !errors.Is(err, ErrIdempotencyInProgress) {
		t.Fatalf("second claim error = %v, want ErrIdempotencyInProgress", err)
	}
}

func TestDispatchSafe_IdempotentFailureCanRetry(t *testing.T) {
	db := dbtest.NewDB(t, &ActionExecution{})
	d := NewDispatcher(dbtest.NewLogger(t), WithIdempotencyStore(NewGormIdempotencyStore(db, time.Minute)))
	var calls atomic.Int32
	d.Register("stock_alert", func(context.Context, map[string]interface{}) (*Result, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("temporary failure")
		}
		return &Result{Success: true, BusinessID: "alert-retried"}, nil
	})
	action := idempotentAction("stock-alert:sku-3")
	first, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil || first.Success || first.ErrorMessage != "temporary failure" {
		t.Fatalf("first result = %+v, err=%v", first, err)
	}
	second, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil || second.BusinessID != "alert-retried" || calls.Load() != 2 {
		t.Fatalf("retry result = %+v, calls=%d, err=%v", second, calls.Load(), err)
	}
}

func TestGormIdempotencyStore_RejectsKeyReuseForDifferentAction(t *testing.T) {
	db := dbtest.NewDB(t, &ActionExecution{})
	store := NewGormIdempotencyStore(db, time.Minute)
	first := idempotentAction("shared-key")
	if _, err := store.Claim(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.ActionType = "different_action"
	if _, err := store.Claim(context.Background(), second); err == nil {
		t.Fatal("reused key for another action was accepted")
	}
}

func TestGormIdempotencyStore_ReclaimsExpiredLease(t *testing.T) {
	db := dbtest.NewDB(t, &ActionExecution{})
	store := NewGormIdempotencyStore(db, time.Minute)
	now := time.Now()
	store.now = func() time.Time { return now }
	action := idempotentAction("expired-claim")
	if _, err := store.Claim(context.Background(), action); err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return now.Add(2 * time.Minute) }
	claim, err := store.Claim(context.Background(), action)
	if err != nil || !claim.Execute {
		t.Fatalf("expired claim was not reclaimed: claim=%+v err=%v", claim, err)
	}
}
