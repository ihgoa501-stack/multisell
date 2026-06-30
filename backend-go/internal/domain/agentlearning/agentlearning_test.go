package agentlearning

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestPeriodHours(t *testing.T) {
	tests := []struct {
		period string
		want   time.Duration
	}{
		{"7d", 7 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"90d", 90 * 24 * time.Hour},
		{"invalid", 0},
		{"", 0},
	}
	for _, tc := range tests {
		got := periodHours(tc.period)
		if got != tc.want {
			t.Errorf("periodHours(%q) = %v, want %v", tc.period, got, tc.want)
		}
	}
}

func TestGuessAgent(t *testing.T) {
	s := &Service{}
	tests := []struct {
		decisionPoint string
		want          string
	}{
		{"acos_analysis", "A3"},
		{"price_watch", "A3"},
		{"profit_watch", "A3"},
		{"sourcing_scan", "A8"},
		{"listing_optimize", "A2"},
		{"discount_risk_check", "G3"},
		{"unknown_point", "A3"},
		{"", "A3"},
	}
	for _, tc := range tests {
		got := s.guessAgent(tc.decisionPoint)
		if got != tc.want {
			t.Errorf("guessAgent(%q) = %q, want %q", tc.decisionPoint, got, tc.want)
		}
	}
}

func TestComputeMarginAccuracy(t *testing.T) {
	s := &Service{}
	tests := []struct {
		name         string
		pred, actual float64
		want         float64
	}{
		{"perfect match", 0.15, 0.15, 1.0},
		{"both zero", 0, 0, 1.0},
		{"predicted zero", 0, 0.1, 0},
		{"actual half of predicted", 0.20, 0.10, 0.5},
		{"actual double predicted", 0.10, 0.20, 0.5},
		{"close 90%", 0.20, 0.18, 0.9},
		{"very optimistic", 0.50, 0.05, 0.1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.computeMarginAccuracy(tc.pred, tc.actual)
			if got != tc.want {
				t.Errorf("computeMarginAccuracy(%v, %v) = %v, want %v", tc.pred, tc.actual, got, tc.want)
			}
		})
	}
}

func TestRecalculateAccuracy_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{}, &AgentAccuracy{})
	s := NewService(db, dbtest.NewLogger(t))

	err := s.RecalculateAccuracy("A3", "30d")
	if err != nil {
		t.Fatalf("RecalculateAccuracy on empty DB: %v", err)
	}

	records, err := s.GetAllAccuracy()
	if err != nil {
		t.Fatalf("GetAllAccuracy: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 accuracy record, got %d", len(records))
	}
	if records[0].AgentID != "A3" {
		t.Errorf("expected agent A3, got %s", records[0].AgentID)
	}
	if records[0].TotalDecisions != 0 {
		t.Errorf("expected 0 total decisions, got %d", records[0].TotalDecisions)
	}
}

func TestRecalculateAccuracy_WithEvaluations(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{}, &AgentAccuracy{})
	s := NewService(db, dbtest.NewLogger(t))

	now := time.Now()
	for _, e := range []DecisionEvaluation{
		{AgentID: "A3", Score: 0.8, CreatedAt: now.Add(-12 * time.Hour)},
		{AgentID: "A3", Score: 0.6, CreatedAt: now.Add(-24 * time.Hour)},
		{AgentID: "A3", Score: 0.3, CreatedAt: now.Add(-36 * time.Hour)},
		{AgentID: "A3", Score: 0.9, CreatedAt: now.Add(-48 * time.Hour)},
	} {
		db.Create(&e)
	}

	err := s.RecalculateAccuracy("A3", "7d")
	if err != nil {
		t.Fatalf("RecalculateAccuracy: %v", err)
	}

	records, _ := s.GetAccuracyByAgent("A3")
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].TotalDecisions != 4 {
		t.Errorf("expected 4 total, got %d", records[0].TotalDecisions)
	}
	if records[0].CorrectDecisions != 3 {
		t.Errorf("expected 3 correct, got %d", records[0].CorrectDecisions)
	}
	if records[0].AccuracyPct != 75.0 {
		t.Errorf("expected 75%%, got %.2f", records[0].AccuracyPct)
	}
}

func TestListEvaluations(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{})
	s := NewService(db, dbtest.NewLogger(t))

	for i := int64(1); i <= 3; i++ {
		db.Create(&DecisionEvaluation{
			DecisionTraceID: i, ProductID: i, AgentID: "A3", Score: float64(i) * 0.25,
		})
	}

	all, _ := s.ListEvaluations("", 0)
	if len(all) != 3 {
		t.Errorf("expected 3, got %d", len(all))
	}

	filtered, _ := s.ListEvaluations("A3", 2)
	if len(filtered) != 1 || filtered[0].ProductID != 2 {
		t.Errorf("expected 1 eval for product 2, got %d", len(filtered))
	}
}

func TestGetAllAccuracy_Sorted(t *testing.T) {
	db := dbtest.NewDB(t, &AgentAccuracy{})
	s := NewService(db, dbtest.NewLogger(t))

	db.Create(&AgentAccuracy{AgentID: "A3", Period: "30d", TotalDecisions: 10, CorrectDecisions: 7, AccuracyPct: 70, Trend: "stable"})
	db.Create(&AgentAccuracy{AgentID: "A8", Period: "30d", TotalDecisions: 5, CorrectDecisions: 4, AccuracyPct: 80, Trend: "improving"})

	records, _ := s.GetAllAccuracy()
	if len(records) != 2 {
		t.Fatalf("expected 2, got %d", len(records))
	}
	if records[0].AgentID != "A8" {
		t.Errorf("expected A8 first (higher accuracy), got %s", records[0].AgentID)
	}
}

func TestRecalculateAccuracy_InvalidPeriod(t *testing.T) {
	db := dbtest.NewDB(t, &DecisionEvaluation{}, &AgentAccuracy{})
	s := NewService(db, dbtest.NewLogger(t))

	err := s.RecalculateAccuracy("A3", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
}
