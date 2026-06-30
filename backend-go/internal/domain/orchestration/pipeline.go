package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PipelineOrchestrator manages the end-to-end product lifecycle.
type PipelineOrchestrator struct {
	db     *gorm.DB
	bus    *eventbus.Bus
	aiOrch *ai.Orchestrator
	logger *zap.Logger
}

// NewPipelineOrchestrator creates a new PipelineOrchestrator.
func NewPipelineOrchestrator(db *gorm.DB, bus *eventbus.Bus, aiOrch *ai.Orchestrator, logger *zap.Logger) *PipelineOrchestrator {
	return &PipelineOrchestrator{db: db, bus: bus, aiOrch: aiOrch, logger: logger}
}

// StartPipeline begins the lifecycle for a new product.
// It creates lifecycle_step records for each step in the pipeline,
// sets the first step as "running", and publishes a pipeline started event.
func (o *PipelineOrchestrator) StartPipeline(ctx context.Context, productID int64) error {
	o.logger.Info("starting product lifecycle pipeline",
		zap.Int64("product_id", productID))

	// Get pipeline config for this product (use default if none found).
	steps := DefaultPipeline
	cfg, err := o.getConfigForProduct(ctx, productID)
	if err == nil && cfg.Steps != "" {
		if parsed, parseErr := parseStepsJSON(cfg.Steps); parseErr == nil && len(parsed) > 0 {
			steps = parsed
		}
	}

	now := time.Now()
	for i, stepName := range steps {
		agentID := stepAgentMapping[stepName]
		if agentID == "" {
			agentID = stepName
		}

		step := LifecycleStep{
			ProductID: productID,
			Step:      stepName,
			AgentID:   agentID,
			Status:    StepStatusPending,
		}

		// The first step starts immediately.
		if i == 0 {
			step.Status = StepStatusRunning
			step.StartedAt = &now
		}

		if err := o.db.WithContext(ctx).Create(&step).Error; err != nil {
			return fmt.Errorf("create lifecycle step %s: %w", stepName, err)
		}
	}

	// Publish pipeline started event.
	_, pubErr := o.bus.Publish(ctx, "orchestration.pipeline.started", "orchestration",
		map[string]interface{}{
			"product_id": productID,
			"steps":      steps,
		})
	if pubErr != nil {
		o.logger.Warn("publish orchestration.pipeline.started failed",
			zap.Int64("product_id", productID),
			zap.Error(pubErr))
	}

	// Trigger the first step agent.
	firstStep := steps[0]
	return o.triggerAgent(ctx, productID, firstStep)
}

// AdvancePipeline moves the product to the next lifecycle step.
// Called by event bus subscribers when a step completes.
func (o *PipelineOrchestrator) AdvancePipeline(ctx context.Context, productID int64, currentStep string, success bool) error {
	o.logger.Info("advancing pipeline",
		zap.Int64("product_id", productID),
		zap.String("step", currentStep),
		zap.Bool("success", success))

	// Get all lifecycle steps for this product.
	var allSteps []LifecycleStep
	if err := o.db.WithContext(ctx).
		Where("product_id = ?", productID).
		Order("id ASC").
		Find(&allSteps).Error; err != nil {
		return fmt.Errorf("query lifecycle steps: %w", err)
	}

	if len(allSteps) == 0 {
		return fmt.Errorf("no lifecycle steps found for product %d", productID)
	}

	// Find the current step index and update it.
	currentIdx := -1
	for i, step := range allSteps {
		if step.Step == currentStep {
			currentIdx = i
			now := time.Now()
			duration := int(time.Since(*step.StartedAt).Milliseconds())

			if success {
				allSteps[i].Status = StepStatusCompleted
			} else {
				allSteps[i].Status = StepStatusFailed
			}
			allSteps[i].CompletedAt = &now
			allSteps[i].DurationMs = duration

			if err := o.db.WithContext(ctx).Model(&allSteps[i]).
				Select("status", "completed_at", "duration_ms", "result", "error").
				Updates(map[string]interface{}{
					"status":       allSteps[i].Status,
					"completed_at": now,
					"duration_ms":  duration,
				}).Error; err != nil {
				return fmt.Errorf("update lifecycle step %s: %w", currentStep, err)
			}
			break
		}
	}

	if currentIdx == -1 {
		return fmt.Errorf("step %s not found for product %d", currentStep, productID)
	}

	// Get config for retry/failure policy.
	cfg, cfgErr := o.getConfigForProduct(ctx, productID)
	if cfgErr != nil {
		cfg = &OrchestrationConfig{FailureAction: FailureActionStop, AutoRetryCount: 3}
	}

	// If failed, check retry policy.
	if !success {
		switch cfg.FailureAction {
		case FailureActionSkip:
			// Mark this step skipped and advance to next.
			_ = o.db.WithContext(ctx).Model(&LifecycleStep{}).
				Where("product_id = ? AND step = ?", productID, currentStep).
				Update("status", StepStatusSkipped)
			return o.advanceToNext(ctx, productID, currentIdx, allSteps)
		case FailureActionRetry:
			// Check retry count via error_attempts column (or just retry once).
			return o.triggerAgent(ctx, productID, currentStep)
		case FailureActionStop:
		default:
			// Stop on failure  publish notification.
			_, pubErr := o.bus.Publish(ctx, "orchestration.pipeline.failed", "orchestration",
				map[string]interface{}{
					"product_id": productID,
					"step":       currentStep,
				})
			if pubErr != nil {
				o.logger.Warn("publish orchestration.pipeline.failed failed",
					zap.Int64("product_id", productID), zap.Error(pubErr))
			}
			return nil
		}
		return nil
	}

	// Success  advance to next step.
	if currentIdx < len(allSteps)-1 {
		return o.advanceToNext(ctx, productID, currentIdx, allSteps)
	}

	// Pipeline complete.
	_, pubErr := o.bus.Publish(ctx, "orchestration.pipeline.completed", "orchestration",
		map[string]interface{}{
			"product_id": productID,
		})
	if pubErr != nil {
		o.logger.Warn("publish orchestration.pipeline.completed failed",
			zap.Int64("product_id", productID), zap.Error(pubErr))
	}

	return nil
}

// advanceToNext sets the next step as running and triggers its agent.
func (o *PipelineOrchestrator) advanceToNext(ctx context.Context, productID int64, currentIdx int, allSteps []LifecycleStep) error {
	nextIdx := currentIdx + 1
	if nextIdx >= len(allSteps) {
		return nil
	}

	now := time.Now()
	if err := o.db.WithContext(ctx).Model(&LifecycleStep{}).
		Where("product_id = ? AND step = ?", productID, allSteps[nextIdx].Step).
		Updates(map[string]interface{}{
			"status":     StepStatusRunning,
			"started_at": now,
		}).Error; err != nil {
		return fmt.Errorf("start next step %s: %w", allSteps[nextIdx].Step, err)
	}

	// Trigger the next step agent.
	return o.triggerAgent(ctx, productID, allSteps[nextIdx].Step)
}

// triggerAgent runs the agent responsible for the given pipeline step.
func (o *PipelineOrchestrator) triggerAgent(ctx context.Context, productID int64, stepName string) error {
	agentID := stepAgentMapping[stepName]
	if agentID == "" {
		o.logger.Warn("no agent mapping for pipeline step", zap.String("step", stepName))
		return nil
	}
	if o.aiOrch == nil {
		o.logger.Warn("ai orchestrator not configured, skipping agent trigger",
			zap.String("step", stepName), zap.String("agent_id", agentID))
		return nil
	}

	dp := stepDecisionPoint[stepName]
	if dp == "" {
		dp = stepName + "_decision"
	}

	_, err := o.aiOrch.RunWithContext(ctx, &ai.RunAgentRequest{
		AgentID:       agentID,
		DecisionPoint: dp,
		Context: map[string]interface{}{
			"product_id": productID,
			"step":       stepName,
		},
	})
	if err != nil {
		o.logger.Warn("agent trigger failed for pipeline step",
			zap.String("step", stepName),
			zap.String("agent_id", agentID),
			zap.Int64("product_id", productID),
			zap.Error(err))
		return err
	}
	return nil
}

// GetPipelineStatus returns the current status of all lifecycle steps for a product.
func (o *PipelineOrchestrator) GetPipelineStatus(ctx context.Context, productID int64) ([]LifecycleStep, error) {
	var steps []LifecycleStep
	err := o.db.WithContext(ctx).
		Where("product_id = ?", productID).
		Order("id ASC").
		Find(&steps).Error
	return steps, err
}

// RetryStep resets a failed step to running and triggers its agent.
func (o *PipelineOrchestrator) RetryStep(ctx context.Context, productID int64, stepName string) error {
	return o.triggerAgent(ctx, productID, stepName)
}

// getConfigForProduct retrieves the orchestration config for a product.
// For now, returns the first active config. In production, this should
// respect per-product or per-tenant configuration.
func (o *PipelineOrchestrator) getConfigForProduct(ctx context.Context, productID int64) (*OrchestrationConfig, error) {
	var cfg OrchestrationConfig
	if err := o.db.WithContext(ctx).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// parseStepsJSON parses a JSON string into a string slice of step names.
func parseStepsJSON(jsonStr string) ([]string, error) {
	var steps []string
	if err := json.Unmarshal([]byte(jsonStr), &steps); err != nil {
		return nil, err
	}
	return steps, nil
}

// ListConfigs returns all orchestration configs.
func (o *PipelineOrchestrator) ListConfigs(ctx context.Context) ([]OrchestrationConfig, error) {
	var configs []OrchestrationConfig
	if err := o.db.WithContext(ctx).Order("id ASC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// CreateConfig creates a new orchestration config.
func (o *PipelineOrchestrator) CreateConfig(ctx context.Context, cfg *OrchestrationConfig) error {
	return o.db.WithContext(ctx).Create(cfg).Error
}
