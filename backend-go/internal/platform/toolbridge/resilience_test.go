package toolbridge

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecuteResilientOpensAndHalfOpensCircuit(t *testing.T) {
	tracker := NewExternalCallTracker(2)
	tracker.cooldown = 5 * time.Millisecond
	var calls atomic.Int32
	failing := func(context.Context) (string, error) {
		calls.Add(1)
		return "", errors.New("provider down")
	}
	if _, err := executeResilient(context.Background(), "provider", 2, 0, tracker, failing); err == nil {
		t.Fatal("provider failure was hidden")
	}
	if _, err := executeResilient(context.Background(), "provider", 1, 0, tracker, failing); !errors.Is(err, DegradedErr) {
		t.Fatalf("open circuit error=%v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("open circuit reached provider, calls=%d", calls.Load())
	}
	time.Sleep(6 * time.Millisecond)
	result, err := executeResilient(context.Background(), "provider", 1, 0, tracker, func(context.Context) (string, error) {
		calls.Add(1)
		return "recovered", nil
	})
	if err != nil || result != "recovered" || tracker.IsDegraded("provider") {
		t.Fatalf("half-open result=%q err=%v degraded=%v", result, err, tracker.IsDegraded("provider"))
	}
}

func TestExecuteResilientCancellationStopsBackoff(t *testing.T) {
	tracker := NewExternalCallTracker(10)
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	_, err := executeResilient(ctx, "cancel-provider", 5, time.Second, tracker, func(context.Context) (string, error) {
		calls.Add(1)
		cancel()
		return "", errors.New("temporary")
	})
	if !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}
