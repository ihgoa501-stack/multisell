package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Engine is the generic workflow execution engine.
type Engine struct {
	db       *gorm.DB
	bus      *eventbus.Bus
	aiOrch   *ai.Orchestrator
	cmd      *command.Dispatcher
	logger   *zap.Logger

	mu       sync.Mutex
	running  map[int64]context.CancelFunc // cancel running steps per run
}

// NewEngine creates a workflow engine.
func NewEngine(db *gorm.DB, bus *eventbus.Bus, aiOrch *ai.Orchestrator, dispatcher *command.Dispatcher, logger *zap.Logger) *Engine {
	return &Engine{
		db:      db,
		bus:     bus,
		aiOrch:  aiOrch,
		cmd:     dispatcher,
		logger:  logger.Named("workflow"),
		running: make(map[int64]context.CancelFunc),
	}
}

// ── CRUD for workflow definitions ────────────────────────────────────

func (e *Engine) CreateDef(ctx context.Context, def *WorkflowDef) error {
	return e.db.WithContext(ctx).Create(def).Error
}

func (e *Engine) ListDefs(ctx context.Context) ([]WorkflowDef, error) {
	var defs []WorkflowDef
	err := e.db.WithContext(ctx).Order("id ASC").Find(&defs).Error
	return defs, err
}

func (e *Engine) GetDef(ctx context.Context, id int64) (*WorkflowDef, error) {
	var def WorkflowDef
	err := e.db.WithContext(ctx).First(&def, id).Error
	return &def, err
}

func (e *Engine) UpdateDef(ctx context.Context, def *WorkflowDef) error {
	return e.db.WithContext(ctx).Save(def).Error
}

func (e *Engine) DeleteDef(ctx context.Context, id int64) error {
	return e.db.WithContext(ctx).Delete(&WorkflowDef{}, id).Error
}

// ── Run lifecycle ────────────────────────────────────────────────────

// StartRun creates and begins a workflow run.
func (e *Engine) StartRun(ctx context.Context, defID int64, initialCtx map[string]interface{}) (*WorkflowRun, error) {
	def, err := e.GetDef(ctx, defID)
	if err != nil {
		return nil, fmt.Errorf("get workflow def: %w", err)
	}

	ctxJSON := "{}"
	if initialCtx != nil {
		b, _ := json.Marshal(initialCtx)
		ctxJSON = string(b)
	}

	now := time.Now()
	run := &WorkflowRun{
		WorkflowDefID: defID,
		Name:          def.Name,
		Status:        RunStatusRunning,
		Context:       ctxJSON,
		StartedAt:     &now,
	}
	if err := e.db.WithContext(ctx).Create(run).Error; err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	// Parse steps and create initial step runs.
	var steps []StepDef
	if err := json.Unmarshal([]byte(def.Steps), &steps); err != nil {
		return nil, fmt.Errorf("parse steps: %w", err)
	}

	for i, s := range steps {
		stepType := s.Type
		if stepType == "" {
			stepType = StepTypeAgent
		}
		maxAttempts := s.RetryCount + 1
		if maxAttempts < 1 {
			maxAttempts = 1
		}

		sr := WorkflowStepRun{
			WorkflowRunID:  run.ID,
			StepName:       s.Name,
			StepType:       stepType,
			Status:         StepStatusPending,
			Input:          mustJSON(s.Inputs),
			MaxAttempts:    maxAttempts,
			TimeoutSeconds: s.TimeoutSeconds,
		}
		if sr.TimeoutSeconds <= 0 {
			sr.TimeoutSeconds = 300
		}

		// First step starts running immediately.
		if i == 0 {
			sr.Status = StepStatusRunning
			sr.StartedAt = &now
		}
		if err := e.db.WithContext(ctx).Create(&sr).Error; err != nil {
			return nil, fmt.Errorf("create step run: %w", err)
		}
	}

	e.logger.Info("workflow run started", zap.Int64("run_id", run.ID), zap.String("name", def.Name))

	// Execute first step.
	if len(steps) > 0 {
		e.executeStep(ctx, run.ID, steps[0])
	}

	return run, nil
}

// PauseRun pauses a running workflow. The running goroutines are left to
// time out naturally — the pause is effective after the current step finishes.
func (e *Engine) PauseRun(ctx context.Context, runID int64) error {
	return e.db.WithContext(ctx).Model(&WorkflowRun{}).
		Where("id = ?", runID).
		Where("status IN ?", []string{RunStatusRunning, RunStatusPending}).
		Update("status", RunStatusPaused).Error
}

// ResumeRun resumes a paused workflow.
func (e *Engine) ResumeRun(ctx context.Context, runID int64) error {
	var run WorkflowRun
	if err := e.db.WithContext(ctx).First(&run, runID).Error; err != nil {
		return err
	}
	if run.Status != RunStatusPaused {
		return fmt.Errorf("run %d is not paused (status: %s)", runID, run.Status)
	}

	if err := e.db.WithContext(ctx).Model(&run).Update("status", RunStatusRunning).Error; err != nil {
		return err
	}

	// Find the last pending step and resume it.
	var pending StepResult
	e.db.WithContext(ctx).Raw(`
		SELECT step_name, status, attempt
		FROM workflow_step_run
		WHERE workflow_run_id = ?
		ORDER BY id DESC LIMIT 1`, runID).Scan(&pending)
	if pending.Status == StepStatusFailed {
		return e.retryStep(ctx, runID, pending.StepName)
	}
	return nil
}

type StepResult struct {
	StepName string
	Status   string
	Attempt  int
}

// ── Step execution ───────────────────────────────────────────────────

// AdvanceStep is called when an asynchronous step (agent, event) completes.
// Sets step status and advances to the next step.
func (e *Engine) AdvanceStep(ctx context.Context, runID int64, stepName string, output map[string]interface{}, execErr error) error {
	// Find the step run record.
	var sr WorkflowStepRun
	if err := e.db.WithContext(ctx).
		Where("workflow_run_id = ? AND step_name = ?", runID, stepName).
		Order("id DESC").First(&sr).Error; err != nil {
		return fmt.Errorf("find step run: %w", err)
	}

	return e.finalizeStep(ctx, runID, sr, output, execErr)
}

func (e *Engine) executeStep(ctx context.Context, runID int64, def StepDef) {
	e.logger.Info("executing step", zap.Int64("run_id", runID), zap.String("step", def.Name),
		zap.String("type", def.Type))

	ctx, cancel := context.WithTimeout(ctx, time.Duration(def.TimeoutSeconds)*time.Second)
	e.mu.Lock()
	e.running[runID] = cancel
	e.mu.Unlock()

	go func() {
		defer cancel()
		e.mu.Lock()
		delete(e.running, runID)
		e.mu.Unlock()

		var output map[string]interface{}
		var execErr error

		switch def.Type {
		case StepTypeAgent:
			output, execErr = e.execAgent(ctx, runID, def)
		case StepTypeCommand:
			output, execErr = e.execCommand(ctx, runID, def)
		case StepTypeFork:
			output, execErr = e.execFork(ctx, runID, def)
		case StepTypeJoin:
			output, execErr = e.execJoin(ctx, runID, def)
		case StepTypeDelay:
			output, execErr = e.execDelay(ctx, runID, def)
		case StepTypeEvent:
			output, execErr = e.execEvent(ctx, runID, def)
		default:
			execErr = fmt.Errorf("unknown step type: %s", def.Type)
		}

		_ = e.finalizeStep(context.Background(), runID, WorkflowStepRun{StepName: def.Name, WorkflowRunID: runID}, output, execErr)
	}()
}

func (e *Engine) finalizeStep(ctx context.Context, runID int64, sr WorkflowStepRun, output map[string]interface{}, execErr error) error {
	now := time.Now()
	outJSON := mustJSON(output)
	errMsg := ""
	status := StepStatusCompleted

	if execErr != nil {
		errMsg = execErr.Error()
		status = StepStatusFailed

		// Retry if under max attempts.
		if sr.Attempt < sr.MaxAttempts {
			e.db.WithContext(ctx).Model(&WorkflowStepRun{}).
				Where("workflow_run_id = ? AND step_name = ?", runID, sr.StepName).
				Updates(map[string]interface{}{
					"status":       StepStatusFailed,
					"output":       outJSON,
					"error":        errMsg,
					"completed_at": now,
				})

			// Re-run: create a new attempt.
			e.logger.Info("retrying step", zap.Int64("run_id", runID),
				zap.String("step", sr.StepName), zap.Int("attempt", sr.Attempt+1))

			time.Sleep(time.Second) // brief backoff
			return e.retryStep(ctx, runID, sr.StepName)
		}
	}

	tx := e.db.WithContext(ctx)
	if err := tx.Model(&WorkflowStepRun{}).
		Where("workflow_run_id = ? AND step_name = ?", runID, sr.StepName).
		Updates(map[string]interface{}{
			"status":       status,
			"output":       outJSON,
			"error":        errMsg,
			"completed_at": now,
		}).Error; err != nil {
		return fmt.Errorf("update step: %w", err)
	}

	// Publish step completed/failed event.
	if e.bus != nil {
		topic := "workflow.step.completed"
		if status == StepStatusFailed {
			topic = "workflow.step.failed"
		}
		e.bus.Publish(ctx, topic, "workflow", map[string]interface{}{
			"run_id":    runID,
			"step_name": sr.StepName,
			"status":    status,
		})
	}

	// If failed and no retries left, mark run as failed.
	if status == StepStatusFailed {
		e.db.WithContext(ctx).Model(&WorkflowRun{}).Where("id = ?", runID).
			Updates(map[string]interface{}{
				"status":       RunStatusFailed,
				"error":        errMsg,
				"completed_at": now,
			})
		return execErr
	}

	// Advance to next step.
	return e.advanceToNext(ctx, runID, sr.StepName)
}

func (e *Engine) retryStep(ctx context.Context, runID int64, stepName string) error {
	now := time.Now()
	return e.db.WithContext(ctx).Model(&WorkflowStepRun{}).
		Where("workflow_run_id = ? AND step_name = ?", runID, stepName).
		Updates(map[string]interface{}{
			"status":     StepStatusRunning,
			"started_at": now,
			"attempt":    gorm.Expr("attempt + 1"),
		}).Error
}

func (e *Engine) advanceToNext(ctx context.Context, runID int64, currentStep string) error {
	var steps []WorkflowStepRun
	if err := e.db.WithContext(ctx).
		Where("workflow_run_id = ?", runID).
		Order("id ASC").Find(&steps).Error; err != nil {
		return err
	}

	currentIdx := -1
	for i, s := range steps {
		if s.StepName == currentStep {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 || currentIdx >= len(steps)-1 {
		// No more steps → mark run completed.
		now := time.Now()
		e.db.WithContext(ctx).Model(&WorkflowRun{}).Where("id = ?", runID).
			Updates(map[string]interface{}{
				"status":       RunStatusCompleted,
				"completed_at": now,
			})
		if e.bus != nil {
			e.bus.Publish(ctx, "workflow.completed", "workflow", map[string]interface{}{
				"run_id": runID,
			})
		}
		return nil
	}

	// Get the def to parse step conditions.
	var run WorkflowRun
	if err := e.db.WithContext(ctx).First(&run, runID).Error; err != nil {
		return err
	}

	// Advance to next step.
	nextStep := steps[currentIdx+1]
	now := time.Now()
	nextStep.Status = StepStatusRunning
	nextStep.StartedAt = &now

	if err := e.db.WithContext(ctx).Model(&nextStep).
		Updates(map[string]interface{}{
			"status":     StepStatusRunning,
			"started_at": now,
		}).Error; err != nil {
		return err
	}

	// Parse definition steps to find and execute the next step definition.
	var defSteps []StepDef
	var def WorkflowDef
	if err := e.db.WithContext(ctx).First(&def, run.WorkflowDefID).Error; err == nil {
		json.Unmarshal([]byte(def.Steps), &defSteps)
	}

	for _, d := range defSteps {
		if d.Name == nextStep.StepName {
			// Check condition.
			if d.Condition != "" {
				ok, err := e.evalCondition(ctx, runID, d.Condition)
				if err != nil || !ok {
					e.logger.Info("step condition not met, skipping", zap.String("step", d.Name))
					e.db.WithContext(ctx).Model(&WorkflowStepRun{}).
						Where("id = ?", nextStep.ID).
						Update("status", StepStatusSkipped)
					return e.advanceToNext(ctx, runID, d.Name)
				}
			}
			e.executeStep(ctx, runID, d)
			break
		}
	}

	return nil
}

// ── Step executors ───────────────────────────────────────────────────

func (e *Engine) execAgent(ctx context.Context, runID int64, def StepDef) (map[string]interface{}, error) {
	if e.aiOrch == nil {
		// In test mode or when AI not configured, return input as output.
		return def.Inputs, nil
	}

	req := &ai.RunAgentRequest{
		AgentID:       def.AgentID,
		DecisionPoint: def.DecisionPoint,
		Context:       def.Inputs,
	}
	if req.DecisionPoint == "" {
		req.DecisionPoint = def.Name + "_decision"
	}

	var run WorkflowRun
	e.db.WithContext(ctx).First(&run, runID)
	runCtx := make(map[string]interface{})
	json.Unmarshal([]byte(run.Context), &runCtx)
	for k, v := range runCtx {
		if req.Context == nil {
			req.Context = make(map[string]interface{})
		}
		req.Context[k] = v
	}

	result, err := e.aiOrch.RunWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("agent %s: %w", def.AgentID, err)
	}

	// Serialize result to map.
	out := map[string]interface{}{
		"agent_id": def.AgentID,
		"result":   result,
		"step":     def.Name,
	}
	return out, nil
}

func (e *Engine) execCommand(ctx context.Context, runID int64, def StepDef) (map[string]interface{}, error) {
	if e.cmd == nil {
		return def.Inputs, nil
	}

	_, err := e.cmd.Dispatch(ctx, def.Command, def.Inputs)
	if err != nil {
		return nil, fmt.Errorf("command %s: %w", def.Command, err)
	}
	return map[string]interface{}{
		"command": def.Command,
		"step":    def.Name,
	}, nil
}

func (e *Engine) execFork(ctx context.Context, runID int64, def StepDef) (map[string]interface{}, error) {
	if len(def.Forks) == 0 {
		return map[string]interface{}{"fork": def.Name, "sub_steps": 0}, nil
	}

	var parentID int64
	e.db.WithContext(ctx).Model(&WorkflowStepRun{}).
		Where("workflow_run_id = ? AND step_name = ?", runID, def.Name).
		Select("id").Scan(&parentID)

	now := time.Now()
	results := make(map[string]interface{})

	// Run sub-steps sequentially within this goroutine to avoid SQLite
	// concurrency issues. For PostgreSQL, this can be made parallel.
	for _, f := range def.Forks {
		stepType := f.Type
		if stepType == "" {
			stepType = StepTypeAgent
		}
		maxAttempts := f.RetryCount + 1
		if maxAttempts < 1 {
			maxAttempts = 1
		}

		sr := WorkflowStepRun{
			WorkflowRunID:  runID,
			StepName:       f.Name,
			StepType:       stepType,
			ParentID:       &parentID,
			Status:         StepStatusRunning,
			Input:          mustJSON(f.Inputs),
			MaxAttempts:    maxAttempts,
			TimeoutSeconds: f.TimeoutSeconds,
			StartedAt:      &now,
		}
		if sr.TimeoutSeconds <= 0 {
			sr.TimeoutSeconds = 300
		}
		e.db.WithContext(ctx).Create(&sr)

		// Dispatch by step type.
		var out map[string]interface{}
		var err error
		switch f.Type {
		case StepTypeEvent:
			out, err = e.execEvent(ctx, runID, f)
		case StepTypeCommand:
			out, err = e.execCommand(ctx, runID, f)
		case StepTypeAgent:
			fallthrough
		default:
			out, err = e.execAgent(ctx, runID, f)
		}
		results[f.Name] = map[string]interface{}{"output": out, "error": errStr(err)}

		completedAt := time.Now()
		status := StepStatusCompleted
		errMsg := ""
		if err != nil {
			status = StepStatusFailed
			errMsg = err.Error()
		}
		e.db.WithContext(ctx).Model(&WorkflowStepRun{}).
			Where("id = ?", sr.ID).
			Updates(map[string]interface{}{
				"status":       status,
				"output":       mustJSON(out),
				"error":        errMsg,
				"completed_at": completedAt,
			})
	}

	return map[string]interface{}{"fork": def.Name, "results": results}, nil
}

func (e *Engine) execJoin(ctx context.Context, runID int64, def StepDef) (map[string]interface{}, error) {
	// Small initial delay to let any racing goroutines finish writes.
	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	for _, joinStep := range def.JoinSteps {
		for {
			select {
			case <-time.After(100 * time.Millisecond):
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			var sr WorkflowStepRun
			if err := e.db.WithContext(ctx).
				Where("workflow_run_id = ? AND step_name = ?", runID, joinStep).
				Order("id DESC").First(&sr).Error; err != nil {
				return nil, fmt.Errorf("join step %s not found: %w", joinStep, err)
			}
			if sr.Status == StepStatusCompleted || sr.Status == StepStatusFailed || sr.Status == StepStatusSkipped {
				break
			}
		}
	}

	return map[string]interface{}{
		"join":       def.Name,
		"join_steps": def.JoinSteps,
	}, nil
}

func (e *Engine) execDelay(ctx context.Context, runID int64, def StepDef) (map[string]interface{}, error) {
	select {
	case <-time.After(time.Duration(def.DelaySeconds) * time.Second):
		return map[string]interface{}{"delayed_seconds": def.DelaySeconds}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (e *Engine) execEvent(ctx context.Context, runID int64, def StepDef) (map[string]interface{}, error) {
	// Without a bus, block until the context times out.
	if e.bus == nil || def.WaitForEvent == "" {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	ch := make(chan map[string]interface{}, 1)
	subID := e.bus.Subscribe(def.WaitForEvent, func(ctx context.Context, evt eventbus.Event) error {
		select {
		case ch <- evt.Payload:
		default:
		}
		return nil
	})
	defer e.bus.Unsubscribe(subID)

	select {
	case payload := <-ch:
		return payload, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ── Helpers ──────────────────────────────────────────────────────────

func (e *Engine) evalCondition(ctx context.Context, runID int64, condition string) (bool, error) {
	// Simple condition evaluation: check if condition starts with "!"
	if strings.HasPrefix(condition, "!") {
		return false, nil
	}
	// For now, simple truthy check — always true unless specific patterns.
	// ponytail: simple condition eval, expand to expression parser if needed.
	return true, nil
}

func mustJSON(v interface{}) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ── Run queries ──────────────────────────────────────────────────────

func (e *Engine) ListRuns(ctx context.Context) ([]WorkflowRun, error) {
	var runs []WorkflowRun
	err := e.db.WithContext(ctx).Order("id DESC").Find(&runs).Error
	return runs, err
}

func (e *Engine) GetRun(ctx context.Context, runID int64) (*WorkflowRun, error) {
	var run WorkflowRun
	if err := e.db.WithContext(ctx).First(&run, runID).Error; err != nil {
		return nil, err
	}

	var steps []WorkflowStepRun
	e.db.WithContext(ctx).Where("workflow_run_id = ?", runID).
		Order("id ASC").Find(&steps)
	run.Steps = steps

	return &run, nil
}
