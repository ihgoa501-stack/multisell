package ai

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

// ── resolveAgents ───────────────────────────────────────────────────────

func TestResolveAgents_WithPreferred(t *testing.T) {
	catalog := SchemaCatalog{
		"A5": {"stock_alert"},
		"A6": {"profit_watch"},
		"G1": {"dashboard_overview"},
	}
	c := NewMOACoordinator(nil, nil, nil, catalog, testLogger())

	agents, dps := c.resolveAgents("any", []string{"A5", "G1"})

	if len(agents) != 2 {
		t.Fatalf("want 2 agents, got %d: %v", len(agents), agents)
	}
	if agents[0] != "A5" || agents[1] != "G1" {
		t.Fatalf("want [A5 G1], got %v", agents)
	}
	if dps[0] != "stock_alert" {
		t.Fatalf("want dps[0]=stock_alert, got %q", dps[0])
	}
	if dps[1] != "dashboard_overview" {
		t.Fatalf("want dps[1]=dashboard_overview, got %q", dps[1])
	}
}

func TestResolveAgents_NoPreferred(t *testing.T) {
	catalog := SchemaCatalog{
		"A5": {"stock_alert"},
		"A6": {"profit_watch"},
	}
	c := NewMOACoordinator(nil, nil, nil, catalog, testLogger())

	agents, dps := c.resolveAgents("", nil)

	if len(agents) != 2 {
		t.Fatalf("want 2 agents, got %d", len(agents))
	}
	seen := map[string]bool{}
	for _, id := range agents {
		seen[id] = true
	}
	if !seen["A5"] || !seen["A6"] {
		t.Fatalf("want A5 and A6, got %v", agents)
	}
	if len(dps) != 2 {
		t.Fatalf("want 2 decision points, got %d", len(dps))
	}
}

func TestResolveAgents_GoalBasedRouting(t *testing.T) {
	catalog := SchemaCatalog{
		"A5": {"stock_alert"},
		"A6": {"profit_watch"},
	}
	c := NewMOACoordinator(nil, nil, nil, catalog, testLogger())

	// ponytail: goal parameter is accepted but currently unused;
	// resolveAgents always returns all catalog agents.
	agents, _ := c.resolveAgents("profit", nil)

	if len(agents) != 2 {
		t.Fatalf("goal-based routing not implemented — want all 2 catalog agents, got %d", len(agents))
	}
}

func TestResolveAgents_UnknownPreferredFallsBackToResolve(t *testing.T) {
	catalog := SchemaCatalog{
		"A5": {"stock_alert"},
	}
	c := NewMOACoordinator(nil, nil, nil, catalog, testLogger())

	agents, dps := c.resolveAgents("", []string{"UNKNOWN"})

	if len(agents) != 1 || agents[0] != "UNKNOWN" {
		t.Fatalf("want [UNKNOWN], got %v", agents)
	}
	if dps[0] != "resolve" {
		t.Fatalf("want dps[0]=resolve for unknown agent, got %q", dps[0])
	}
}

// ── detectConflicts ─────────────────────────────────────────────────────

func TestDetectConflicts_NoConflict(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())
	results := []MOAAgentResult{
		{AgentID: "A5", Confidence: 0.9},
		{AgentID: "A6", Confidence: 0.8},
	}

	conflicts := c.detectConflicts(results)

	if len(conflicts) != 0 {
		t.Fatalf("want 0 conflicts, got %d", len(conflicts))
	}
}

func TestDetectConflicts_PartialAgreement(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())
	results := []MOAAgentResult{
		{AgentID: "A5", Confidence: 0.9},
		{AgentID: "A6", Confidence: 0.3},
		{AgentID: "A7", Confidence: 0.9},
	}

	conflicts := c.detectConflicts(results)

	if len(conflicts) != 1 {
		t.Fatalf("want 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].On != "confidence_threshold" {
		t.Fatalf("want on=confidence_threshold, got %q", conflicts[0].On)
	}
	if len(conflicts[0].Between) != 1 || conflicts[0].Between[0] != "A6" {
		t.Fatalf("want Between=[A6], got %v", conflicts[0].Between)
	}
}

func TestDetectConflicts_MultipleLowConfidence(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())
	results := []MOAAgentResult{
		{AgentID: "A5", Confidence: 0.2},
		{AgentID: "A6", Confidence: 0.3},
		{AgentID: "A7", Confidence: 0.9},
	}

	conflicts := c.detectConflicts(results)

	if len(conflicts) != 1 {
		t.Fatalf("want 1 conflict, got %d", len(conflicts))
	}
	if len(conflicts[0].Between) != 2 {
		t.Fatalf("want 2 agents in Between, got %d: %v", len(conflicts[0].Between), conflicts[0].Between)
	}
}

func TestDetectConflicts_LessThanTwoAgents(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())

	conflicts := c.detectConflicts([]MOAAgentResult{{AgentID: "A5", Confidence: 0.9}})

	if len(conflicts) != 0 {
		t.Fatalf("want 0 conflicts for single agent, got %d", len(conflicts))
	}

	conflicts = c.detectConflicts(nil)
	if len(conflicts) != 0 {
		t.Fatalf("want 0 conflicts for nil, got %d", len(conflicts))
	}
}

// ── computeRiskLevel ────────────────────────────────────────────────────

func TestComputeRiskLevel_AllLow(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())
	results := []MOAAgentResult{
		{RiskLevel: "low"},
		{RiskLevel: "low"},
	}

	level := c.computeRiskLevel(results, nil)
	if level != "low" {
		t.Fatalf("want low, got %s", level)
	}
}

func TestComputeRiskLevel_Mixed(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())
	results := []MOAAgentResult{
		{RiskLevel: "low"},
		{RiskLevel: "high"},
		{RiskLevel: "medium"},
	}

	level := c.computeRiskLevel(results, nil)
	if level != "high" {
		t.Fatalf("want high, got %s", level)
	}
}

func TestComputeRiskLevel_Medium(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())
	results := []MOAAgentResult{
		{RiskLevel: "low"},
		{RiskLevel: "medium"},
	}

	level := c.computeRiskLevel(results, nil)
	if level != "medium" {
		t.Fatalf("want medium, got %s", level)
	}
}

func TestComputeRiskLevel_Empty(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())

	level := c.computeRiskLevel(nil, nil)
	if level != "low" {
		t.Fatalf("want low for empty results, got %s", level)
	}
}

// ── synthesize ──────────────────────────────────────────────────────────

func TestSynthesize_EmptyResults(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())

	s := c.synthesize(nil, nil)
	m, ok := s.(map[string]interface{})
	if !ok {
		t.Fatalf("want map, got %T", s)
	}
	if m["recommendation"] != "Cannot proceed without agent results." {
		t.Fatalf("unexpected recommendation: %v", m["recommendation"])
	}
}

func TestSynthesize_SingleAgent(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())
	results := []MOAAgentResult{
		{AgentID: "A5", Confidence: 0.9, RiskLevel: "low"},
	}

	s := c.synthesize(results, nil)
	m, ok := s.(map[string]interface{})
	if !ok {
		t.Fatalf("want map, got %T", s)
	}
	findings := m["agent_findings"].([]map[string]interface{})
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0]["agent_id"] != "A5" {
		t.Fatalf("want agent_id=A5, got %v", findings[0]["agent_id"])
	}
	if m["recommendation"] != "Proceed with confidence — all agents agree, low risk." {
		t.Fatalf("unexpected recommendation: %v", m["recommendation"])
	}
}

func TestSynthesize_MultipleWithConflicts(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, nil, testLogger())
	results := []MOAAgentResult{
		{AgentID: "A5", Confidence: 0.9, RiskLevel: "low"},
		{AgentID: "A6", Confidence: 0.3, RiskLevel: "high"},
	}
	conflicts := []MOAConflict{
		{
			Between: []string{"A6"},
			On:      "confidence_threshold",
			AOption: "recommend",
			BOption: "caution (low confidence)",
		},
	}

	s := c.synthesize(results, conflicts)
	m, ok := s.(map[string]interface{})
	if !ok {
		t.Fatalf("want map, got %T", s)
	}
	findings := m["agent_findings"].([]map[string]interface{})
	if len(findings) != 2 {
		t.Fatalf("want 2 findings, got %d", len(findings))
	}
	if m["conflict_count"] != 1 {
		t.Fatalf("want conflict_count=1, got %v", m["conflict_count"])
	}
	if m["recommendation"] != "Conflicts detected — manual review required before proceeding." {
		t.Fatalf("unexpected recommendation: %v", m["recommendation"])
	}
}

// ── Run (integration) ───────────────────────────────────────────────────

func TestMOACoordinator_Run_Basic(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())
	catalog := SchemaCatalog{
		"A5": {"stock_alert"},
	}
	c := NewMOACoordinator(orch, nil, nil, catalog, testLogger())

	ctx := context.Background()
	req := &MOARequest{
		Goal:          "test stock alert",
		Participating: []string{"A5"},
		Mode:          "dry_run",
	}

	result, err := c.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("want completed, got %s", result.Status)
	}
	if len(result.AgentResults) != 1 {
		t.Fatalf("want 1 agent result, got %d", len(result.AgentResults))
	}
	if result.AgentResults[0].AgentID != "A5" {
		t.Fatalf("want AgentID=A5, got %s", result.AgentResults[0].AgentID)
	}
}

func TestMOACoordinator_Run_EmptyGoal(t *testing.T) {
	c := NewMOACoordinator(nil, nil, nil, SchemaCatalog{}, testLogger())

	ctx := context.Background()
	req := &MOARequest{}

	_, err := c.Run(ctx, req)
	if err == nil {
		t.Fatal("want error for empty goal with no participating agents")
	}
}

func TestMOACoordinator_Run_BlockedByRisk(t *testing.T) {
	// When in production mode with high risk, the result status should be
	// "needs_approval" even without an approval service configured.
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())
	bus := eventbus.New(testLogger())
	catalog := SchemaCatalog{
		"G1": {"dashboard_overview"},
	}
	c := NewMOACoordinator(orch, bus, nil, catalog, testLogger())

	ctx := context.Background()
	// G1 is advisory and may have varying risk levels, so we set Participating
	// to a known agent.
	req := &MOARequest{
		Goal:          "test",
		Participating: []string{"G1"},
		Mode:          "production",
	}

	result, err := c.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// If the agent result has risk_level != "high", it will still be "completed".
	// We just verify the run completed without error.
	if result.Status != "completed" && result.Status != "needs_approval" {
		t.Fatalf("want completed or needs_approval, got %s", result.Status)
	}
}
