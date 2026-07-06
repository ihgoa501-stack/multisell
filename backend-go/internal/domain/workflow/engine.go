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

// approvalSignal is sent through the approval channel to unblock a pending approval step.
type approvalSignal struct {
	approved bool
	comment  string
	reviewer string
}

// Engine is the generic workflow execution engine.
type Engine struct {
	db       *gorm.DB
	bus      *eventbus.Bus
	aiOrch   *ai.Orchestrator
	cmd      *command.Dispatcher
	logger   *zap.Logger

	mu            sync.Mutex
	running       map[int64]context.CancelFunc // cancel running steps per run
	approvalChans map[string]chan approvalSignal
	approvalMu    sync.Mutex
}

// NewEngine creates a workflow engine.
func NewEngine(db *gorm.DB, bus *eventbus.Bus, aiOrch *ai.Orchestrator, dispatcher *command.Dispatcher, logger *zap.Logger) *Engine {
	return &Engine{
		db:            db,
		bus:           bus,
		aiOrch:        aiOrch,
		cmd:           dispatcher,
		logger:        logger.Named("workflow"),
		running:       make(map[int64]context.CancelFunc),
		approvalChans: make(map[string]chan approvalSignal),
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

// ListDefsPaginated returns a paginated list of workflow definitions with total count.
func (e *Engine) ListDefsPaginated(ctx context.Context, page, size int) ([]WorkflowDef, int64, error) {
	var total int64
	e.db.WithContext(ctx).Model(&WorkflowDef{}).Count(&total)

	var defs []WorkflowDef
	offset := (page - 1) * size
	err := e.db.WithContext(ctx).Order("id DESC").Offset(offset).Limit(size).Find(&defs).Error
	return defs, total, err
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
		case StepTypeCondition:
			output, execErr = e.execCondition(ctx, runID, def)
		case StepTypeApproval:
			output, execErr = e.execApproval(ctx, runID, def)
		case StepTypeAction:
			output, execErr = e.execAction(ctx, runID, def)
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

	// ponytail: raw Dispatch — workflow engine runs predefined pipeline steps,
	// not AI-generated actions. No action gate needed; command registration is
	// the authorization boundary here.
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
	type forkResult struct {
		name   string
		output map[string]interface{}
		err    error
	}
	ch := make(chan forkResult, len(def.Forks))

	for _, f := range def.Forks {
		f := f // capture
		go func() {
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

			// ponytail: global DB in goroutines — safe for PostgreSQL.
			// SQLite concurrency is handled by dbtest.NewDB (single connection).
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

			ch <- forkResult{name: f.Name, output: out, err: err}
		}()
	}

	results := make(map[string]interface{})
	var firstErr error
	for range def.Forks {
		r := <-ch
		results[r.name] = map[string]interface{}{"output": r.output, "error": errStr(r.err)}
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
	}
	close(ch)

	return map[string]interface{}{"fork": def.Name, "results": results, "error": errStr(firstErr)}, nil
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

// ── Condition / Approval / Action executors ──────────────────────────

func (e *Engine) execCondition(ctx context.Context, runID int64, def StepDef) (map[string]interface{}, error) {
	if def.Condition == "" {
		return map[string]interface{}{"result": true, "condition": ""}, nil
	}
	result, err := e.evalCondition(ctx, runID, def.Condition)
	if err != nil {
		return nil, fmt.Errorf("condition eval: %w", err)
	}
	return map[string]interface{}{
		"result":    result,
		"condition": def.Condition,
	}, nil
}

func (e *Engine) execApproval(ctx context.Context, runID int64, def StepDef) (map[string]interface{}, error) {
	key := fmt.Sprintf("%d:%s", runID, def.Name)
	ch := make(chan approvalSignal, 1)

	e.approvalMu.Lock()
	e.approvalChans[key] = ch
	e.approvalMu.Unlock()

	defer func() {
		e.approvalMu.Lock()
		delete(e.approvalChans, key)
		e.approvalMu.Unlock()
	}()

	// Publish event that the step awaits approval.
	if e.bus != nil {
		e.bus.Publish(ctx, "workflow.step.pending_approval", "workflow", map[string]interface{}{
			"run_id":    runID,
			"step_name": def.Name,
		})
	}

	// Mark step as pending approval.
	e.db.WithContext(ctx).Model(&WorkflowStepRun{}).
		Where("workflow_run_id = ? AND step_name = ?", runID, def.Name).
		Update("status", StepStatusPending)

	e.logger.Info("step awaiting approval", zap.Int64("run_id", runID), zap.String("step", def.Name))

	select {
	case sig := <-ch:
		if sig.approved {
			return map[string]interface{}{
				"approved": true,
				"comment":  sig.comment,
				"reviewer": sig.reviewer,
			}, nil
		}
		return map[string]interface{}{
			"approved": false,
			"comment":  sig.comment,
			"reviewer": sig.reviewer,
		}, fmt.Errorf("step rejected: %s", sig.comment)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (e *Engine) execAction(ctx context.Context, runID int64, def StepDef) (map[string]interface{}, error) {
	if def.Command != "" {
		return e.execCommand(ctx, runID, def)
	}
	if def.AgentID != "" {
		return e.execAgent(ctx, runID, def)
	}
	return def.Inputs, nil
}

// ── Approval API ─────────────────────────────────────────────────────

// ApproveStep signals a pending approval step to proceed.
func (e *Engine) ApproveStep(ctx context.Context, runID int64, stepName, reviewer, comment string) error {
	key := fmt.Sprintf("%d:%s", runID, stepName)

	e.approvalMu.Lock()
	ch, ok := e.approvalChans[key]
	e.approvalMu.Unlock()

	if !ok {
		return fmt.Errorf("no pending approval for run %d step %s", runID, stepName)
	}

	ch <- approvalSignal{approved: true, comment: comment, reviewer: reviewer}

	e.db.WithContext(ctx).Model(&WorkflowStepRun{}).
		Where("workflow_run_id = ? AND step_name = ?", runID, stepName).
		Updates(map[string]interface{}{
			"status": StepStatusApproved,
			"output": mustJSON(map[string]interface{}{
				"approved": true,
				"comment":  comment,
				"reviewer": reviewer,
			}),
		})
	return nil
}

// RejectStep signals a pending approval step to be rejected.
func (e *Engine) RejectStep(ctx context.Context, runID int64, stepName, reviewer, comment string) error {
	key := fmt.Sprintf("%d:%s", runID, stepName)

	e.approvalMu.Lock()
	ch, ok := e.approvalChans[key]
	e.approvalMu.Unlock()

	if !ok {
		return fmt.Errorf("no pending approval for run %d step %s", runID, stepName)
	}

	ch <- approvalSignal{approved: false, comment: comment, reviewer: reviewer}

	e.db.WithContext(ctx).Model(&WorkflowStepRun{}).
		Where("workflow_run_id = ? AND step_name = ?", runID, stepName).
		Updates(map[string]interface{}{
			"status": StepStatusRejected,
			"output": mustJSON(map[string]interface{}{
				"approved": false,
				"comment":  comment,
				"reviewer": reviewer,
			}),
		})
	return nil
}

// ── Node CRUD ────────────────────────────────────────────────────────

// CreateNode adds a node to a workflow definition.
func (e *Engine) CreateNode(ctx context.Context, node *WorkflowNode) error {
	return e.db.WithContext(ctx).Create(node).Error
}

// ListNodes returns all nodes for a workflow definition, ordered by order_index.
func (e *Engine) ListNodes(ctx context.Context, workflowID uint) ([]WorkflowNode, error) {
	var nodes []WorkflowNode
	err := e.db.WithContext(ctx).
		Where("workflow_def_id = ?", workflowID).
		Order("order_index ASC, id ASC").
		Find(&nodes).Error
	return nodes, err
}

// DeleteNode removes a single node.
func (e *Engine) DeleteNode(ctx context.Context, nodeID uint) error {
	return e.db.WithContext(ctx).Delete(&WorkflowNode{}, nodeID).Error
}

// ── Event helpers ───────────────────────────────────────────────────────────

// PublishEvent wraps the event bus so callers don't need to import eventbus directly.
func (e *Engine) PublishEvent(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error) {
	if e.bus == nil {
		return "", nil
	}
	return e.bus.Publish(ctx, topic, source, payload)
}

// ── Monitoring ──────────────────────────────────────────────────────────────

// MonitorStats holds aggregated run statistics.
type MonitorStats struct {
	TotalRuns        int              `json:"total_runs"`
	ByStatus         map[string]int   `json:"by_status"`
	AverageDurationS float64          `json:"average_duration_s"`
	FailureByStep    map[string]int   `json:"failure_by_step"`
}

// GetMonitorStats computes aggregated monitoring statistics.
func (e *Engine) GetMonitorStats(ctx context.Context) (*MonitorStats, error) {
	var total int64
	e.db.WithContext(ctx).Model(&WorkflowRun{}).Count(&total)

	var runs []WorkflowRun
	e.db.WithContext(ctx).Order("id ASC").Find(&runs)

	byStatus := map[string]int{}
	var totalDurationS float64
	var completedCount int

	for _, r := range runs {
		byStatus[r.Status]++
		if r.StartedAt != nil && r.CompletedAt != nil && (r.Status == RunStatusCompleted || r.Status == RunStatusFailed) {
			duration := r.CompletedAt.Sub(*r.StartedAt).Seconds()
			totalDurationS += duration
			completedCount++
		}
	}

	var avgDurationS float64
	if completedCount > 0 {
		avgDurationS = totalDurationS / float64(completedCount)
	}

	// Failure by step name.
	var failedSteps []struct {
		StepName string
		Count    int
	}
	e.db.WithContext(ctx).Model(&WorkflowStepRun{}).
		Select("step_name, COUNT(*) as count").
		Where("status = ?", StepStatusFailed).
		Group("step_name").
		Order("count DESC").
		Limit(10).
		Scan(&failedSteps)

	failureByStep := make(map[string]int, len(failedSteps))
	for _, fs := range failedSteps {
		failureByStep[fs.StepName] = fs.Count
	}

	return &MonitorStats{
		TotalRuns:        int(total),
		ByStatus:         byStatus,
		AverageDurationS: avgDurationS,
		FailureByStep:    failureByStep,
	}, nil
}

// evalCondition evaluates expressions in the format:
//
//	$steps.STEP_NAME.field == "value"
//	$steps.STEP_NAME.field != "value"
func (e *Engine) evalCondition(ctx context.Context, runID int64, condition string) (bool, error) {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true, nil
	}

	// Match: $steps.STEP_NAME.FIELD OP "VALUE"
	parts := strings.SplitN(condition, " ", 3)
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid condition format: %q", condition)
	}

	left := strings.TrimSpace(parts[0])
	op := strings.TrimSpace(parts[1])
	right := strings.TrimSpace(parts[2])

	// Parse left side: $steps.STEP_NAME.FIELD
	if !strings.HasPrefix(left, "$steps.") {
		return false, fmt.Errorf("condition must reference $steps.*, got: %q", left)
	}
	ref := strings.TrimPrefix(left, "$steps.")
	refParts := strings.SplitN(ref, ".", 2)
	if len(refParts) != 2 {
		return false, fmt.Errorf("condition reference must be step_name.field, got: %q", ref)
	}
	stepName, fieldName := refParts[0], refParts[1]

	// Look up the referenced step run.
	var sr WorkflowStepRun
	if err := e.db.WithContext(ctx).
		Where("workflow_run_id = ? AND step_name = ?", runID, stepName).
		Order("id DESC").First(&sr).Error; err != nil {
		return false, fmt.Errorf("reference step %q not found: %w", stepName, err)
	}

	// Get the field value from the step run.
	var actual string
	switch fieldName {
	case "status":
		actual = sr.Status
	case "step_type":
		actual = sr.StepType
	default:
		return false, fmt.Errorf("unsupported field: %q", fieldName)
	}

	// Strip quotes from right side.
	expected := strings.Trim(right, `"'`)

	switch op {
	case "==":
		return actual == expected, nil
	case "!=":
		return actual != expected, nil
	default:
		return false, fmt.Errorf("unsupported operator: %q", op)
	}
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

// ── Monitoring ──────────────────────────────────────────────────────────

// MonitorResult holds simple per-status counts plus 24h completion count.
// Used by the GET /workflow/monitor endpoint.
type MonitorResult struct {
	Running      int `json:"running"`
	Pending      int `json:"pending"`
	Blocked      int `json:"blocked"`
	Failed       int `json:"failed"`
	Completed24h int `json:"completed_24h"`
}

// GetMonitor returns counts of workflow runs grouped by status, plus
// the number completed in the last 24 hours.
func (e *Engine) GetMonitor(ctx context.Context) (*MonitorResult, error) {
	result := &MonitorResult{}

	type statusCount struct {
		Status string
		Count  int
	}
	var counts []statusCount
	e.db.WithContext(ctx).Model(&WorkflowRun{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&counts)

	for _, c := range counts {
		switch c.Status {
		case RunStatusRunning:
			result.Running = c.Count
		case RunStatusPending:
			result.Pending = c.Count
		case RunStatusPaused:
			result.Blocked = c.Count
		case RunStatusFailed:
			result.Failed = c.Count
		}
	}

	// Completed in the last 24 hours.
	var completed24h int64
	e.db.WithContext(ctx).Model(&WorkflowRun{}).
		Where("status = ? AND completed_at > ?", RunStatusCompleted, time.Now().Add(-24*time.Hour)).
		Count(&completed24h)
	result.Completed24h = int(completed24h)

	return result, nil
}

// ── Run queries with filtering ──────────────────────────────────────────

// ListRunsFiltered returns paginated workflow runs, optionally filtered by
// workflow_def_id.
func (e *Engine) ListRunsFiltered(ctx context.Context, workflowID *int64, page, size int) ([]WorkflowRun, int64, error) {
	query := e.db.WithContext(ctx).Model(&WorkflowRun{})
	if workflowID != nil {
		query = query.Where("workflow_def_id = ?", *workflowID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var runs []WorkflowRun
	if err := query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&runs).Error; err != nil {
		return nil, 0, err
	}
	return runs, total, nil
}

// RetryRun resets a failed run and increments the retry counter.
// Returns an error if the run is not failed or has exhausted retries.
func (e *Engine) RetryRun(ctx context.Context, runID int64) error {
	var run WorkflowRun
	if err := e.db.WithContext(ctx).First(&run, runID).Error; err != nil {
		return fmt.Errorf("run not found: %w", err)
	}
	if run.Status != RunStatusFailed {
		return fmt.Errorf("run %d is not failed (status: %s)", runID, run.Status)
	}
	if run.RetryCount >= run.MaxRetries {
		return fmt.Errorf("run %d has exhausted max retries (%d)", runID, run.MaxRetries)
	}

	now := time.Now()

	// Reset steps to pending.
	e.db.WithContext(ctx).Model(&WorkflowStepRun{}).
		Where("workflow_run_id = ?", runID).
		Updates(map[string]interface{}{
			"status":       StepStatusPending,
			"error":        "",
			"started_at":   nil,
			"completed_at": nil,
		})

	// Reset run.
	return e.db.WithContext(ctx).Model(&run).Updates(map[string]interface{}{
		"status":          RunStatusRunning,
		"error":           "",
		"started_at":      now,
		"completed_at":    nil,
		"retry_count":     run.RetryCount + 1,
		"current_node_id": 0,
	}).Error
}
