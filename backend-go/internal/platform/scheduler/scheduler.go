// Package scheduler provides a lightweight task scheduler that periodically
// publishes events to the event bus, triggering agent runs on a fixed interval.
//
// Scheduler creates one goroutine per registered task. Each goroutine uses
// time.NewTicker and publishes a "scheduler.tick.{agent_id}" event on each tick.
//
// On publish failure, the scheduler retries with exponential backoff
// (default: 500ms, 1s, 2s — configurable via RetryConfig).
package scheduler

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/google/uuid"
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
		Help: "Total number of scheduler tick publish errors (final, after retries).",
	})

	schedulerTicksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "multisell_scheduler_ticks_total",
			Help: "Total number of scheduler ticks by agent and decision point.",
		},
		[]string{"agent_id", "decision_point"},
	)

	// defaultRetryBackoff is the default exponential backoff sequence (attempt 0..N-1).
	defaultRetryBackoff = []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second}
)

// RetryConfig controls retry behavior for tick publish failures.
// Zero value uses defaults (3 attempts, 500ms/1s/2s backoff).
type RetryConfig struct {
	// MaxAttempts is the maximum number of publish attempts including the
	// initial attempt. Must be >= 1. Default: 3.
	MaxAttempts int

	// Backoff is the per-attempt delay before the next retry, indexed by
	// failed attempt number (0 = first failure). The last element is used
	// for any additional attempts beyond the defined slice. Default:
	// [500ms, 1s, 2s].
	Backoff []time.Duration

	// ConsecutiveErrorThreshold is the number of consecutive failures after
	// which the log level escalates from Warn to Error. 0 = always Error on
	// final failure. Default: 3.
	ConsecutiveErrorThreshold int
}

// safe returns a usable RetryConfig, replacing zero fields with defaults.
func (c *RetryConfig) safe() RetryConfig {
	out := *c
	if out.MaxAttempts < 1 {
		out.MaxAttempts = 3
	}
	if len(out.Backoff) == 0 {
		out.Backoff = defaultRetryBackoff
	}
	if out.ConsecutiveErrorThreshold <= 0 {
		out.ConsecutiveErrorThreshold = 3
	}
	return out
}

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

// RetryEntry represents a failed scheduler tick awaiting retry.
type RetryEntry struct {
	ID            string                 `json:"id"`
	TaskID        string                 `json:"task_id"`
	AgentID       string                 `json:"agent_id"`
	DecisionPoint string                 `json:"decision_point"`
	FailedAt      time.Time              `json:"failed_at"`
	LastError     string                 `json:"last_error"`
	Attempts      int                    `json:"attempts"`
	Payload       map[string]interface{} `json:"payload"`
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

	// retry config — defaults used if zero.
	retry RetryConfig

	// Runtime tracking per task.
	lastTickAt        sync.Map // key=task.ID, value=*time.Time
	lastTickDuration  sync.Map // key=task.ID, value=time.Duration
	cumulativeTicks   sync.Map // key=task.ID, value=int64
	cumulativeSkips   sync.Map // key=task.ID, value=int64
	consecutiveErrors sync.Map // key=task.ID, value=int64

	// Retry queue for failed ticks.
	retryMu             sync.Mutex
	retries             []RetryEntry
	retryStore          RetryStore
	leaderLease         LeaderLease
	leaderRetryInterval time.Duration
}

// New creates a new scheduler. The bus is used to publish tick events.
func New(bus *eventbus.Bus, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		bus:    bus,
		logger: logger,
	}
}

// WithRetryConfig sets the retry configuration for publish failures.
func (s *Scheduler) WithRetryConfig(cfg RetryConfig) *Scheduler {
	s.retry = cfg
	return s
}

func (s *Scheduler) WithRetryStore(store RetryStore) *Scheduler {
	s.retryStore = store
	return s
}

func (s *Scheduler) WithLeaderLease(lease LeaderLease) *Scheduler {
	s.leaderLease = lease
	return s
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
	releaseLeader, acquired := s.acquireLeader(ctx)
	if !acquired {
		return
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := releaseLeader(releaseCtx); err != nil {
			s.logger.Error("scheduler leader lease release failed", zap.Error(err))
		}
	}()
	var recovered []RetryEntry
	if s.retryStore != nil {
		entries, err := s.retryStore.List(ctx)
		if err != nil {
			s.logger.Error("scheduler retry recovery failed", zap.Error(err))
			return
		}
		recovered = entries
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.logger.Warn("scheduler already running")
		return
	}
	s.running = true
	s.cancels = make([]context.CancelFunc, 0, len(s.tasks))
	s.mu.Unlock()
	if s.retryStore != nil {
		s.retryMu.Lock()
		s.retries = recovered
		s.retryMu.Unlock()
	}

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
	// Start retry loop for failed ticks.
	retryCtx, retryCancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancels = append(s.cancels, retryCancel)
	s.mu.Unlock()
	s.wg.Add(1)
	go s.retryLoop(retryCtx)
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
				ok := s.emitTick(ctx, task)
				elapsed := time.Since(start)
				schedulerTickDuration.WithLabelValues(task.AgentID, task.DecisionPoint).Observe(elapsed.Seconds())
				s.lastTickAt.Store(task.ID, &start)
				s.lastTickDuration.Store(task.ID, elapsed)
				s.incCumulativeTicks(task.ID)
				if ok {
					s.consecutiveErrors.Store(task.ID, int64(0))
				}
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
// Returns true if the publish succeeded (after any retries).
func (s *Scheduler) emitTick(ctx context.Context, task Task) bool {
	payload := map[string]interface{}{
		"agent_id":       task.AgentID,
		"decision_point": task.DecisionPoint,
		"schedule_id":    task.ID,
		"description":    task.Description,
	}
	tickID := fmt.Sprintf("sched-tick-%s-%d", task.AgentID, time.Now().UnixMilli())
	ctx = eventbus.WithCorrelationID(ctx, tickID)
	cfg := s.retry.safe()

	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff before retry. Index into Backoff at
			// attempt-1 (the failed attempt index), clamping to the last
			// defined value for any excess.
			backoffIdx := attempt - 1
			if backoffIdx >= len(cfg.Backoff) {
				backoffIdx = len(cfg.Backoff) - 1
			}
			delay := cfg.Backoff[backoffIdx]

			// Context-aware sleep — return early on shutdown.
			select {
			case <-ctx.Done():
				s.logger.Warn("scheduler tick retry cancelled by shutdown",
					zap.String("agent_id", task.AgentID),
					zap.Int("attempt", attempt+1),
					zap.Duration("slept", 0))
				return false
			case <-time.After(delay):
			}
		}

		_, lastErr = s.bus.Publish(ctx,
			"scheduler.tick."+task.AgentID,
			"scheduler",
			payload,
		)
		if lastErr == nil {
			schedulerTicksTotal.WithLabelValues(task.AgentID, task.DecisionPoint).Inc()
			return true
		}

		// Failed this attempt. Log at Warn for early failures, escalate
		// to Error when consecutive error count exceeds threshold.
		raw, _ := s.consecutiveErrors.LoadOrStore(task.ID, int64(0))
		consecutive := raw.(int64) + 1
		s.consecutiveErrors.Store(task.ID, consecutive)

		isLast := attempt == cfg.MaxAttempts-1
		if isLast || consecutive >= int64(cfg.ConsecutiveErrorThreshold) {
			s.logger.Error("scheduler tick publish failed after retries",
				zap.String("agent_id", task.AgentID),
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", cfg.MaxAttempts),
				zap.Int64("consecutive_errors", consecutive),
				zap.Error(lastErr))
		} else {
			s.logger.Warn("scheduler tick publish failed, retrying",
				zap.String("agent_id", task.AgentID),
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", cfg.MaxAttempts),
				zap.Int64("consecutive_errors", consecutive),
				zap.Duration("next_retry_in", cfg.Backoff[int(math.Min(float64(attempt), float64(len(cfg.Backoff)-1)))]),
				zap.Error(lastErr))
		}
	}

	// All retries exhausted — push to retry queue for later recovery.
	s.pushRetry(task, payload, lastErr.Error())

	schedulerTickErrors.Inc()
	return false
}

// Shutdown stops all running tasks gracefully.
func (s *Scheduler) Shutdown() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	// Mark not-running before waiting so readiness fails immediately. Copy the
	// callbacks and release the lock because worker shutdown paths also inspect
	// scheduler state.
	s.running = false
	cancels := append([]context.CancelFunc(nil), s.cancels...)
	s.cancels = nil
	s.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	s.wg.Wait()
	s.logger.Info("scheduler stopped")
}

// IsRunning reports whether scheduler task loops are active.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
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
			ID:       t.ID,
			AgentID:  t.AgentID,
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

// pushRetry adds a failed tick to the retry queue.
func (s *Scheduler) pushRetry(task Task, payload map[string]interface{}, errMsg string) {
	entry := RetryEntry{
		ID: uuid.NewString(), TaskID: task.ID, AgentID: task.AgentID,
		DecisionPoint: task.DecisionPoint, FailedAt: time.Now(), LastError: errMsg,
		Attempts: 1, Payload: payload,
	}
	if s.retryStore != nil {
		if err := s.retryStore.Save(context.Background(), entry); err != nil {
			s.logger.Error("scheduler retry persistence failed", zap.String("task_id", task.ID), zap.Error(err))
			return
		}
	}
	s.retryMu.Lock()
	defer s.retryMu.Unlock()
	// Cap the queue at 100 entries (oldest dropped).
	if len(s.retries) >= 100 {
		s.retries = s.retries[len(s.retries)-99:]
	}
	s.retries = append(s.retries, entry)
}

// RetryQueue returns a copy of the current retry queue entries.
func (s *Scheduler) RetryQueue() []RetryEntry {
	s.retryMu.Lock()
	defer s.retryMu.Unlock()
	out := make([]RetryEntry, len(s.retries))
	copy(out, s.retries)
	return out
}

// retryLoop runs as a background goroutine, retrying failed ticks every 30s.
// Max 3 attempts per entry; entries exhausted beyond that are dropped.
func (s *Scheduler) retryLoop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			running := s.running
			s.mu.Unlock()
			if !running {
				return
			}
			s.retryMu.Lock()
			entries := append([]RetryEntry(nil), s.retries...)
			s.retryMu.Unlock()
			var kept []RetryEntry
			for _, entry := range entries {
				if entry.Attempts >= 3 {
					s.logger.Warn("retry exhausted, dropping tick",
						zap.String("agent_id", entry.AgentID),
						zap.String("task_id", entry.TaskID),
						zap.Int("attempts", entry.Attempts))
					if s.retryStore != nil {
						if err := s.retryStore.Delete(ctx, entry.ID); err != nil {
							s.logger.Error("scheduler retry delete failed", zap.Error(err))
							kept = append(kept, entry)
						}
					}
					continue
				}
				ctx2 := context.Background()
				ctx2 = eventbus.WithCorrelationID(ctx2, "retry-"+entry.TaskID+"-"+fmt.Sprintf("%d", time.Now().UnixMilli()))
				_, err := s.bus.Publish(ctx2, "scheduler.tick."+entry.AgentID, "scheduler", entry.Payload)
				if err != nil {
					entry.Attempts++
					entry.LastError = err.Error()
					if s.retryStore != nil {
						if persistErr := s.retryStore.Update(ctx, entry); persistErr != nil {
							s.logger.Error("scheduler retry update failed", zap.Error(persistErr))
						}
					}
					kept = append(kept, entry)
					schedulerTickErrors.Inc()
				} else {
					if s.retryStore != nil {
						if deleteErr := s.retryStore.Delete(ctx, entry.ID); deleteErr != nil {
							s.logger.Error("scheduler retry delete failed", zap.Error(deleteErr))
							kept = append(kept, entry)
							continue
						}
					}
					s.logger.Info("retry succeeded",
						zap.String("agent_id", entry.AgentID),
						zap.String("task_id", entry.TaskID))
				}
			}
			s.retryMu.Lock()
			s.retries = kept
			s.retryMu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
