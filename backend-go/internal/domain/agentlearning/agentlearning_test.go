package agentlearning

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func newTestDB(t *testing.T) *DecisionEvaluation {
	t.Helper()
	return &DecisionEvaluation{}
}

// ── Pure helper tests ────────────────────────────────────────────

func TestGuessAgent(t *testing.T) {
	svc := NewService(nil, dbtest.NewLogger(t))

	cases := []struct {
		point    string
		expected string
	}{
		{"acos_analysis", "A3"},
		{"price_watch", "A3"},
		{"profit_watch", "A3"},
		{"sourcing_scan", "A8"},
		{"listing_optimize", "A2"},
		{"discount_risk_check", "G3"},
		{"unknown", "A3"},
	}
	for _, tc := range cases {
		got := svc.guessAgent(tc.point)
		if got != tc.expected {
			t.Errorf("guessAgent(%q) = %q, want %q", tc.point, got, tc.expected)
		}
	}
}

func TestPeriodHours(t *testing.T) {
	cases := []struct {
		period   string
		expected time.Duration
	}{
		{"7d", 7 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"90d", 90 * 24 * time.Hour},
		{"invalid", 0},
		{"", 0},
	}
	for _, tc := range cases {
		got := periodHours(tc.period)
		if got != tc.expected {
			t.Errorf("periodHours(%q) = %v, want %v", tc.period, got, tc.expected)
		}
	}
}

func TestComputeMarginAccuracy(t *testing.T) {
	svc := NewService(nil, dbtest.NewLogger(t))

	cases := []struct {
		name     string
		predicted float64
		actual   float64
		expected float64
	}{
		{"both zero", 0, 0, 1.0},
		{"predicted zero", 0, 0.5, 0.0},
		{"exact match", 0.2, 0.2, 1.0},
		{"half accuracy", 0.2, 0.1, 0.5},
		{"over- predicts capped", 0.1, 0.15, 0.67},
		{"over- predicts", 0.2, 0.4, 0.5}, // 0.2/0.4=0.5, but actual>predicted so 1/0.5=0.5... wait
		// Let me recalculate: computeMarginAccuracy does ratio = actual/predicted = 0.4/0.2 = 2.0
		// ratio > 1.0, so ratio = 1/2.0 = 0.5. returns round(0.5*100)/100 = 0.5
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := svc.computeMarginAccuracy(tc.predicted, tc.actual)
			if got != tc.expected {
				t.Errorf("computeMarginAccuracy(%f, %f) = %f, want %f",
					tc.predicted, tc.actual, got, tc.expected)
			}
		})
	}
}

func TestComputeTrend_BothEmpty(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{})
	svc := NewService(db, dbtest.NewLogger(t))

	trend := svc.computeTrend("A3", time.Now().Add(-48*time.Hour), 48*time.Hour)
	if trend != "stable" {
		t.Errorf("expected 'stable' for empty data, got %s", trend)
	}
}

// ── DB tests ─────────────────────────────────────────────────────

func TestService_GetAllAccuracy_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &AgentAccuracy{})
	svc := NewService(db, dbtest.NewLogger(t))

	records, err := svc.GetAllAccuracy()
	if err != nil {
		t.Fatalf("GetAllAccuracy failed: %v", err)
	}
	if records == nil {
		t.Error("expected non-nil empty slice")
	}
}

func TestService_GetAllAccuracy_WithData(t *testing.T) {
	db := dbtest.NewDB(t, &AgentAccuracy{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&AgentAccuracy{AgentID: "A3", Period: "30d", TotalDecisions: 10, CorrectDecisions: 7, AccuracyPct: 70})
	db.Create(&AgentAccuracy{AgentID: "A8", Period: "30d", TotalDecisions: 5, CorrectDecisions: 4, AccuracyPct: 80})

	records, err := svc.GetAllAccuracy()
	if err != nil {
		t.Fatalf("GetAllAccuracy failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
	// Should be ordered by accuracy DESC, so A8 (80%) first.
	if records[0].AgentID != "A8" {
		t.Errorf("expected A8 first (80%% accuracy), got %s", records[0].AgentID)
	}
}

func TestService_GetAccuracyByAgent_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &AgentAccuracy{})
	svc := NewService(db, dbtest.NewLogger(t))

	records, err := svc.GetAccuracyByAgent("NONEXISTENT")
	if err != nil {
		t.Fatalf("GetAccuracyByAgent failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestService_GetAccuracyByAgent(t *testing.T) {
	db := dbtest.NewDB(t, &AgentAccuracy{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&AgentAccuracy{AgentID: "A3", Period: "7d", TotalDecisions: 5, CorrectDecisions: 4, AccuracyPct: 80})
	db.Create(&AgentAccuracy{AgentID: "A3", Period: "30d", TotalDecisions: 20, CorrectDecisions: 14, AccuracyPct: 70})

	records, err := svc.GetAccuracyByAgent("A3")
	if err != nil {
		t.Fatalf("GetAccuracyByAgent failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestService_ListEvaluations_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{})
	svc := NewService(db, dbtest.NewLogger(t))

	evals, err := svc.ListEvaluations("", 0)
	if err != nil {
		t.Fatalf("ListEvaluations failed: %v", err)
	}
	if evals == nil {
		t.Error("expected non-nil empty slice")
	}
}

func TestService_ListEvaluations_FilterByAgent(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{})
	svc := NewService(db, dbtest.NewLogger(t))

	now := time.Now()
	db.Create(&DecisionEvaluation{DecisionTraceID: 1, ProductID: 1, AgentID: "A3", Score: 0.8, EvaluatedAt: &now, EvaluationType: "T+30"})
	db.Create(&DecisionEvaluation{DecisionTraceID: 2, ProductID: 2, AgentID: "A8", Score: 0.6, EvaluatedAt: &now, EvaluationType: "T+30"})

	evals, err := svc.ListEvaluations("A3", 0)
	if err != nil {
		t.Fatalf("ListEvaluations filter by agent failed: %v", err)
	}
	if len(evals) != 1 {
		t.Errorf("expected 1 evaluation for A3, got %d", len(evals))
	}
}

func TestService_ListEvaluations_FilterByProduct(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{})
	svc := NewService(db, dbtest.NewLogger(t))

	now := time.Now()
	db.Create(&DecisionEvaluation{DecisionTraceID: 1, ProductID: 1, AgentID: "A3", Score: 0.8, EvaluatedAt: &now})
	db.Create(&DecisionEvaluation{DecisionTraceID: 2, ProductID: 2, AgentID: "A8", Score: 0.6, EvaluatedAt: &now})

	evals, err := svc.ListEvaluations("", 1)
	if err != nil {
		t.Fatalf("ListEvaluations filter by product failed: %v", err)
	}
	if len(evals) != 1 {
		t.Errorf("expected 1 evaluation for product 1, got %d", len(evals))
	}
}

func TestService_RecalculateAccuracy_InvalidPeriod(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{}, &AgentAccuracy{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.RecalculateAccuracy("A3", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
}

func TestService_RecalculateAccuracy_NoEvaluations(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{}, &AgentAccuracy{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.RecalculateAccuracy("A3", "30d")
	if err != nil {
		t.Fatalf("RecalculateAccuracy with no evaluations failed: %v", err)
	}

	records, _ := svc.GetAccuracyByAgent("A3")
	if len(records) != 1 {
		t.Errorf("expected 1 accuracy record, got %d", len(records))
	}
	if records[0].TotalDecisions != 0 {
		t.Errorf("expected 0 total decisions, got %d", records[0].TotalDecisions)
	}
}

func TestService_RecalculateAccuracy_Upsert(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{}, &AgentAccuracy{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create evaluations with various scores.
	now := time.Now()
	db.Create(&DecisionEvaluation{DecisionTraceID: 1, AgentID: "A3", Score: 0.8, EvaluatedAt: &now, CreatedAt: now})
	db.Create(&DecisionEvaluation{DecisionTraceID: 2, AgentID: "A3", Score: 0.3, EvaluatedAt: &now, CreatedAt: now})
	db.Create(&DecisionEvaluation{DecisionTraceID: 3, AgentID: "A3", Score: 0.9, EvaluatedAt: &now, CreatedAt: now})

	err := svc.RecalculateAccuracy("A3", "30d")
	if err != nil {
		t.Fatalf("RecalculateAccuracy failed: %v", err)
	}

	records, _ := svc.GetAccuracyByAgent("A3")
	if len(records) != 1 {
		t.Fatalf("expected 1 accuracy record, got %d", len(records))
	}
	if records[0].TotalDecisions != 3 {
		t.Errorf("expected 3 total decisions, got %d", records[0].TotalDecisions)
	}
	if records[0].CorrectDecisions != 2 {
		t.Errorf("expected 2 correct decisions (scores >= 0.5), got %d", records[0].CorrectDecisions)
	}
	if records[0].AccuracyPct <= 0 {
		t.Errorf("expected positive accuracy pct, got %f", records[0].AccuracyPct)
	}
}

func TestService_RecalculateAccuracy_DuplicateUpsert(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{}, &AgentAccuracy{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Call twice — second call should update existing record.
	svc.RecalculateAccuracy("A3", "30d")

	records, _ := svc.GetAccuracyByAgent("A3")
	if len(records) != 1 {
		t.Errorf("expected 1 record after first call, got %d", len(records))
	}

	svc.RecalculateAccuracy("A3", "30d")
	records2, _ := svc.GetAccuracyByAgent("A3")
	if len(records2) != 1 {
		t.Errorf("expected 1 record after second call (upsert), got %d", len(records2))
	}
	if records2[0].ID != records[0].ID {
		t.Errorf("expected same ID on upsert, got old %d vs new %d", records[0].ID, records2[0].ID)
	}
}
