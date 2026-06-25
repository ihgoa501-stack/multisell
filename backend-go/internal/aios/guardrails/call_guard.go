package guardrails

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// PermissionGuard implements L2 call guardrail for checking whether an agent
// is permitted to invoke a given tool.
//
// The guard maintains two maps:
//   - permissions: agentID → set of permitted permissions (e.g. "inventory.read").
//   - toolPermissions: toolName → list of permissions required to call it.
//
// On Check, the guard looks up the required permissions for the requested
// tool (from toolPermissions) and verifies that the requesting agent possesses
// every one of them (in permissions). If any permission is missing the call
// is blocked.
//
// Thread safety: all public methods use sync.RWMutex so the guard is safe for
// concurrent reads while administrative writes (SetPermissions, RemoveAgent)
// are serialised.
type PermissionGuard struct {
	permissions     map[string]map[string]bool // agentID → permission → granted
	toolPermissions map[string][]string         // toolName → required permissions
	mu              sync.RWMutex
	logger          *zap.Logger
}

// NewPermissionGuard creates an empty PermissionGuard.
func NewPermissionGuard() *PermissionGuard {
	return &PermissionGuard{
		permissions:     make(map[string]map[string]bool),
		toolPermissions: make(map[string][]string),
		logger:          zap.NewNop(),
	}
}

// NewPermissionGuardWithLogger creates a guard with a custom logger.
func NewPermissionGuardWithLogger(logger *zap.Logger) *PermissionGuard {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PermissionGuard{
		permissions:     make(map[string]map[string]bool),
		toolPermissions: make(map[string][]string),
		logger:          logger,
	}
}

// Name returns "permission_guard".
func (g *PermissionGuard) Name() string {
	return "permission_guard"
}

// Check verifies that the agent identified by input.AgentID has all
// permissions required by the tool identified by input.ToolName.
//
// Rules:
//   - If the tool has no registered required permissions, it passes
//     (no restrictions = allowed).
//   - If the agent has no permissions entry at all, it is blocked
//     (no permissions = no access).
//   - If the agent possesses every required permission, it passes.
//   - If any required permission is missing, it is blocked.
func (g *PermissionGuard) Check(ctx context.Context, input *GuardInput) (*GuardResult, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Look up required permissions for the requested tool.
	required, toolExists := g.toolPermissions[input.ToolName]
	if !toolExists || len(required) == 0 {
		// No restrictions registered for this tool — allow.
		return &GuardResult{
			Pass:    true,
			Blocked: false,
			Retry:   false,
			Reason:  "tool has no permission requirements",
			Risk:    "low",
		}, nil
	}

	// Look up the agent's granted permissions.
	agentPerms, agentExists := g.permissions[input.AgentID]
	if !agentExists || len(agentPerms) == 0 {
		g.logger.Warn("permission denied — agent has no permissions",
			zap.String("agent_id", input.AgentID),
			zap.String("tool_name", input.ToolName),
		)
		return &GuardResult{
			Pass:    false,
			Blocked: true,
			Retry:   false,
			Reason:  "agent has no granted permissions",
			Risk:    "high",
		}, nil
	}

	// Check each required permission.
	for _, perm := range required {
		if !agentPerms[perm] {
			g.logger.Warn("permission denied — missing permission",
				zap.String("agent_id", input.AgentID),
				zap.String("tool_name", input.ToolName),
				zap.String("missing_permission", perm),
			)
			return &GuardResult{
				Pass:    false,
				Blocked: true,
				Retry:   false,
				Reason:  "missing permission: " + perm,
				Risk:    "high",
			}, nil
		}
	}

	return &GuardResult{
		Pass:    true,
		Blocked: false,
		Retry:   false,
		Reason:  "all permissions granted",
		Risk:    "low",
	}, nil
}

// SetPermissions configures the set of permissions granted to an agent.
// Existing permissions for the same agent are replaced.
//
// The perms slice is flattened into a map for O(1) lookup during Check.
func (g *PermissionGuard) SetPermissions(agentID string, perms []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	permSet := make(map[string]bool, len(perms))
	for _, p := range perms {
		permSet[p] = true
	}
	g.permissions[agentID] = permSet

	g.logger.Info("agent permissions updated",
		zap.String("agent_id", agentID),
		zap.Int("permission_count", len(perms)),
	)
}

// SetToolPermissions configures the set of permissions required to invoke
// a tool. Existing requirements for the same tool are replaced.
//
// This method is called once per tool at registration time, typically by
// the ToolRegistry when a tool is registered.
func (g *PermissionGuard) SetToolPermissions(toolName string, perms []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.toolPermissions[toolName] = perms

	g.logger.Info("tool permissions updated",
		zap.String("tool_name", toolName),
		zap.Int("required_permission_count", len(perms)),
	)
}

// RemoveAgent deletes all permission entries for an agent. Future checks
// for this agent will fail (blocked).
func (g *PermissionGuard) RemoveAgent(agentID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.permissions, agentID)

	g.logger.Info("agent permissions removed",
		zap.String("agent_id", agentID),
	)
}
