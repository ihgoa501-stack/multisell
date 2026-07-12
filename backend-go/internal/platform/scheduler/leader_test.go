package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

type fakeLeaderLease struct {
	attempts     atomic.Int32
	acquireAfter int32
	released     atomic.Bool
}

func (l *fakeLeaderLease) TryAcquire(context.Context) (func(context.Context) error, bool, error) {
	if l.attempts.Add(1) < l.acquireAfter {
		return nil, false, nil
	}
	return func(context.Context) error { l.released.Store(true); return nil }, true, nil
}

func TestSchedulerWaitsForLeaderLeaseBeforeRunning(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	busCtx, stopBus := context.WithCancel(context.Background())
	bus.Start(busCtx)
	t.Cleanup(func() { stopBus(); bus.Stop() })
	lease := &fakeLeaderLease{acquireAfter: 3}
	s := New(bus, logger).WithLeaderLease(lease)
	s.leaderRetryInterval = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Start(ctx); close(done) }()
	deadline := time.Now().Add(time.Second)
	for !s.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !s.IsRunning() || lease.attempts.Load() < 3 {
		t.Fatalf("running=%v attempts=%d", s.IsRunning(), lease.attempts.Load())
	}
	cancel()
	<-done
	if !lease.released.Load() {
		t.Fatal("leader lease was not released")
	}
}

func TestSchedulerStandbyCancellationNeverRuns(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	lease := &fakeLeaderLease{acquireAfter: 1000}
	s := New(bus, logger).WithLeaderLease(lease)
	s.leaderRetryInterval = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	s.Start(ctx)
	if s.IsRunning() || lease.released.Load() {
		t.Fatalf("standby running=%v released=%v", s.IsRunning(), lease.released.Load())
	}
}
