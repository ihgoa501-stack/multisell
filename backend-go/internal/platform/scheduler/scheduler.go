// Package scheduler provides a lightweight task scheduler that periodically
// publishes events to the event bus, triggering agent runs on a fixed interval.
//
// Scheduler creates one goroutine per registered task. Each goroutine uses
// time.NewTicker and publishes a "scheduler.tick.{agent_id}" event on each tick.
package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
)

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

	for {
		select {
		case <-ticker.C:
			guard, _ := s.guards.LoadOrStore(task.ID, &sync.Mutex{})
			mu := guard.(*sync.Mutex)
			if mu.TryLock() {
				s.emitTick(task)
				mu.Unlock()
			} else {
				s.logger.Debug("skipping tick — previous run still in progress",
					zap.String("agent_id", task.AgentID))
			}
		case <-ctx.Done():
			s.logger.Debug("task loop stopped",
				zap.String("agent_id", task.AgentID))
			return
		}
	}
}

// emitTick publishes a scheduler tick event to the event bus.
func (s *Scheduler) emitTick(task Task) {
	payload := map[string]interface{}{
		"agent_id":       task.AgentID,
		"decision_point": task.DecisionPoint,
		"schedule_id":    task.ID,
		"description":    task.Description,
	}
	_, err := s.bus.Publish(context.Background(),
		"scheduler.tick."+task.AgentID,
		"scheduler",
		payload,
	)
	if err != nil {
		s.logger.Warn("scheduler tick publish failed",
			zap.String("agent_id", task.AgentID),
			zap.Error(err))
	}
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
