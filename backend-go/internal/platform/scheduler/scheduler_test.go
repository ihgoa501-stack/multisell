package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

// ---------------------------------------------------------------------------
// Task registration
// ---------------------------------------------------------------------------

func TestRegisterTask(t *testing.T) {
	s, _ := newTestScheduler(t)

	task := Task{
		ID:            "test-1",
		AgentID:       "A1",
		DecisionPoint: "check",
		Interval:      time.Hour,
		Description:   "Test task",
	}
	s.Register(task)

	tasks := s.RegisteredTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 registered task, got %d", len(tasks))
	}
	if tasks[0].ID != "test-1" {
		t.Errorf("expected task ID test-1, got %s", tasks[0].ID)
	}
	if tasks[0].AgentID != "A1" {
		t.Errorf("expected AgentID A1, got %s", tasks[0].AgentID)
	}
	if tasks[0].DecisionPoint != "check" {
		t.Errorf("expected DecisionPoint check, got %s", tasks[0].DecisionPoint)
	}
	if tasks[0].Interval != time.Hour {
		t.Errorf("expected Interval 1h, got %s", tasks[0].Interval)
	}
}

func TestRegisterMultipleTasks(t *testing.T) {
	s, _ := newTestScheduler(t)

	s.Register(Task{ID: "t1", AgentID: "A1", DecisionPoint: "dp1", Interval: time.Hour})
	s.Register(Task{ID: "t2", AgentID: "A2", DecisionPoint: "dp2", Interval: 30 * time.Minute})
	s.Register(Task{ID: "t3", AgentID: "A3", DecisionPoint: "dp3", Interval: 5 * time.Minute})

	tasks := s.RegisteredTasks()
	if len(tasks) != 3 {
		t.Fatalf("expected 3 registered tasks, got %d", len(tasks))
	}

	ids := make(map[string]bool)
	for _, task := range tasks {
		ids[task.ID] = true
	}
	if !ids["t1"] || !ids["t2"] || !ids["t3"] {
		t.Errorf("missing registered tasks, got IDs: %v", ids)
	}
}

func TestRegisteredTasksReturnsCopy(t *testing.T) {
	s, _ := newTestScheduler(t)

	s.Register(Task{ID: "original", AgentID: "A1", Interval: time.Hour})

	tasks := s.RegisteredTasks()
	tasks[0].ID = "mutated"

	// Verify original is unchanged.
	again := s.RegisteredTasks()
	if again[0].ID != "original" {
		t.Errorf("expected original task ID to be preserved, got %s", again[0].ID)
	}
}

// ---------------------------------------------------------------------------
// Tick publishing
// ---------------------------------------------------------------------------

func TestSchedulerTickPublishing(t *testing.T) {
	s, bus := newTestScheduler(t)

	var tickCount atomic.Int32
	bus.Subscribe("scheduler.tick.A1", func(_ context.Context, evt eventbus.Event) error {
		tickCount.Add(1)
		if evt.Payload == nil {
			t.Error("expected non-nil payload")
		}
		if aid, ok := evt.Payload["agent_id"]; !ok || aid != "A1" {
			t.Errorf("expected payload agent_id=A1, got %v", aid)
		}
		return nil
	})

	s.Register(Task{
		ID:            "tick-test",
		AgentID:       "A1",
		DecisionPoint: "check",
		Interval:      10 * time.Millisecond,
		Description:   "Tick test",
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := startAndWaitDone(t, s, ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if n := tickCount.Load(); n < 1 {
		t.Errorf("expected at least 1 tick, got %d", n)
	}
}

func TestMultipleTasksReceiveTicks(t *testing.T) {
	s, bus := newTestScheduler(t)

	var countA atomic.Int32
	var countB atomic.Int32

	bus.Subscribe("scheduler.tick.M1", func(_ context.Context, _ eventbus.Event) error {
		countA.Add(1)
		return nil
	})
	bus.Subscribe("scheduler.tick.M2", func(_ context.Context, _ eventbus.Event) error {
		countB.Add(1)
		return nil
	})

	s.Register(Task{ID: "m1", AgentID: "M1", DecisionPoint: "dp1", Interval: 10 * time.Millisecond})
	s.Register(Task{ID: "m2", AgentID: "M2", DecisionPoint: "dp2", Interval: 10 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	done := startAndWaitDone(t, s, ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if n := countA.Load(); n < 1 {
		t.Errorf("expected at least 1 tick for M1, got %d", n)
	}
	if n := countB.Load(); n < 1 {
		t.Errorf("expected at least 1 tick for M2, got %d", n)
	}
}

// ---------------------------------------------------------------------------
// Shutdown behavior
// ---------------------------------------------------------------------------

func TestSchedulerShutdownStopsTicks(t *testing.T) {
	s, bus := newTestScheduler(t)

	var tickCount atomic.Int32
	bus.Subscribe("scheduler.tick.B1", func(_ context.Context, _ eventbus.Event) error {
		tickCount.Add(1)
		return nil
	})

	s.Register(Task{
		ID:            "shutdown-test",
		AgentID:       "B1",
		DecisionPoint: "check",
		Interval:      10 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := startAndWaitDone(t, s, ctx)
	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done // Start has finished (including Shutdown)

	afterShutdown := tickCount.Load()
	time.Sleep(50 * time.Millisecond)
	final := tickCount.Load()

	if final != afterShutdown {
		t.Errorf("ticks increased after shutdown: before=%d, after=%d", afterShutdown, final)
	}
}

func TestShutdownWithoutStart(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)

	s := New(bus, logger)
	s.Shutdown()
	// Should not panic or hang.
}

// ---------------------------------------------------------------------------
// Register after Start (auto-start)
// ---------------------------------------------------------------------------

func TestRegisterAfterStart(t *testing.T) {
	s, bus := newTestScheduler(t)

	// Start with one task.
	s.Register(Task{ID: "pre", AgentID: "D1", DecisionPoint: "pre", Interval: time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	done := startAndWaitDone(t, s, ctx)
	time.Sleep(10 * time.Millisecond)

	// Register another task after Start — should auto-start.
	var tickCount atomic.Int32
	bus.Subscribe("scheduler.tick.D2", func(_ context.Context, _ eventbus.Event) error {
		tickCount.Add(1)
		return nil
	})

	s.Register(Task{ID: "post", AgentID: "D2", DecisionPoint: "post", Interval: 10 * time.Millisecond})

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if n := tickCount.Load(); n < 1 {
		t.Errorf("expected at least 1 tick for post-start task, got %d", n)
	}
}

// ---------------------------------------------------------------------------
// Start twice
// ---------------------------------------------------------------------------

func TestStartTwiceDoesNotPanic(t *testing.T) {
	s, bus := newTestScheduler(t)
	_ = bus // bus is cleaned up by newTestScheduler

	ctx1, cancel1 := context.WithCancel(context.Background())
	done1 := startAndWaitDone(t, s, ctx1)
	time.Sleep(10 * time.Millisecond)

	// Starting again should not panic.
	ctx2, cancel2 := context.WithCancel(context.Background())
	go s.Start(ctx2)
	time.Sleep(10 * time.Millisecond)

	cancel1()
	<-done1
	cancel2()
	s.Shutdown()
}

// ---------------------------------------------------------------------------
// TaskRunState
// ---------------------------------------------------------------------------

func TestTaskRunStateAfterNoTicks(t *testing.T) {
	s, _ := newTestScheduler(t)

	s.Register(Task{
		ID: "idle", AgentID: "X1", DecisionPoint: "idle",
		Interval: time.Hour, Description: "Idle task",
	})

	states := s.TaskRunState()
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	state := states[0]
	if state.ID != "idle" {
		t.Errorf("expected ID idle, got %s", state.ID)
	}
	if state.AgentID != "X1" {
		t.Errorf("expected AgentID X1, got %s", state.AgentID)
	}
	if state.Interval != "1h0m0s" {
		t.Errorf("expected Interval 1h0m0s, got %s", state.Interval)
	}
	if state.CumulativeTicks != 0 {
		t.Errorf("expected 0 cumulative ticks before start, got %d", state.CumulativeTicks)
	}
}

func TestTaskRunStateAfterTicks(t *testing.T) {
	s, _ := newTestScheduler(t)

	s.Register(Task{
		ID: "state-test", AgentID: "C1", DecisionPoint: "check",
		Interval: 10 * time.Millisecond, Description: "State test",
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := startAndWaitDone(t, s, ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	states := s.TaskRunState()
	if len(states) != 1 {
		t.Fatalf("expected 1 task run state, got %d", len(states))
	}

	state := states[0]
	if state.CumulativeTicks < 1 {
		t.Errorf("expected at least 1 cumulative tick, got %d", state.CumulativeTicks)
	}
	if state.LastTickAt == nil {
		t.Error("expected LastTickAt to be set")
	}
	if state.LastTickDuration == nil {
		t.Error("expected LastTickDuration to be set")
	}
	if state.Running {
		t.Error("expected Running=false after shutdown")
	}
}

func TestTaskRunStateMultipleTasks(t *testing.T) {
	s, _ := newTestScheduler(t)

	s.Register(Task{ID: "fast", AgentID: "F1", DecisionPoint: "dp1", Interval: 10 * time.Millisecond})
	s.Register(Task{ID: "slow", AgentID: "F2", DecisionPoint: "dp2", Interval: time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	done := startAndWaitDone(t, s, ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	states := s.TaskRunState()
	if len(states) != 2 {
		t.Fatalf("expected 2 task run states, got %d", len(states))
	}

	// The fast task should have ticks.
	for _, st := range states {
		if st.ID == "fast" && st.CumulativeTicks < 1 {
			t.Errorf("expected fast task to have ticks, got %d", st.CumulativeTicks)
		}
		// The slow task (1h interval) should NOT have ticks.
		if st.ID == "slow" && st.CumulativeTicks > 0 {
			t.Errorf("expected slow task to have 0 ticks, got %d", st.CumulativeTicks)
		}
	}
}

// ---------------------------------------------------------------------------
// Tick event payload verification
// ---------------------------------------------------------------------------

func TestTickPayloadFields(t *testing.T) {
	s, bus := newTestScheduler(t)

	received := make(chan eventbus.Event, 4)
	bus.Subscribe("scheduler.tick.P1", func(_ context.Context, evt eventbus.Event) error {
		select {
		case received <- evt:
		default:
		}
		return nil
	})

	s.Register(Task{
		ID: "payload-test", AgentID: "P1", DecisionPoint: "daily",
		Interval: 10 * time.Millisecond, Description: "Payload verification",
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := startAndWaitDone(t, s, ctx)
	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done

	select {
	case evt := <-received:
		if evt.Topic != "scheduler.tick.P1" {
			t.Errorf("expected topic scheduler.tick.P1, got %s", evt.Topic)
		}
		if evt.Source != "scheduler" {
			t.Errorf("expected source scheduler, got %s", evt.Source)
		}
		if evt.Payload["schedule_id"] != "payload-test" {
			t.Errorf("expected schedule_id payload-test, got %v", evt.Payload["schedule_id"])
		}
		if evt.Payload["decision_point"] != "daily" {
			t.Errorf("expected decision_point daily, got %v", evt.Payload["decision_point"])
		}
		if evt.Payload["description"] != "Payload verification" {
			t.Errorf("expected description 'Payload verification', got %v", evt.Payload["description"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for tick event")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestScheduler creates a scheduler backed by a real event bus.
// The bus is started and registered for cleanup. The scheduler is NOT started
// (caller must call Start). See startAndWaitDone.
func newTestScheduler(t *testing.T) (*Scheduler, *eventbus.Bus) {
	t.Helper()
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger)
	bus.Start(context.Background())
	t.Cleanup(bus.Stop)
	time.Sleep(10 * time.Millisecond)

	s := New(bus, logger)
	return s, bus
}

// startAndWaitDone runs s.Start in a goroutine and returns a channel that
// is closed when Start returns (including shutdown). The caller must cancel
// the context and then wait on the returned channel.
func startAndWaitDone(t *testing.T, s *Scheduler, ctx context.Context) <-chan struct{} {
	t.Helper()
	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()
	return done
}
