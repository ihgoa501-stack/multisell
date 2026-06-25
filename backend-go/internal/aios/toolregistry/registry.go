package toolregistry

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// ToolRegistry is the central registry for all agent-callable tools.
// It provides thread-safe registration, lookup, listing, and invocation
// of tools with hook chain support and circuit breaker integration.
//
// Tools are identified by their Name+Version combination (Key method).
// Lookup by bare name returns the version with the highest string value
// (lexicographic comparison — suitable for semver with equal-width segments).
type ToolRegistry struct {
	tools  map[string]*Tool
	mu     sync.RWMutex
	hooks  []ToolHook
	logger *zap.Logger
}

// NewToolRegistry creates a new ToolRegistry with the given logger.
func NewToolRegistry(logger *zap.Logger) *ToolRegistry {
	return &ToolRegistry{
		tools:  make(map[string]*Tool),
		hooks:  make([]ToolHook, 0),
		logger: logger,
	}
}

// Register adds a tool to the registry. It panics if a tool with the same
// Name+Version combination is already registered, if tool is nil, or if
// Name or Version is empty.
func (r *ToolRegistry) Register(tool *Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool == nil {
		panic("toolregistry: cannot register nil tool")
	}
	if tool.Name == "" {
		panic("toolregistry: tool name must not be empty")
	}
	if tool.Version == "" {
		panic("toolregistry: tool version must not be empty")
	}
	key := tool.Key()
	if _, exists := r.tools[key]; exists {
		panic(fmt.Sprintf("toolregistry: tool %s is already registered", key))
	}
	r.tools[key] = tool
	r.logger.Info("tool registered",
		zap.String("name", tool.Name),
		zap.String("version", tool.Version),
		zap.String("squad", tool.Squad),
		zap.String("risk_level", string(tool.RiskLevel)),
	)
}

// Lookup finds a tool by name. If the lookup key matches the "name@version"
// format exactly (contains "@"), it looks up a specific version. Otherwise
// it searches by tool name and returns the one with the highest version
// string value (lexicographic). Returns nil and false if not found.
func (r *ToolRegistry) Lookup(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First try exact lookup (supports both "name@version" and bare keys)
	if tool, ok := r.tools[name]; ok {
		return tool, true
	}

	// Search by tool name, return latest version
	var candidates []*Tool
	for _, tool := range r.tools {
		if tool.Name == name {
			candidates = append(candidates, tool)
		}
	}

	if len(candidates) == 0 {
		return nil, false
	}
	if len(candidates) == 1 {
		return candidates[0], true
	}

	// Return the version with the highest string value
	latest := candidates[0]
	for _, c := range candidates[1:] {
		if c.Version > latest.Version {
			latest = c
		}
	}
	return latest, true
}

// List returns all registered tools. If squads are provided, only tools
// whose Squad matches one of the given squads are returned.
func (r *ToolRegistry) List(squad ...string) []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	squadSet := make(map[string]struct{}, len(squad))
	for _, s := range squad {
		squadSet[s] = struct{}{}
	}

	result := make([]*Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		if len(squad) > 0 {
			if _, ok := squadSet[tool.Squad]; !ok {
				continue
			}
		}
		result = append(result, tool)
	}
	return result
}

// AddHook appends a hook to the registry's hook chain. Hooks are executed
// in the order they are added. This method is thread-safe.
func (r *ToolRegistry) AddHook(hook ToolHook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, hook)
	r.logger.Info("hook added to registry",
		zap.String("hook_type", fmt.Sprintf("%T", hook)),
	)
}

// Call executes a tool by name with the given input, applying all registered
// hooks before and after the handler. It returns the handler's output or an
// error if the tool was not found, a hook rejected the call, or the handler
// failed. This method is thread-safe.
func (r *ToolRegistry) Call(ctx context.Context, name string, input map[string]interface{}) (interface{}, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if input == nil {
		input = make(map[string]interface{})
	}

	tool, ok := r.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}

	// Snapshot hooks for this call to avoid holding the lock during execution
	hooks := r.snapshotHooks()

	return tool.Call(ctx, input, hooks)
}

// snapshotHooks returns a copy of the hooks slice under read lock.
func (r *ToolRegistry) snapshotHooks() []ToolHook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	hooks := make([]ToolHook, len(r.hooks))
	copy(hooks, r.hooks)
	return hooks
}

// ToolCount returns the number of registered tools.
func (r *ToolRegistry) ToolCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}
