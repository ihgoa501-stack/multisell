package sdk

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/lingmirror/backend-go/internal/aios/runtime"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zap.NewNop()
}

func testBus(t *testing.T) *eventbus.Bus {
	t.Helper()
	return eventbus.New(testLogger(t))
}

func testRuntime(t *testing.T) *runtime.Runtime {
	t.Helper()
	return runtime.New(testLogger(t), testBus(t))
}

func validDef() *AgentDef {
	return &AgentDef{
		AgentID:     "A5",
		Name:        "Inventory Alert",
		Squad:       "fulfillment",
		Version:     "1.0.0",
		Description: "库存预警：缺货/补货/物流切换",
		DecisionPoints: []DecisionPointDef{
			{Name: "stock_alert", Description: "Detect low-stock SKUs"},
			{Name: "replenishment_plan", Description: "Generate replenishment plan"},
		},
		Tools:    []string{"stock_query", "inventory_forecast"},
		Triggers: []TriggerDef{
			{Type: "schedule", Interval: "5m", DecisionPoint: "stock_alert"},
		},
		ModelHint: "gpt-4o",
		Autonomy:  "supervised",
		RiskFloor: "medium",
		ResourceLimits: ResourceLimitsDef{
			MaxTokensPerMinute:  10000,
			MaxTokensPerHour:    100000,
			MaxDecisionDuration: "30s",
		},
		Memory: MemoryConfigDef{
			ShortTermTTL:    "5m",
			LongTermEnabled: false,
		},
	}
}

// yamlPath returns the absolute path to a YAML agent definition file.
// Tests run from the package directory, so we resolve relative to the go.mod root.
func yamlPath(t *testing.T, relative string) string {
	t.Helper()
	// Tests may run from either the package dir or the module root.
	// Try the common locations.
	candidates := []string{
		relative,
		filepath.Join("yaml", relative),
		filepath.Join("internal/aios/sdk/yaml", relative),
		filepath.Join("internal/aios/sdk/yaml/agents", relative),
	}
	// Also try from module root via os.Getwd
	if wd, err := os.Getwd(); err == nil {
		for _, c := range candidates {
			p := filepath.Join(wd, c)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	t.Fatalf("cannot find YAML file %q (tried from %v)", relative, candidates)
	return ""
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestValidate_ValidDef(t *testing.T) {
	err := Validate(validDef())
	if err != nil {
		t.Fatalf("Validate(valid def): %v", err)
	}
}

func TestValidate_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		def     *AgentDef
		wantMsg string
	}{
		{
			name:    "nil def",
			def:     nil,
			wantMsg: "agent definition is nil",
		},
		{
			name: "missing agent_id",
			def: &AgentDef{
				Name: "Test",
				Squad: "growth",
				Autonomy: "advisory",
				RiskFloor: "low",
				ModelHint: "gpt-4o",
				DecisionPoints: []DecisionPointDef{{Name: "test"}},
			},
			wantMsg: "agent_id is required",
		},
		{
			name: "missing name",
			def: &AgentDef{
				AgentID: "X1",
				Squad: "growth",
				Autonomy: "advisory",
				RiskFloor: "low",
				ModelHint: "gpt-4o",
				DecisionPoints: []DecisionPointDef{{Name: "test"}},
			},
			wantMsg: "name is required",
		},
		{
			name: "invalid squad",
			def: &AgentDef{
				AgentID: "X1", Name: "Test",
				Squad: "invalid",
				Autonomy: "advisory",
				RiskFloor: "low",
				ModelHint: "gpt-4o",
				DecisionPoints: []DecisionPointDef{{Name: "test"}},
			},
			wantMsg: "squad",
		},
		{
			name: "invalid autonomy",
			def: &AgentDef{
				AgentID: "X1", Name: "Test",
				Squad: "growth",
				Autonomy: "invalid",
				RiskFloor: "low",
				ModelHint: "gpt-4o",
				DecisionPoints: []DecisionPointDef{{Name: "test"}},
			},
			wantMsg: "autonomy",
		},
		{
			name: "invalid risk_floor",
			def: &AgentDef{
				AgentID: "X1", Name: "Test",
				Squad: "growth",
				Autonomy: "advisory",
				RiskFloor: "invalid",
				ModelHint: "gpt-4o",
				DecisionPoints: []DecisionPointDef{{Name: "test"}},
			},
			wantMsg: "risk_floor",
		},
		{
			name: "missing model_hint",
			def: &AgentDef{
				AgentID: "X1", Name: "Test",
				Squad: "growth",
				Autonomy: "advisory",
				RiskFloor: "low",
				DecisionPoints: []DecisionPointDef{{Name: "test"}},
			},
			wantMsg: "model_hint",
		},
		{
			name: "empty decision_points",
			def: &AgentDef{
				AgentID: "X1", Name: "Test",
				Squad: "growth",
				Autonomy: "advisory",
				RiskFloor: "low",
				ModelHint: "gpt-4o",
			},
			wantMsg: "decision_points",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.def)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestRegisterFromYAML(t *testing.T) {
	rt := testRuntime(t)

	// Register from the A5 YAML definition.
	err := RegisterFromYAML(rt, yamlPath(t, "agents/a5-stock-alert.yaml"))
	if err != nil {
		t.Fatalf("RegisterFromYAML: %v", err)
	}

	// Verify it was registered and started.
	inst, ok := rt.GetInstance("A5")
	if !ok {
		t.Fatal("GetInstance(A5) returned false after RegisterFromYAML")
	}
	if inst.State != runtime.StateReady {
		t.Fatalf("expected StateReady, got %s", inst.State)
	}
	if inst.Manifest.Name != "Inventory Alert" {
		t.Fatalf("expected Name 'Inventory Alert', got %q", inst.Manifest.Name)
	}
	if inst.Manifest.Squad != "fulfillment" {
		t.Fatalf("expected Squad 'fulfillment', got %q", inst.Manifest.Squad)
	}
}

func TestRoundTrip(t *testing.T) {
	// Read the A5 YAML, unmarshal into AgentDef, validate, convert to manifest.
	data, err := os.ReadFile(yamlPath(t, "agents/a5-stock-alert.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var def AgentDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	// Validate.
	if err := Validate(&def); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// Convert to manifest.
	manifest, err := toManifest(&def)
	if err != nil {
		t.Fatalf("toManifest: %v", err)
	}

	// Check field consistency.
	if manifest.ID != def.AgentID {
		t.Errorf("manifest.ID=%q, want %q", manifest.ID, def.AgentID)
	}
	if manifest.Name != def.Name {
		t.Errorf("manifest.Name=%q, want %q", manifest.Name, def.Name)
	}
	if manifest.Squad != def.Squad {
		t.Errorf("manifest.Squad=%q, want %q", manifest.Squad, def.Squad)
	}
	if manifest.Version != def.Version {
		t.Errorf("manifest.Version=%q, want %q", manifest.Version, def.Version)
	}
	if manifest.Description != def.Description {
		t.Errorf("manifest.Description=%q, want %q", manifest.Description, def.Description)
	}
	if len(manifest.Triggers) != len(def.Triggers) {
		t.Errorf("len(manifest.Triggers)=%d, want %d", len(manifest.Triggers), len(def.Triggers))
	}

	// Verify agent was registered successfully via the runtime.
	rt := testRuntime(t)
	if err := rt.RegisterAgent(*manifest); err != nil {
		t.Fatalf("rt.RegisterAgent: %v", err)
	}
	if err := rt.StartAgent(manifest.ID); err != nil {
		t.Fatalf("rt.StartAgent: %v", err)
	}
	inst, ok := rt.GetInstance(manifest.ID)
	if !ok {
		t.Fatal("GetInstance failed after registration")
	}
	if inst.State != runtime.StateReady {
		t.Fatalf("expected StateReady, got %s", inst.State)
	}
}

func TestRegisterMultipleYAML(t *testing.T) {
	// Register all three YAML agents.
	rt := testRuntime(t)

	for _, path := range []string{
		"agents/a5-stock-alert.yaml",
		"agents/g0-coordinator.yaml",
		"agents/a6-profit-watch.yaml",
	} {
		err := RegisterFromYAML(rt, yamlPath(t, path))
		if err != nil {
			t.Fatalf("RegisterFromYAML(%s): %v", path, err)
		}
	}

	// Verify all three are registered and running.
	for _, id := range []string{"A5", "G0", "A6"} {
		inst, ok := rt.GetInstance(id)
		if !ok {
			t.Fatalf("agent %q not found after registration", id)
		}
		if inst.State != runtime.StateReady {
			t.Fatalf("agent %q expected StateReady, got %s", id, inst.State)
		}
	}
}

func TestG0YAMLValidation(t *testing.T) {
	data, err := os.ReadFile(yamlPath(t, "agents/g0-coordinator.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var def AgentDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if err := Validate(&def); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// Check squad-specific validation.
	if def.Squad != "governance" {
		t.Errorf("Squad=%q, want %q", def.Squad, "governance")
	}
	if def.RiskFloor != "high" {
		t.Errorf("RiskFloor=%q, want %q", def.RiskFloor, "high")
	}
	if len(def.DecisionPoints) != 4 {
		t.Errorf("len(DecisionPoints)=%d, want %d", len(def.DecisionPoints), 4)
	}
}

func TestA6YAMLValidation(t *testing.T) {
	data, err := os.ReadFile(yamlPath(t, "agents/a6-profit-watch.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var def AgentDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if err := Validate(&def); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if def.Squad != "risk" {
		t.Errorf("Squad=%q, want %q", def.Squad, "risk")
	}
	if len(def.Triggers) != 3 {
		t.Errorf("len(Triggers)=%d, want %d", len(def.Triggers), 3)
	}
}
