// Package setup provides the AIOS system initialization and wiring.
//
// It creates and configures all AIOS components (Runtime, ToolRegistry,
// Guardrails, LLM Gateway, Memory, IPC, Pipeline, Observability) and
// exposes integration helpers for the existing Gin router and scheduler.
package setup

import (
	"context"
	"time"

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
// Replace with a real provider wrapper in production.
type stubProvider struct{}

func (s *stubProvider) Name() string { return "stub" }

func (s *stubProvider) Chat(_ context.Context, req *llmgateway.Request) (*llmgateway.Response, error) {
	return &llmgateway.Response{
		Content: "stub response for " + req.AgentID,
		ModelUsed: "stub-v1",
		Cached:  false,
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

	// 2. Create Runtime (agent lifecycle, heartbeats, resource limits).
	rt := runtime.New(logger, bus)

	// 3. Create Guardrails Chain (L1-L5 defensive checks).
	chain := guardrails.NewChainWithLogger(logger)
	chain.Add(guardrails.NewPromptInjectionGuard())
	chain.Add(guardrails.NewPermissionGuard())
	chain.Add(guardrails.NewOutputGuard())
	chain.Add(guardrails.NewExecutionGuard())
	chain.Add(guardrails.NewRollbackGuard(logger))

	// 4. Create LLM Gateway with default routing and caching.
	gw := llmgateway.NewGateway(&stubProvider{}, logger)

	// 5. Create Working Memory (short-term per-session memory).
	mem := memory.NewWorkingMemoryBucket("aios", "global", 15*time.Minute, logger)

	// 6. Create IPC (inter-agent communication bus).
	ipcBus := ipc.New(logger)

	// 7. Create Pipeline Engine (serial, fan-out, self-correct, fallback).
	pl := pipeline.New(logger)

	// 8. Create Observability Collector.
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
