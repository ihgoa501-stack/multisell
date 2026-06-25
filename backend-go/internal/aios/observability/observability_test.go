package observability

import (
	"math"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func testLogger(t *testing.T) *zap.Logger {
	t.Helper()
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("NewDevelopment: %v", err)
	}
	return logger
}

func fixedTime() time.Time {
	return time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
}

// ---------------------------------------------------------------------------
// Record / aggregation
// ---------------------------------------------------------------------------

func TestRecordAndQuery(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	metrics := AgentMetrics{
		AgentID:           "A5",
		PeriodStart:       now.Add(-5 * time.Minute),
		PeriodEnd:         now,
		DecisionsMade:     10,
		ActionsCreated:    5,
		ActionsApproved:   4,
		ActionsRejected:   1,
		ActionsExecuted:   4,
		AverageConfidence: 0.85,
		AvgLatencyMs:      120,
		SuccessRate:       0.90,
		HighRiskActions:   1,
		TokensUsed:        15000,
		EstimatedCostUsd:  0.45,
		ToolCallsMade:     12,
	}
	c.Record(metrics)

	result := c.Query("A5", now.Add(-1*time.Hour))
	if result == nil {
		t.Fatal("Query returned nil, expected metrics")
	}
	if result.AgentID != "A5" {
		t.Errorf("AgentID = %q, want %q", result.AgentID, "A5")
	}
	if result.DecisionsMade != 10 {
		t.Errorf("DecisionsMade = %d, want %d", result.DecisionsMade, 10)
	}
	if result.AverageConfidence != 0.85 {
		t.Errorf("AverageConfidence = %f, want %f", result.AverageConfidence, 0.85)
	}
}

func TestRecordAggregation(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	c.Record(AgentMetrics{
		AgentID:           "A5",
		PeriodStart:       now.Add(-10 * time.Minute),
		PeriodEnd:         now.Add(-5 * time.Minute),
		DecisionsMade:     10,
		ActionsCreated:    6,
		ActionsApproved:   5,
		ActionsRejected:   1,
		ActionsExecuted:   6,
		AverageConfidence: 0.85,
		AvgLatencyMs:      120,
		SuccessRate:       0.90,
		HighRiskActions:   1,
		TokensUsed:        15000,
		EstimatedCostUsd:  0.45,
		ToolCallsMade:     12,
	})
	c.Record(AgentMetrics{
		AgentID:           "A5",
		PeriodStart:       now.Add(-5 * time.Minute),
		PeriodEnd:         now,
		DecisionsMade:     20,
		ActionsCreated:    10,
		ActionsApproved:   8,
		ActionsRejected:   2,
		ActionsExecuted:   10,
		AverageConfidence: 0.92,
		AvgLatencyMs:      80,
		SuccessRate:       0.95,
		HighRiskActions:   0,
		TokensUsed:        25000,
		EstimatedCostUsd:  0.75,
		ToolCallsMade:     20,
	})

	result := c.Query("A5", now.Add(-1*time.Hour))
	if result == nil {
		t.Fatal("Query returned nil")
	}

	// Volume fields are summed.
	if result.DecisionsMade != 30 {
		t.Errorf("DecisionsMade = %d, want %d", result.DecisionsMade, 30)
	}
	if result.ActionsCreated != 16 {
		t.Errorf("ActionsCreated = %d, want %d", result.ActionsCreated, 16)
	}
	if result.ActionsApproved != 13 {
		t.Errorf("ActionsApproved = %d, want %d", result.ActionsApproved, 13)
	}
	if result.ActionsRejected != 3 {
		t.Errorf("ActionsRejected = %d, want %d", result.ActionsRejected, 3)
	}
	if result.ActionsExecuted != 16 {
		t.Errorf("ActionsExecuted = %d, want %d", result.ActionsExecuted, 16)
	}
	if result.TokensUsed != 40000 {
		t.Errorf("TokensUsed = %d, want %d", result.TokensUsed, 40000)
	}
	if result.ToolCallsMade != 32 {
		t.Errorf("ToolCallsMade = %d, want %d", result.ToolCallsMade, 32)
	}
	if result.HighRiskActions != 1 {
		t.Errorf("HighRiskActions = %d, want %d", result.HighRiskActions, 1)
	}

	// Quality/cost fields are averaged.
	expectedConf := (0.85 + 0.92) / 2.0
	if math.Abs(result.AverageConfidence-expectedConf) > 0.001 {
		t.Errorf("AverageConfidence = %f, want %f", result.AverageConfidence, expectedConf)
	}
	expectedLatency := (120 + 80) / 2
	if result.AvgLatencyMs != expectedLatency {
		t.Errorf("AvgLatencyMs = %d, want %d", result.AvgLatencyMs, expectedLatency)
	}
	expectedRate := (0.90 + 0.95) / 2.0
	if math.Abs(result.SuccessRate-expectedRate) > 0.001 {
		t.Errorf("SuccessRate = %f, want %f", result.SuccessRate, expectedRate)
	}
	expectedCost := 0.45 + 0.75
	if math.Abs(result.EstimatedCostUsd-expectedCost) > 0.001 {
		t.Errorf("EstimatedCostUsd = %f, want %f", result.EstimatedCostUsd, expectedCost)
	}
}

// ---------------------------------------------------------------------------
// Query — time range filtering
// ---------------------------------------------------------------------------

func TestQueryTimeRangeFilter(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	// Record an old event (outside range).
	c.Record(AgentMetrics{
		AgentID:           "A5",
		PeriodStart:       now.Add(-2 * time.Hour),
		PeriodEnd:         now.Add(-90 * time.Minute),
		DecisionsMade:     5,
		AverageConfidence: 0.80,
	})
	// Record a recent event (inside range).
	c.Record(AgentMetrics{
		AgentID:           "A5",
		PeriodStart:       now.Add(-30 * time.Minute),
		PeriodEnd:         now,
		DecisionsMade:     8,
		AverageConfidence: 0.90,
	})

	// Query with a recent since time — should only see the recent event.
	result := c.Query("A5", now.Add(-1*time.Hour))
	if result == nil {
		t.Fatal("Query returned nil for recent range")
	}
	if result.DecisionsMade != 8 {
		t.Errorf("DecisionsMade = %d, want 8 (only recent event)", result.DecisionsMade)
	}

	// Query with a broader since time — should see both events.
	result = c.Query("A5", now.Add(-3*time.Hour))
	if result == nil {
		t.Fatal("Query returned nil for broad range")
	}
	if result.DecisionsMade != 13 {
		t.Errorf("DecisionsMade = %d, want 13 (both events)", result.DecisionsMade)
	}

	// Query for a non-existent agent.
	result = c.Query("NONEXISTENT", now.Add(-3*time.Hour))
	if result != nil {
		t.Errorf("expected nil for non-existent agent, got %+v", result)
	}
}

// ---------------------------------------------------------------------------
// QueryAll
// ---------------------------------------------------------------------------

func TestQueryAll(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	c.Record(AgentMetrics{AgentID: "A5", PeriodStart: now.Add(-5 * time.Minute), PeriodEnd: now, DecisionsMade: 10})
	c.Record(AgentMetrics{AgentID: "A6", PeriodStart: now.Add(-5 * time.Minute), PeriodEnd: now, DecisionsMade: 20})
	c.Record(AgentMetrics{AgentID: "A5", PeriodStart: now.Add(-3 * time.Minute), PeriodEnd: now, DecisionsMade: 5})

	results := c.QueryAll(now.Add(-1 * time.Hour))
	if len(results) != 2 {
		t.Fatalf("expected 2 agent groups, got %d", len(results))
	}

	// Results should be sorted by AgentID.
	if results[0].AgentID != "A5" {
		t.Errorf("results[0].AgentID = %q, want %q", results[0].AgentID, "A5")
	}
	if results[0].DecisionsMade != 15 {
		t.Errorf("results[0].DecisionsMade = %d, want %d", results[0].DecisionsMade, 15)
	}
	if results[1].AgentID != "A6" {
		t.Errorf("results[1].AgentID = %q, want %q", results[1].AgentID, "A6")
	}
	if results[1].DecisionsMade != 20 {
		t.Errorf("results[1].DecisionsMade = %d, want %d", results[1].DecisionsMade, 20)
	}
}

func TestQueryAllEmptyCollector(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	results := c.QueryAll(time.Now())
	if results == nil || len(results) != 0 {
		t.Errorf("expected empty slice, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// QueryDecisionPoint
// ---------------------------------------------------------------------------

func TestQueryDecisionPoint(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	c.recordDecisionObservation("stock_alert", "A5", 0.85, 120, true, "", now.Add(-30*time.Minute))
	c.recordDecisionObservation("stock_alert", "A5", 0.92, 100, true, "", now.Add(-20*time.Minute))
	c.recordDecisionObservation("stock_alert", "A5", 0.45, 500, false, "timeout", now.Add(-10*time.Minute))
	c.recordDecisionObservation("replenish", "A5", 0.90, 150, true, "", now.Add(-5*time.Minute))

	// Query all stock_alert observations.
	stats := c.QueryDecisionPoint("stock_alert", now.Add(-1*time.Hour))
	if stats == nil {
		t.Fatal("QueryDecisionPoint returned nil")
	}
	if stats.DecisionPoint != "stock_alert" {
		t.Errorf("DecisionPoint = %q, want %q", stats.DecisionPoint, "stock_alert")
	}
	if stats.TotalExecutions != 3 {
		t.Errorf("TotalExecutions = %d, want %d", stats.TotalExecutions, 3)
	}
	expectedConf := (0.85 + 0.92 + 0.45) / 3.0
	if math.Abs(stats.AverageConfidence-expectedConf) > 0.001 {
		t.Errorf("AverageConfidence = %f, want %f", stats.AverageConfidence, expectedConf)
	}
	expectedLatency := (120 + 100 + 500) / 3
	if stats.AverageLatencyMs != expectedLatency {
		t.Errorf("AverageLatencyMs = %d, want %d", stats.AverageLatencyMs, expectedLatency)
	}
	expectedRate := 2.0 / 3.0
	if math.Abs(stats.SuccessRate-expectedRate) > 0.001 {
		t.Errorf("SuccessRate = %f, want %f", stats.SuccessRate, expectedRate)
	}
	if stats.FailureBreakdown["timeout"] != 1 {
		t.Errorf("FailureBreakdown[timeout] = %d, want %d", stats.FailureBreakdown["timeout"], 1)
	}

	// Time range filtering — should return nothing if out of range.
	stats = c.QueryDecisionPoint("stock_alert", now.Add(-1*time.Minute))
	if stats != nil {
		t.Errorf("expected nil for out-of-range query, got %+v", stats)
	}

	// Non-existent decision point.
	stats = c.QueryDecisionPoint("nonexistent", now.Add(-1*time.Hour))
	if stats != nil {
		t.Errorf("expected nil for non-existent decision point, got %+v", stats)
	}
}

// ---------------------------------------------------------------------------
// QueryCost
// ---------------------------------------------------------------------------

func TestQueryCostByAgent(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	c.Record(AgentMetrics{
		AgentID:          "A5",
		PeriodStart:      now.Add(-10 * time.Minute),
		PeriodEnd:        now,
		TokensUsed:       15000,
		EstimatedCostUsd: 0.45,
		ToolCallsMade:    12,
	})
	c.Record(AgentMetrics{
		AgentID:          "A5",
		PeriodStart:      now.Add(-5 * time.Minute),
		PeriodEnd:        now,
		TokensUsed:       25000,
		EstimatedCostUsd: 0.75,
		ToolCallsMade:    20,
	})
	c.Record(AgentMetrics{
		AgentID:          "A6",
		PeriodStart:      now.Add(-5 * time.Minute),
		PeriodEnd:        now,
		TokensUsed:       10000,
		EstimatedCostUsd: 0.30,
		ToolCallsMade:    8,
	})

	rows := c.QueryCost("agent", now.Add(-1*time.Hour))
	if len(rows) != 2 {
		t.Fatalf("expected 2 cost rows, got %d", len(rows))
	}

	for _, r := range rows {
		switch r.Dimension {
		case "A5":
			if math.Abs(r.TotalCost-1.20) > 0.001 {
				t.Errorf("A5 TotalCost = %f, want %f", r.TotalCost, 1.20)
			}
			if r.TotalTokens != 40000 {
				t.Errorf("A5 TotalTokens = %d, want %d", r.TotalTokens, 40000)
			}
			if r.CallCount != 32 {
				t.Errorf("A5 CallCount = %d, want %d", r.CallCount, 32)
			}
		case "A6":
			if math.Abs(r.TotalCost-0.30) > 0.001 {
				t.Errorf("A6 TotalCost = %f, want %f", r.TotalCost, 0.30)
			}
			if r.TotalTokens != 10000 {
				t.Errorf("A6 TotalTokens = %d, want %d", r.TotalTokens, 10000)
			}
			if r.CallCount != 8 {
				t.Errorf("A6 CallCount = %d, want %d", r.CallCount, 8)
			}
		default:
			t.Errorf("unexpected dimension %q", r.Dimension)
		}
	}
}

func TestQueryCostByDecisionPoint(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	c.recordDecisionObservation("stock_alert", "A5", 0.85, 100, true, "", now.Add(-30*time.Minute))
	c.recordDecisionObservation("stock_alert", "A5", 0.92, 80, true, "", now.Add(-20*time.Minute))
	c.recordDecisionObservation("replenish", "A5", 0.90, 150, true, "", now.Add(-10*time.Minute))

	rows := c.QueryCost("decision_point", now.Add(-1*time.Hour))
	if len(rows) != 2 {
		t.Fatalf("expected 2 cost rows, got %d", len(rows))
	}

	for _, r := range rows {
		switch r.Dimension {
		case "stock_alert":
			if r.CallCount != 2 {
				t.Errorf("stock_alert CallCount = %d, want %d", r.CallCount, 2)
			}
		case "replenish":
			if r.CallCount != 1 {
				t.Errorf("replenish CallCount = %d, want %d", r.CallCount, 1)
			}
		default:
			t.Errorf("unexpected dimension %q", r.Dimension)
		}
	}
}

func TestQueryCostDefaultGroupBy(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	c.Record(AgentMetrics{
		AgentID:          "A5",
		PeriodStart:      now.Add(-5 * time.Minute),
		PeriodEnd:        now,
		TokensUsed:       15000,
		EstimatedCostUsd: 0.45,
		ToolCallsMade:    12,
	})

	// Unknown groupBy should default to "agent".
	rows := c.QueryCost("squad", now.Add(-1*time.Hour))
	if len(rows) != 1 {
		t.Fatalf("expected 1 row with default grouping, got %d", len(rows))
	}
	if rows[0].Dimension != "A5" {
		t.Errorf("Dimension = %q, want %q", rows[0].Dimension, "A5")
	}
}

// ---------------------------------------------------------------------------
// ScanAnomalies
// ---------------------------------------------------------------------------

func TestScanAnomalies_ConfidenceDrop(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	// Record 5 normal events with high confidence.
	for i := 0; i < 5; i++ {
		c.Record(AgentMetrics{
			AgentID:           "A5",
			PeriodStart:       now.Add(-time.Duration(10-i) * time.Minute),
			PeriodEnd:         now.Add(-time.Duration(10-i-1) * time.Minute),
			AverageConfidence: 0.92 + float64(i)*0.01,
			HighRiskActions:   1,
		})
	}

	// Record a clear outlier — very low confidence.
	c.Record(AgentMetrics{
		AgentID:           "A5",
		PeriodStart:       now.Add(-1 * time.Minute),
		PeriodEnd:         now,
		AverageConfidence: 0.15,
		HighRiskActions:   10,
	})

	reports := c.ScanAnomalies(1.5)
	if len(reports) == 0 {
		t.Fatal("ScanAnomalies returned no reports, expected at least confidence_drop")
	}

	foundConfidenceDrop := false
	foundRiskSpike := false
	for _, r := range reports {
		if r.Type == "confidence_drop" {
			foundConfidenceDrop = true
			if r.Severity != "warning" && r.Severity != "critical" {
				t.Errorf("unexpected severity %q", r.Severity)
			}
			if r.AgentID != "A5" {
				t.Errorf("expected agent A5, got %s", r.AgentID)
			}
		}
		if r.Type == "risk_spike" {
			foundRiskSpike = true
		}
	}
	if !foundConfidenceDrop {
		t.Errorf("expected confidence_drop anomaly, got types: %v", anomalyTypes(reports))
	}
	if !foundRiskSpike {
		t.Errorf("expected risk_spike anomaly, got types: %v", anomalyTypes(reports))
	}
}

func TestScanAnomalies_InsufficientData(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	c.Record(AgentMetrics{AgentID: "A5", PeriodStart: now.Add(-5 * time.Minute), PeriodEnd: now, AverageConfidence: 0.95})
	c.Record(AgentMetrics{AgentID: "A5", PeriodStart: now.Add(-3 * time.Minute), PeriodEnd: now, AverageConfidence: 0.20})

	// Only 2 events — not enough for a baseline.
	reports := c.ScanAnomalies(1.5)
	if len(reports) != 0 {
		t.Errorf("expected 0 reports with <3 events, got %d", len(reports))
	}
}

func TestScanAnomalies_NormalBehavior(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	now := fixedTime()

	// All events have similar confidence and low risk.
	for i := 0; i < 5; i++ {
		c.Record(AgentMetrics{
			AgentID:           "A5",
			PeriodStart:       now.Add(-time.Duration(10-i) * time.Minute),
			PeriodEnd:         now.Add(-time.Duration(10-i-1) * time.Minute),
			AverageConfidence: 0.90 + float64(i)*0.02,
			HighRiskActions:   0,
		})
	}

	reports := c.ScanAnomalies(1.5)
	if len(reports) != 0 {
		t.Errorf("expected 0 reports for normal behavior, got %d: %v", len(reports), anomalyTypes(reports))
	}
}

func TestScanAnomalies_EmptyCollector(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	reports := c.ScanAnomalies(1.5)
	if len(reports) != 0 {
		t.Errorf("expected 0 reports for empty collector, got %d", len(reports))
	}
}

func anomalyTypes(reports []AnomalyReport) []string {
	types := make([]string, len(reports))
	for i, r := range reports {
		types[i] = r.Type
	}
	return types
}

// ---------------------------------------------------------------------------
// TraceLink tree building
// ---------------------------------------------------------------------------

func TestBuildTraceTree_SimpleChain(t *testing.T) {
	now := fixedTime()
	links := []TraceLink{
		{
			RootTraceID:   "trace1",
			ParentTraceID: "",
			ChildTraceID:  "span1",
			FromAgent:     "G3",
			ToAgent:       "A5",
			Action:        "discount_risk_check",
			StartedAt:     now,
			DurationMs:    200,
		},
		{
			RootTraceID:   "trace1",
			ParentTraceID: "span1",
			ChildTraceID:  "span2",
			FromAgent:     "A5",
			ToAgent:       "A6",
			Action:        "profit_watch",
			StartedAt:     now.Add(200 * time.Millisecond),
			DurationMs:    150,
		},
		{
			RootTraceID:   "trace1",
			ParentTraceID: "span2",
			ChildTraceID:  "span3",
			FromAgent:     "A6",
			ToAgent:       "A2",
			Action:        "listing_optimize",
			StartedAt:     now.Add(350 * time.Millisecond),
			DurationMs:    300,
		},
	}

	tree := BuildTraceTree(links)
	if tree == nil {
		t.Fatal("BuildTraceTree returned nil")
	}

	// Root should be G3 -> A5.
	if tree.Root.FromAgent != "G3" {
		t.Errorf("root FromAgent = %q, want %q", tree.Root.FromAgent, "G3")
	}
	if tree.Root.ToAgent != "A5" {
		t.Errorf("root ToAgent = %q, want %q", tree.Root.ToAgent, "A5")
	}
	if tree.Root.ChildTraceID != "span1" {
		t.Errorf("root ChildTraceID = %q, want %q", tree.Root.ChildTraceID, "span1")
	}

	// Should have one child: A5 -> A6.
	if len(tree.Children) != 1 {
		t.Fatalf("expected 1 child at root, got %d", len(tree.Children))
	}
	child1 := tree.Children[0]
	if child1.Root.FromAgent != "A5" {
		t.Errorf("child FromAgent = %q, want %q", child1.Root.FromAgent, "A5")
	}
	if child1.Root.ToAgent != "A6" {
		t.Errorf("child ToAgent = %q, want %q", child1.Root.ToAgent, "A6")
	}

	// That child should have its own child: A6 -> A2.
	if len(child1.Children) != 1 {
		t.Fatalf("expected 1 child at depth 2, got %d", len(child1.Children))
	}
	child2 := child1.Children[0]
	if child2.Root.FromAgent != "A6" {
		t.Errorf("grandchild FromAgent = %q, want %q", child2.Root.FromAgent, "A6")
	}
	if child2.Root.ToAgent != "A2" {
		t.Errorf("grandchild ToAgent = %q, want %q", child2.Root.ToAgent, "A2")
	}
	if child2.Children != nil {
		t.Errorf("expected no children at leaf, got %d", len(child2.Children))
	}
}

func TestBuildTraceTree_Fork(t *testing.T) {
	now := fixedTime()
	links := []TraceLink{
		{
			RootTraceID:   "trace2",
			ParentTraceID: "",
			ChildTraceID:  "root1",
			FromAgent:     "G1",
			ToAgent:       "A5",
			Action:        "stock_check",
			StartedAt:     now,
			DurationMs:    100,
		},
		{
			RootTraceID:   "trace2",
			ParentTraceID: "root1",
			ChildTraceID:  "child_a",
			FromAgent:     "A5",
			ToAgent:       "A3",
			Action:        "price_review",
			StartedAt:     now.Add(100 * time.Millisecond),
			DurationMs:    80,
		},
		{
			RootTraceID:   "trace2",
			ParentTraceID: "root1",
			ChildTraceID:  "child_b",
			FromAgent:     "A5",
			ToAgent:       "A6",
			Action:        "profit_watch",
			StartedAt:     now.Add(100 * time.Millisecond),
			DurationMs:    120,
		},
	}

	tree := BuildTraceTree(links)
	if tree == nil {
		t.Fatal("BuildTraceTree returned nil")
	}
	if len(tree.Children) != 2 {
		t.Fatalf("expected 2 children (fork), got %d", len(tree.Children))
	}

	// Should have two parallel children: A3 and A6.
	childAgents := make(map[string]string)
	for _, child := range tree.Children {
		childAgents[child.Root.ToAgent] = child.Root.Action
	}
	if _, ok := childAgents["A3"]; !ok {
		t.Errorf("expected child to A3 (price_review), got children for: %v", childAgents)
	}
	if _, ok := childAgents["A6"]; !ok {
		t.Errorf("expected child to A6 (profit_watch), got children for: %v", childAgents)
	}
}

func TestBuildTraceTree_Empty(t *testing.T) {
	tree := BuildTraceTree(nil)
	if tree != nil {
		t.Errorf("expected nil for empty input, got %+v", tree)
	}
}

func TestBuildTraceTree_SingleLink(t *testing.T) {
	links := []TraceLink{
		{
			RootTraceID:   "trace3",
			ParentTraceID: "",
			ChildTraceID:  "span1",
			FromAgent:     "A5",
			ToAgent:       "A6",
			Action:        "handoff",
			StartedAt:     fixedTime(),
			DurationMs:    50,
		},
	}
	tree := BuildTraceTree(links)
	if tree == nil {
		t.Fatal("BuildTraceTree returned nil for single link")
	}
	if tree.Root.FromAgent != "A5" {
		t.Errorf("FromAgent = %q, want %q", tree.Root.FromAgent, "A5")
	}
	if tree.Children != nil {
		t.Errorf("expected no children for single link, got %d children", len(tree.Children))
	}
}

// ---------------------------------------------------------------------------
// Collector — edge cases
// ---------------------------------------------------------------------------

func TestCollector_NilLogger(t *testing.T) {
	c := NewCollector(nil)
	if c == nil {
		t.Fatal("NewCollector returned nil")
	}
	// Should not panic on Record with nil logger.
	now := fixedTime()
	c.Record(AgentMetrics{AgentID: "A5", PeriodStart: now.Add(-5 * time.Minute), PeriodEnd: now, DecisionsMade: 1})
	result := c.Query("A5", now.Add(-1*time.Hour))
	if result == nil {
		t.Errorf("expected non-nil result with nil logger")
	}
}

func TestCollector_QueryNoMatch(t *testing.T) {
	logger := testLogger(t)
	c := NewCollector(logger)
	result := c.Query("A5", time.Now())
	if result != nil {
		t.Errorf("expected nil for no matching data, got %+v", result)
	}
}
