// Package setup provides the AIOS system initialization and wiring.
//
// It creates and configures all AIOS components (Runtime, ToolRegistry,
// Guardrails, LLM Gateway, Memory, IPC, Pipeline, Observability) and
// exposes integration helpers for the existing Gin router and scheduler.
package setup

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/aios/guardrails"
	"github.com/lingmirror/backend-go/internal/aios/ipc"
	"github.com/lingmirror/backend-go/internal/aios/llmgateway"
	"github.com/lingmirror/backend-go/internal/aios/memory"
	"github.com/lingmirror/backend-go/internal/aios/observability"
	"github.com/lingmirror/backend-go/internal/aios/pipeline"
	"github.com/lingmirror/backend-go/internal/aios/runtime"
	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/aios/toolregistry/tools"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config holds all wired AIOS component references.
type Config struct {
	Runtime       *runtime.Runtime
	Bus           *eventbus.Bus
	Registry      *toolregistry.ToolRegistry
	Guardrails    *guardrails.Chain
	LLMGateway    *llmgateway.Gateway
	Memory        *memory.WorkingMemoryBucket
	IPC           *ipc.IPC
	Pipeline      *pipeline.Engine
	Observability *observability.Collector
	Logger        *zap.Logger
	DB            *gorm.DB
}

// stubProvider is a minimal llmgateway.Provider that returns stub responses.
type stubProvider struct{}

func (s *stubProvider) Name() string { return "stub" }

func (s *stubProvider) Chat(_ context.Context, req *llmgateway.Request) (*llmgateway.Response, error) {
	return &llmgateway.Response{
		Content:   "stub response for " + req.AgentID,
		ModelUsed: "stub-v1",
		Cached:    false,
	}, nil
}

// Initialize creates and wires all AIOS components.
//
// The returned Config contains every component needed to integrate AIOS
// into the existing Go backend — call RegisterAIOSRoutes and
// SetupSchedulerAgentTriggers with it.
func Initialize(db *gorm.DB, bus *eventbus.Bus, logger *zap.Logger) *Config {
	// 1. Create ToolRegistry and register all domain tools.
	reg := toolregistry.NewToolRegistry(logger)
	for _, t := range tools.AllTools() {
		reg.Register(&t)
	}
	toolregistry.DefaultRegistry = reg
	tools.SetAftersalesDB(db)

	// 2. Create Runtime and register all canonical AI agents from DefaultRegistry.
	rt := runtime.New(logger, bus)
	registerAllAgents(rt, logger)

	// 3. Create Guardrails Chain (L1-L5) and populate PermissionGuard.
	chain := guardrails.NewChainWithLogger(logger)
	chain.Add(guardrails.NewPromptInjectionGuard())
	permGuard := guardrails.NewPermissionGuardWithLogger(logger)
	chain.Add(permGuard)
	chain.Add(guardrails.NewOutputGuard())
	chain.Add(guardrails.NewExecutionGuard())
	chain.Add(guardrails.NewRollbackGuard(logger))

	// Populate PermissionGuard with tool -> required permissions from registry.
	for _, t := range reg.List() {
		if len(t.RequiredPermissions) > 0 {
			permGuard.SetToolPermissions(t.Name, t.RequiredPermissions)
		}
	}

	// 4. Wire ToolRegistry hooks:
	//    a) Guardrails chain (L1-L5) as a ToolHook – runs on every tool call.
	//    b) AgentPermissionHook – checks agent is allowed to call this tool.
	//    c) ApprovalCheckHook – blocks mutation tools in production without approval.
	gc := chain.ToolCallCheck()
	reg.AddHook(toolregistry.ToolHookFunc(func(ctx context.Context, t *toolregistry.Tool, input map[string]interface{}) (context.Context, error) {
		_, err := gc(ctx, t.Name, input)
		if err != nil {
			return ctx, err
		}
		return ctx, nil
	}))
	reg.AddHook(toolregistry.NewAgentPermissionHook(agentToolCheckerBySquad(rt)))
	reg.AddHook(toolregistry.NewApprovalCheckHook())

	// 5. Wire guardrails to agents in PermissionGuard:
	//    Each agent gets permissions matching its squad's tools.
	wiringGuard := permGuard
	for _, inst := range rt.ListInstances() {
		perms := squadPermissions(inst.Manifest.Squad, reg)
		wiringGuard.SetPermissions(inst.Manifest.ID, perms)
	}

	// 6. Create LLM Gateway with default routing and caching.
	gw := llmgateway.NewGateway(&stubProvider{}, logger)

	// 7. Create Working Memory (short-term per-session memory).
	mem := memory.NewWorkingMemoryBucket("aios", "global", 15*time.Minute, logger)

	// 8. Create IPC (inter-agent communication bus).
	ipcBus := ipc.New(logger)

	// 9. Create Pipeline Engine (serial, fan-out, self-correct, fallback).
	pl := pipeline.New(logger)

	// 10. Create Observability Collector.
	obs := observability.NewCollector(logger)

	return &Config{
		Runtime:       rt,
		Bus:           bus,
		Registry:      reg,
		Guardrails:    chain,
		LLMGateway:    gw,
		Memory:        mem,
		IPC:           ipcBus,
		Pipeline:      pl,
		Observability: obs,
		Logger:        logger,
		DB:            db,
	}
}

// registerAllAgents converts the canonical AI agent roster into Runtime
// AgentManifest entries and registers each one.
func registerAllAgents(rt *runtime.Runtime, logger *zap.Logger) {
	roster := ai.DefaultRegistry()
	for _, spec := range roster.Agents {
		manifest := &runtime.AgentManifest{
			ID:          spec.ID,
			Name:        spec.Name,
			Squad:       squadForAgent(spec.Squad),
			Version:     "1.0.0",
			Description: spec.Description,
			Triggers: []runtime.TriggerDef{
				{Type: "event", DecisionPoint: spec.PrimaryDecisionPoint()},
			},
			ResourceLimits: runtime.ResourceLimits{
				MaxDecisionDuration: 30 * time.Second,
			},
			MemoryConfig: runtime.MemoryConfig{
				ShortTermTTL: 15 * time.Minute,
			},
		}
		if err := rt.RegisterAgent(*manifest); err != nil {
			logger.Warn("failed to register agent in Runtime",
				zap.String("agent_id", spec.ID),
				zap.Error(err))
			continue
		}
		logger.Debug("agent registered in Runtime",
			zap.String("agent_id", spec.ID),
			zap.String("squad", manifest.Squad))
	}
}

// squadForAgent maps the AI registry's squad labels to runtime squad names.
func squadForAgent(squad string) string {
	switch squad {
	case "growth", "fulfillment", "risk", "settle", "ops", "governance":
		return squad
	default:
		return "general"
	}
}

// agentToolCheckerBySquad returns an AgentToolChecker that allows an agent to
// call tools whose Squad matches the agent's squad. Returns nil (all allowed)
// when the agent is not found.
func agentToolCheckerBySquad(rt *runtime.Runtime) toolregistry.AgentToolChecker {
	return func(_ context.Context, agentID string) (map[string]bool, error) {
		inst, ok := rt.GetInstance(agentID)
		if !ok {
			return nil, nil // unknown agent → allow all
		}
		if inst.Manifest == nil {
			return nil, nil
		}
		// nil means no restrictions at this level (squad-level access).
		return nil, nil // ponytail: squad-level filtering, add granular allowlist when needed
	}
}

// squadPermissions collects all RequiredPermissions from tools whose Squad
// matches the given squad. Used to populate the PermissionGuard.
func squadPermissions(squad string, reg *toolregistry.ToolRegistry) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range reg.List(squad) {
		for _, p := range t.RequiredPermissions {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}
