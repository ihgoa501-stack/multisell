package sdk

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/lingmirror/backend-go/internal/aios/runtime"
)

// RegisterAgent validates an AgentDef, converts it to a runtime.AgentManifest,
// and registers it with the agent runtime. The agent is started immediately
// after registration.
func RegisterAgent(rt *runtime.Runtime, def *AgentDef) error {
	if err := Validate(def); err != nil {
		return fmt.Errorf("validate agent %q: %w", def.AgentID, err)
	}
	manifest, err := toManifest(def)
	if err != nil {
		return fmt.Errorf("convert agent %q to manifest: %w", def.AgentID, err)
	}
	if err := rt.RegisterAgent(*manifest); err != nil {
		return fmt.Errorf("register agent %q: %w", def.AgentID, err)
	}
	if err := rt.StartAgent(manifest.ID); err != nil {
		return fmt.Errorf("start agent %q: %w", def.AgentID, err)
	}
	return nil
}

// RegisterFromYAML reads an agent definition from a YAML file and registers it
// with the agent runtime.
func RegisterFromYAML(rt *runtime.Runtime, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read YAML file %q: %w", path, err)
	}
	var def AgentDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return fmt.Errorf("parse YAML file %q: %w", path, err)
	}
	return RegisterAgent(rt, &def)
}

// Validate checks an AgentDef for correctness.
func Validate(def *AgentDef) error {
	if def == nil {
		return fmt.Errorf("agent definition is nil")
	}
	if def.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if def.Name == "" {
		return fmt.Errorf("name is required for agent %q", def.AgentID)
	}
	if !isValidSquad(def.Squad) {
		return fmt.Errorf("squad %q for agent %q is not valid: must be one of growth/fulfillment/risk/settle/governance", def.Squad, def.AgentID)
	}
	if !isValidAutonomy(def.Autonomy) {
		return fmt.Errorf("autonomy %q for agent %q is not valid: must be one of advisory/guided/supervised/autonomous", def.Autonomy, def.AgentID)
	}
	if !isValidRiskFloor(def.RiskFloor) {
		return fmt.Errorf("risk_floor %q for agent %q is not valid: must be one of low/medium/high/critical", def.RiskFloor, def.AgentID)
	}
	if def.ModelHint == "" {
		return fmt.Errorf("model_hint is required for agent %q", def.AgentID)
	}
	if len(def.DecisionPoints) == 0 {
		return fmt.Errorf("decision_points is required (non-empty) for agent %q", def.AgentID)
	}
	return nil
}

func isValidSquad(s string) bool {
	switch s {
	case "growth", "fulfillment", "risk", "settle", "governance":
		return true
	}
	return false
}

func isValidAutonomy(s string) bool {
	switch s {
	case "advisory", "guided", "supervised", "autonomous":
		return true
	}
	return false
}

func isValidRiskFloor(s string) bool {
	switch s {
	case "low", "medium", "high", "critical":
		return true
	}
	return false
}

// toManifest converts an AgentDef to a runtime.AgentManifest.
func toManifest(def *AgentDef) (*runtime.AgentManifest, error) {
	tools := def.Tools
	if tools == nil {
		tools = []string{}
	}

	triggers := make([]runtime.TriggerDef, len(def.Triggers))
	for i, t := range def.Triggers {
		triggers[i] = runtime.TriggerDef{
			Type:          t.Type,
			Interval:      t.Interval,
			DecisionPoint: t.DecisionPoint,
		}
	}

	resourceLimits, err := toResourceLimits(def.ResourceLimits)
	if err != nil {
		return nil, err
	}

	memory, err := toMemoryConfig(def.Memory)
	if err != nil {
		return nil, err
	}

	return &runtime.AgentManifest{
		ID:             def.AgentID,
		Name:           def.Name,
		Squad:          def.Squad,
		Version:        def.Version,
		Description:    def.Description,
		AllowedTools:   tools,
		Triggers:       triggers,
		ResourceLimits: *resourceLimits,
		MemoryConfig:   *memory,
	}, nil
}

// toResourceLimits converts SDK-level resource limits to runtime-level.
func toResourceLimits(def ResourceLimitsDef) (*runtime.ResourceLimits, error) {
	var maxDecisionDuration time.Duration
	if def.MaxDecisionDuration != "" {
		var err error
		maxDecisionDuration, err = time.ParseDuration(def.MaxDecisionDuration)
		if err != nil {
			return nil, fmt.Errorf("parse max_decision_duration %q: %w", def.MaxDecisionDuration, err)
		}
	}

	return &runtime.ResourceLimits{
		MaxTokensPerMinute:  def.MaxTokensPerMinute,
		MaxTokensPerHour:    def.MaxTokensPerHour,
		MaxAPICallsPerMin:   def.MaxAPICallsPerMin,
		MaxAPICallsPerHour:  def.MaxAPICallsPerHour,
		MaxToolChainDepth:   def.MaxToolChainDepth,
		MaxDecisionDuration: maxDecisionDuration,
	}, nil
}

// toMemoryConfig converts SDK-level memory config to runtime-level.
func toMemoryConfig(def MemoryConfigDef) (*runtime.MemoryConfig, error) {
	var shortTermTTL time.Duration
	if def.ShortTermTTL != "" {
		var err error
		shortTermTTL, err = time.ParseDuration(def.ShortTermTTL)
		if err != nil {
			return nil, fmt.Errorf("parse short_term_ttl %q: %w", def.ShortTermTTL, err)
		}
	}

	var longTermTTL time.Duration
	if def.LongTermTTL != "" {
		var err error
		longTermTTL, err = time.ParseDuration(def.LongTermTTL)
		if err != nil {
			return nil, fmt.Errorf("parse long_term_ttl %q: %w", def.LongTermTTL, err)
		}
	}

	return &runtime.MemoryConfig{
		ShortTermTTL:    shortTermTTL,
		LongTermEnabled: def.LongTermEnabled,
		LongTermTTL:     longTermTTL,
	}, nil
}
