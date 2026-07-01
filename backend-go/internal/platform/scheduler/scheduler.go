// Package scheduler provides a lightweight task scheduler that periodically
// publishes events to the event bus, triggering agent runs on a fixed interval.
//
// Scheduler creates one goroutine per registered task. Each goroutine uses
// time.NewTicker and publishes a "scheduler.tick.{agent_id}" event on each tick.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
)

var (
	schedulerTickDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "multisell_scheduler_tick_duration_seconds",
			Help:    "Duration of scheduler tick execution.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"agent_id", "decision_point"},
	)

	schedulerTickErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "multisell_scheduler_tick_errors_total",
		Help: "Total number of scheduler tick publish errors.",
	})

	schedulerTicksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multisell_scheduler_ticks_total",
			Help: "Total number of scheduler ticks by agent and decision point.",
		},
		[]string{"agent_id", "decision_point"},
	)
)

// TaskRunState captures the live runtime state of a scheduled task.
type TaskRunState struct {
	ID               string     `json:"id"`
	AgentID          string     `json:"agent_id"`
	Interval         string     `json:"interval"`
	LastTickAt       *time.Time `json:"last_tick_at"`
	LastTickDuration *string    `json:"last_tick_duration"`
	CumulativeTicks  int64      `json:"cumulative_ticks"`
	CumulativeSkips  int64      `json:"cumulative_skips"`
	Running          bool       `json:"running"`
}

// Task describes a scheduled agent run.
type Task struct {
	ID            string
	AgentID       string
	DecisionPoint string
	Interval      time.Duration
	Description   string
}

// Scheduler manages periodic task execution.
type Scheduler struct {
	bus     *eventbus.Bus
	tasks   []Task
	logger  *zap.Logger
	wg      sync.WaitGroup
	cancels []context.CancelFunc
	mu      sync.Mutex
	running bool
	guards  sync.Map // key=task.ID, value=*sync.Mutex

	// Runtime tracking per task.
	lastTickAt       sync.Map // key=task.ID, value=*time.Time
	lastTickDuration sync.Map // key=task.ID, value=time.Duration
	cumulativeTicks  sync.Map // key=task.ID, value=int64
	cumulativeSkips  sync.Map // key=task.ID, value=int64
}

// New creates a new scheduler. The bus is used to publish tick events.
func New(bus *eventbus.Bus, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		bus:    bus,
		logger: logger,
	}
}

// Register adds a task to the scheduler. If the scheduler is already running,
// the task starts immediately.
func (s *Scheduler) Register(task Task) {
	s.mu.Lock()
	s.tasks = append(s.tasks, task)
	s.mu.Unlock()

	s.guards.Store(task.ID, &sync.Mutex{})

	s.logger.Info("scheduler task registered",
		zap.String("agent_id", task.AgentID),
		zap.String("decision_point", task.DecisionPoint),
		zap.Duration("interval", task.Interval),
		zap.String("desc", task.Description))

	// If already running, start this task immediately.
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if running {
		ctx, cancel := context.WithCancel(context.Background())
		s.mu.Lock()
		s.cancels = append(s.cancels, cancel)
		s.mu.Unlock()
		s.wg.Add(1)
		go s.runTask(ctx, task)
	}
}

// Start begins executing all registered tasks. Blocks on ctx.Done for shutdown.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.logger.Warn("scheduler already running")
		return
	}
	s.running = true
	s.cancels = make([]context.CancelFunc, 0, len(s.tasks))
	s.mu.Unlock()

	s.logger.Info("scheduler starting",
		zap.Int("tasks", len(s.tasks)))

	// Start each task in its own goroutine.
	s.mu.Lock()
	for _, task := range s.tasks {
		taskCtx, cancel := context.WithCancel(ctx)
		s.cancels = append(s.cancels, cancel)
		s.wg.Add(1)
		go s.runTask(taskCtx, task)
	}
	s.mu.Unlock()

	// Wait for shutdown signal.
	<-ctx.Done()
	s.logger.Info("scheduler shutting down")
	s.Shutdown()
}

// runTask executes a single task on a ticker loop.
func (s *Scheduler) runTask(ctx context.Context, task Task) {
	defer s.wg.Done()

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	s.logger.Debug("task loop started",
		zap.String("agent_id", task.AgentID),
		zap.Duration("interval", task.Interval))

	// Initialize counters.
	s.cumulativeTicks.Store(task.ID, int64(0))
	s.cumulativeSkips.Store(task.ID, int64(0))

	for {
		select {
		case <-ticker.C:
			guard, _ := s.guards.LoadOrStore(task.ID, &sync.Mutex{})
			mu := guard.(*sync.Mutex)
			if mu.TryLock() {
				start := time.Now()
				s.emitTick(ctx, task)
				elapsed := time.Since(start)
				schedulerTickDuration.WithLabelValues(task.AgentID, task.DecisionPoint).Observe(elapsed.Seconds())
				s.lastTickAt.Store(task.ID, &start)
				s.lastTickDuration.Store(task.ID, elapsed)
				s.incCumulativeTicks(task.ID)
				mu.Unlock()
			} else {
				s.logger.Debug("skipping tick — previous run still in progress",
					zap.String("agent_id", task.AgentID))
				s.incCumulativeSkips(task.ID)
			}
		case <-ctx.Done():
			s.logger.Debug("task loop stopped",
				zap.String("agent_id", task.AgentID))
			return
		}
	}
}

// emitTick publishes a scheduler tick event to the event bus.
func (s *Scheduler) emitTick(ctx context.Context, task Task) {
	payload := map[string]interface{}{
		"agent_id":       task.AgentID,
		"decision_point": task.DecisionPoint,
		"schedule_id":    task.ID,
		"description":    task.Description,
	}
	tickID := fmt.Sprintf("sched-tick-%s-%d", task.AgentID, time.Now().UnixMilli())
	ctx = eventbus.WithCorrelationID(ctx, tickID)
	_, err := s.bus.Publish(ctx,
		"scheduler.tick."+task.AgentID,
		"scheduler",
		payload,
	)
	if err != nil {
		s.logger.Warn("scheduler tick publish failed",
			zap.String("agent_id", task.AgentID),
			zap.Error(err))
		schedulerTickErrors.Inc()
	}

	schedulerTicksTotal.WithLabelValues(task.AgentID, task.DecisionPoint).Inc()
}

// Shutdown stops all running tasks gracefully.
func (s *Scheduler) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	// Cancel all task contexts.
	for _, cancel := range s.cancels {
		cancel()
	}
	s.wg.Wait()
	s.running = false
	s.logger.Info("scheduler stopped")
}

// RegisteredTasks returns a copy of all registered tasks.
func (s *Scheduler) RegisteredTasks() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Task, len(s.tasks))
	copy(out, s.tasks)
	return out
}

// TaskRunState returns the live runtime state of all registered tasks.
func (s *Scheduler) TaskRunState() []TaskRunState {
	s.mu.Lock()
	tasks := make([]Task, len(s.tasks))
	copy(tasks, s.tasks)
	s.mu.Unlock()

	states := make([]TaskRunState, 0, len(tasks))
	for _, t := range tasks {
		state := TaskRunState{
			ID:      t.ID,
			AgentID: t.AgentID,
			Interval: t.Interval.String(),
		}

		if v, ok := s.lastTickAt.Load(t.ID); ok {
			if t, ok := v.(*time.Time); ok {
				state.LastTickAt = t
			}
		}
		if v, ok := s.lastTickDuration.Load(t.ID); ok {
			if d, ok := v.(time.Duration); ok {
				s := d.String()
				state.LastTickDuration = &s
			}
		}
		if v, ok := s.cumulativeTicks.Load(t.ID); ok {
			if c, ok := v.(int64); ok {
				state.CumulativeTicks = c
			}
		}
		if v, ok := s.cumulativeSkips.Load(t.ID); ok {
			if c, ok := v.(int64); ok {
				state.CumulativeSkips = c
			}
		}

		// Check if currently running via guards mutex TryLock.
		if guard, ok := s.guards.Load(t.ID); ok {
			mu := guard.(*sync.Mutex)
			state.Running = !mu.TryLock()
			if !state.Running {
				mu.Unlock()
			}
		}

		states = append(states, state)
	}
	return states
}

// incCumulativeTicks atomically increments the cumulative tick count for a task.
func (s *Scheduler) incCumulativeTicks(taskID string) {
	val, _ := s.cumulativeTicks.LoadOrStore(taskID, int64(0))
	s.cumulativeTicks.Store(taskID, val.(int64)+1)
}

// incCumulativeSkips atomically increments the cumulative skip count for a task.
func (s *Scheduler) incCumulativeSkips(taskID string) {
	val, _ := s.cumulativeSkips.LoadOrStore(taskID, int64(0))
	s.cumulativeSkips.Store(taskID, val.(int64)+1)
}
