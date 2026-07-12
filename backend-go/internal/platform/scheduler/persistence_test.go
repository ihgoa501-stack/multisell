package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

func TestGormRetryStoreLifecycle(t *testing.T) {
	db := dbtest.NewDB(t, &RetryRecord{})
	store := NewGormRetryStore(db)
	entry := RetryEntry{ID: "retry-1", TaskID: "task-1", AgentID: "A1", DecisionPoint: "check", FailedAt: time.Now(), LastError: "down", Attempts: 1, Payload: map[string]interface{}{"key": "value"}}
	if err := store.Save(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List(context.Background())
	if err != nil || len(entries) != 1 || entries[0].Payload["key"] != "value" {
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
	entry.Attempts = 2
	entry.LastError = "still down"
	if err := store.Update(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	entries, _ = store.List(context.Background())
	if entries[0].Attempts != 2 || entries[0].LastError != "still down" {
		t.Fatalf("updated entry=%+v", entries[0])
	}
	if err := store.Delete(context.Background(), entry.ID); err != nil {
		t.Fatal(err)
	}
	entries, _ = store.List(context.Background())
	if len(entries) != 0 {
		t.Fatalf("entries remained after delete: %+v", entries)
	}
}

func TestSchedulerRecoversPersistedRetriesOnStart(t *testing.T) {
	db := dbtest.NewDB(t, &RetryRecord{})
	store := NewGormRetryStore(db)
	entry := RetryEntry{ID: "retry-restart", TaskID: "task-restart", AgentID: "A2", DecisionPoint: "recover", FailedAt: time.Now(), LastError: "restart", Attempts: 1, Payload: map[string]interface{}{"schedule_id": "task-restart"}}
	if err := store.Save(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	busCtx, stopBus := context.WithCancel(context.Background())
	bus.Start(busCtx)
	t.Cleanup(func() { stopBus(); bus.Stop() })
	s := New(bus, logger).WithRetryStore(store)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Start(ctx); close(done) }()
	deadline := time.Now().Add(time.Second)
	for !s.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	entries := s.RetryQueue()
	cancel()
	<-done
	if len(entries) != 1 || entries[0].ID != entry.ID {
		t.Fatalf("recovered entries=%+v", entries)
	}
}
