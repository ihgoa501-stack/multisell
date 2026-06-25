package evolution

import (
	"context"
	"math"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// ABTest tests
// ---------------------------------------------------------------------------

func newTestManager(t *testing.T) *ABTestManager {
	t.Helper()
	return NewABTestManager(zap.NewNop())
}

func TestABTest_StartAndGet(t *testing.T) {
	mgr := newTestManager(t)

	test, err := mgr.StartTest("test-1", "A5", "prompt-v1", "prompt-v2", 0.5, []string{"adoption_rate"})
	if err != nil {
		t.Fatalf("StartTest failed: %v", err)
	}

	if test.ID != "test-1" {
		t.Fatalf("expected ID 'test-1', got %q", test.ID)
	}
	if test.AgentID != "A5" {
		t.Fatalf("expected AgentID 'A5', got %q", test.AgentID)
	}
	if test.VariantA != "prompt-v1" {
		t.Fatalf("expected VariantA 'prompt-v1', got %q", test.VariantA)
	}
	if test.VariantB != "prompt-v2" {
		t.Fatalf("expected VariantB 'prompt-v2', got %q", test.VariantB)
	}
	if test.TrafficA != 0.5 {
		t.Fatalf("expected TrafficA 0.5, got %f", test.TrafficA)
	}
	if test.StartedAt.IsZero() {
		t.Fatal("expected StartedAt to be set")
	}
	if test.EndedAt != nil {
		t.Fatal("expected EndedAt to be nil for new test")
	}
	if test.Winner != nil {
		t.Fatal("expected Winner to be nil for new test")
	}

	got := mgr.GetTest("test-1")
	if got == nil {
		t.Fatal("GetTest returned nil")
	}
	if got.ID != "test-1" {
		t.Fatalf("GetTest returned test with ID %q", got.ID)
	}
}

func TestABTest_StartTest_DuplicateID(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.StartTest("dup", "A5", "a", "b", 0.5, []string{"accuracy"})
	if err != nil {
		t.Fatalf("first StartTest failed: %v", err)
	}

	_, err = mgr.StartTest("dup", "A5", "a", "b", 0.5, []string{"accuracy"})
	if err == nil {
		t.Fatal("expected error for duplicate ID, got nil")
	}
}

func TestABTest_StartTest_InvalidInputs(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.StartTest("", "A5", "a", "b", 0.5, []string{"accuracy"})
	if err == nil {
		t.Fatal("expected error for empty ID")
	}

	_, err = mgr.StartTest("t1", "A5", "a", "b", -0.1, []string{"accuracy"})
	if err == nil {
		t.Fatal("expected error for negative traffic")
	}

	_, err = mgr.StartTest("t2", "A5", "a", "b", 1.5, []string{"accuracy"})
	if err == nil {
		t.Fatal("expected error for traffic > 1")
	}

	_, err = mgr.StartTest("t3", "A5", "a", "b", 0.5, nil)
	if err == nil {
		t.Fatal("expected error for empty metrics")
	}
}

func TestABTest_RecordResult_Success(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.StartTest("record-test", "A5", "ctrl", "test", 0.5, []string{"accuracy"})
	if err != nil {
		t.Fatalf("StartTest failed: %v", err)
	}

	if err := mgr.RecordResult("record-test", "ctrl", 0.85); err != nil {
		t.Fatalf("RecordResult failed: %v", err)
	}
	if err := mgr.RecordResult("record-test", "ctrl", 0.90); err != nil {
		t.Fatalf("RecordResult failed: %v", err)
	}
	if err := mgr.RecordResult("record-test", "test", 0.92); err != nil {
		t.Fatalf("RecordResult failed: %v", err)
	}
}

func TestABTest_RecordResult_Errors(t *testing.T) {
	mgr := newTestManager(t)

	err := mgr.RecordResult("nonexistent", "a", 0.5)
	if err == nil {
		t.Fatal("expected error for non-existent test")
	}

	_, err = mgr.StartTest("err-test", "A5", "ctrl", "test", 0.5, []string{"accuracy"})
	if err != nil {
		t.Fatalf("StartTest failed: %v", err)
	}

	err = mgr.RecordResult("err-test", "unknown_variant", 0.5)
	if err == nil {
		t.Fatal("expected error for unknown variant")
	}

	err = mgr.RecordResult("err-test", "ctrl", math.NaN())
	if err == nil {
		t.Fatal("expected error for NaN value")
	}

	err = mgr.RecordResult("err-test", "ctrl", math.Inf(1))
	if err == nil {
		t.Fatal("expected error for Inf value")
	}

	_, err = mgr.EndTest("err-test")
	if err != nil {
		t.Fatalf("EndTest failed: %v", err)
	}
	err = mgr.RecordResult("err-test", "ctrl", 0.5)
	if err == nil {
		t.Fatal("expected error for ended test")
	}
}

func TestABTest_EndTest_DeclaresWinner(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.StartTest("winner-test", "A5", "ctrl", "test", 0.5, []string{"accuracy"})
	if err != nil {
		t.Fatalf("StartTest failed: %v", err)
	}

	mgr.RecordResult("winner-test", "ctrl", 0.70)
	mgr.RecordResult("winner-test", "ctrl", 0.72)
	mgr.RecordResult("winner-test", "ctrl", 0.68)
	mgr.RecordResult("winner-test", "test", 0.92)
	mgr.RecordResult("winner-test", "test", 0.95)
	mgr.RecordResult("winner-test", "test", 0.91)

	result, err := mgr.EndTest("winner-test")
	if err != nil {
		t.Fatalf("EndTest failed: %v", err)
	}

	if result.Winner != "B" {
		t.Fatalf("expected winner 'B', got %q", result.Winner)
	}
	if result.SampleSize != 6 {
		t.Fatalf("expected sample size 6, got %d", result.SampleSize)
	}
	if result.Lift <= 0 {
		t.Fatalf("expected positive lift, got %f", result.Lift)
	}
}

func TestABTest_EndTest_Latency_lowerIsBetter(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.StartTest("latency-test", "A5", "ctrl", "test", 0.5, []string{"latency"})
	if err != nil {
		t.Fatalf("StartTest failed: %v", err)
	}

	mgr.RecordResult("latency-test", "ctrl", 120.0)
	mgr.RecordResult("latency-test", "ctrl", 130.0)
	mgr.RecordResult("latency-test", "test", 200.0)
	mgr.RecordResult("latency-test", "test", 220.0)

	result, err := mgr.EndTest("latency-test")
	if err != nil {
		t.Fatalf("EndTest failed: %v", err)
	}

	if result.Winner != "A" {
		t.Fatalf("expected winner 'A' (lower latency), got %q", result.Winner)
	}
}

func TestABTest_EndTest_NoObservations(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.StartTest("empty", "A5", "a", "b", 0.5, []string{"accuracy"})
	if err != nil {
		t.Fatalf("StartTest failed: %v", err)
	}

	result, err := mgr.EndTest("empty")
	if err != nil {
		t.Fatalf("EndTest failed: %v", err)
	}

	if result.Winner != "tie" {
		t.Fatalf("expected winner 'tie', got %q", result.Winner)
	}
	if result.SampleSize != 0 {
		t.Fatalf("expected sample size 0, got %d", result.SampleSize)
	}
}

func TestABTest_EndTest_Tie(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.StartTest("tie-test", "A5", "a", "b", 0.5, []string{"accuracy"})
	if err != nil {
		t.Fatalf("StartTest failed: %v", err)
	}

	mgr.RecordResult("tie-test", "a", 0.80)
	mgr.RecordResult("tie-test", "a", 0.80)
	mgr.RecordResult("tie-test", "b", 0.80)
	mgr.RecordResult("tie-test", "b", 0.80)

	result, err := mgr.EndTest("tie-test")
	if err != nil {
		t.Fatalf("EndTest failed: %v", err)
	}

	if result.Winner != "tie" {
		t.Fatalf("expected winner 'tie', got %q", result.Winner)
	}
}

func TestABTest_EndTest_DoubleEnd(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.StartTest("double-end", "A5", "a", "b", 0.5, []string{"accuracy"})
	if err != nil {
		t.Fatalf("StartTest failed: %v", err)
	}

	mgr.RecordResult("double-end", "a", 0.8)
	mgr.RecordResult("double-end", "b", 0.9)

	_, err = mgr.EndTest("double-end")
	if err != nil {
		t.Fatalf("first EndTest failed: %v", err)
	}

	_, err = mgr.EndTest("double-end")
	if err == nil {
		t.Fatal("expected error for double end")
	}
}

func TestABTest_ListTests(t *testing.T) {
	mgr := newTestManager(t)

	mgr.StartTest("t1", "A5", "a", "b", 0.5, []string{"accuracy"})
	mgr.StartTest("t2", "A5", "c", "d", 0.5, []string{"latency"})
	mgr.StartTest("t3", "G3", "e", "f", 0.5, []string{"confidence"})

	all := mgr.ListTests("")
	if len(all) != 3 {
		t.Fatalf("expected 3 tests, got %d", len(all))
	}

	a5Tests := mgr.ListTests("A5")
	if len(a5Tests) != 2 {
		t.Fatalf("expected 2 A5 tests, got %d", len(a5Tests))
	}
}

func TestABTest_GetExperimentResult(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.StartTest("result-test", "A5", "ctrl", "test", 0.5, []string{"adoption_rate"})
	if err != nil {
		t.Fatalf("StartTest failed: %v", err)
	}

	mgr.RecordResult("result-test", "ctrl", 0.6)
	mgr.RecordResult("result-test", "test", 0.9)

	_, err = mgr.EndTest("result-test")
	if err != nil {
		t.Fatalf("EndTest failed: %v", err)
	}

	result, err := mgr.GetExperimentResult("result-test")
	if err != nil {
		t.Fatalf("GetExperimentResult failed: %v", err)
	}

	if result.Winner != "B" {
		t.Fatalf("expected winner 'B', got %q", result.Winner)
	}

	_, err = mgr.GetExperimentResult("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent test result")
	}

	mgr.StartTest("running", "A5", "a", "b", 0.5, []string{"accuracy"})
	_, err = mgr.GetExperimentResult("running")
	if err == nil {
		t.Fatal("expected error for running test result")
	}
}

// ---------------------------------------------------------------------------
// ThresholdOptimizer tests
// ---------------------------------------------------------------------------

func TestThresholdOptimizer_Basic(t *testing.T) {
	opt := NewThresholdOptimizer(zap.NewNop())

	samples := []DecisionSample{
		{PredictedConfidence: 0.95, WasCorrect: true, WasAdopted: true},
		{PredictedConfidence: 0.90, WasCorrect: true, WasAdopted: true},
		{PredictedConfidence: 0.88, WasCorrect: true, WasAdopted: true},
		{PredictedConfidence: 0.85, WasCorrect: true, WasAdopted: true},
		{PredictedConfidence: 0.82, WasCorrect: true, WasAdopted: true},
		{PredictedConfidence: 0.30, WasCorrect: false, WasAdopted: false},
		{PredictedConfidence: 0.25, WasCorrect: false, WasAdopted: false},
		{PredictedConfidence: 0.20, WasCorrect: false, WasAdopted: false},
	}

	bestThreshold, result := opt.OptimizeConfidence(context.Background(), "A5", samples)

	if result.SampleCount != 8 {
		t.Fatalf("expected 8 samples, got %d", result.SampleCount)
	}

	if bestThreshold < 0.31 || bestThreshold > 0.85 {
		t.Fatalf("expected threshold between 0.31 and 0.85, got %f", bestThreshold)
	}

	if result.F1Score <= 0 {
		t.Fatalf("expected positive F1, got %f", result.F1Score)
	}

	if result.Precision <= 0 || result.Recall <= 0 {
		t.Fatalf("expected positive precision (%f) and recall (%f)", result.Precision, result.Recall)
	}
}

func TestThresholdOptimizer_PerfectSeparation(t *testing.T) {
	opt := NewThresholdOptimizer(zap.NewNop())

	samples := []DecisionSample{
		{PredictedConfidence: 0.99, WasCorrect: true},
		{PredictedConfidence: 0.98, WasCorrect: true},
		{PredictedConfidence: 0.97, WasCorrect: true},
		{PredictedConfidence: 0.96, WasCorrect: true},
		{PredictedConfidence: 0.95, WasCorrect: true},
		{PredictedConfidence: 0.05, WasCorrect: false},
		{PredictedConfidence: 0.04, WasCorrect: false},
		{PredictedConfidence: 0.03, WasCorrect: false},
	}

	_, result := opt.OptimizeConfidence(context.Background(), "A5", samples)

	if result.F1Score != 1.0 {
		t.Fatalf("expected perfect F1 score of 1.0, got %f", result.F1Score)
	}
	if result.Precision != 1.0 {
		t.Fatalf("expected perfect precision of 1.0, got %f", result.Precision)
	}
	if result.Recall != 1.0 {
		t.Fatalf("expected perfect recall of 1.0, got %f", result.Recall)
	}
}

func TestThresholdOptimizer_EmptySamples(t *testing.T) {
	opt := NewThresholdOptimizer(zap.NewNop())

	bestThreshold, result := opt.OptimizeConfidence(context.Background(), "A5", nil)

	if bestThreshold != 0 {
		t.Fatalf("expected 0 threshold for empty samples, got %f", bestThreshold)
	}
	if result.SampleCount != 0 {
		t.Fatalf("expected 0 sample count, got %d", result.SampleCount)
	}
}

func TestThresholdOptimizer_AllCorrect(t *testing.T) {
	opt := NewThresholdOptimizer(zap.NewNop())

	samples := []DecisionSample{
		{PredictedConfidence: 0.9, WasCorrect: true},
		{PredictedConfidence: 0.8, WasCorrect: true},
		{PredictedConfidence: 0.7, WasCorrect: true},
	}

	_, result := opt.OptimizeConfidence(context.Background(), "A5", samples)

	if result.Precision != 1.0 {
		t.Fatalf("expected precision 1.0, got %f", result.Precision)
	}
}

func TestThresholdOptimizer_AllWrong(t *testing.T) {
	opt := NewThresholdOptimizer(zap.NewNop())

	samples := []DecisionSample{
		{PredictedConfidence: 0.9, WasCorrect: false},
		{PredictedConfidence: 0.8, WasCorrect: false},
		{PredictedConfidence: 0.7, WasCorrect: false},
	}

	_, result := opt.OptimizeConfidence(context.Background(), "A5", samples)

	if result.F1Score != 0 {
		t.Fatalf("expected F1 score of 0, got %f", result.F1Score)
	}
}

func TestComputePrecisionRecall(t *testing.T) {
	samples := []DecisionSample{
		{PredictedConfidence: 0.9, WasCorrect: true},  // TP
		{PredictedConfidence: 0.8, WasCorrect: true},  // TP
		{PredictedConfidence: 0.9, WasCorrect: false}, // FP
		{PredictedConfidence: 0.2, WasCorrect: true},  // FN (below threshold)
		{PredictedConfidence: 0.1, WasCorrect: false}, // TN
	}

	precision, recall := computePrecisionRecall(samples, 0.5)
	if precision < 0.666 || precision > 0.667 {
		t.Fatalf("expected precision ~0.667, got %f", precision)
	}
	if recall < 0.666 || recall > 0.667 {
		t.Fatalf("expected recall ~0.667, got %f", recall)
	}
}

// ---------------------------------------------------------------------------
// BehaviorAuditor tests
// ---------------------------------------------------------------------------

func newTestAuditor(t *testing.T) *BehaviorAuditor {
	t.Helper()
	return NewBehaviorAuditor(zap.NewNop())
}

func TestAuditor_EmptyRecords(t *testing.T) {
	auditor := newTestAuditor(t)

	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	report := auditor.GenerateReport("A5", start, end, nil)

	if report.AgentID != "A5" {
		t.Fatalf("expected agent ID 'A5', got %q", report.AgentID)
	}
	if len(report.Recommendations) != 1 {
		t.Fatalf("expected 1 recommendation for empty records, got %d", len(report.Recommendations))
	}
}

func TestAuditor_AdoptionRate(t *testing.T) {
	auditor := newTestAuditor(t)

	records := []DecisionRecord{
		{ID: "1", AgentID: "A5", DecisionPoint: "stock_alert", WasAdopted: true},
		{ID: "2", AgentID: "A5", DecisionPoint: "stock_alert", WasAdopted: true},
		{ID: "3", AgentID: "A5", DecisionPoint: "stock_alert", WasAdopted: false, RejectionReason: "threshold_too_low"},
		{ID: "4", AgentID: "A5", DecisionPoint: "replenish", WasAdopted: false, RejectionReason: "wrong_supplier"},
	}

	report := auditor.GenerateReport("A5", time.Now().Add(-7*24*time.Hour), time.Now(), records)

	if report.AdoptionRate != 0.5 {
		t.Fatalf("expected adoption rate 0.5, got %f", report.AdoptionRate)
	}

	if report.DecisionSummary["stock_alert"] != 3 {
		t.Fatalf("expected 3 stock_alert decisions, got %d", report.DecisionSummary["stock_alert"])
	}
	if report.DecisionSummary["replenish"] != 1 {
		t.Fatalf("expected 1 replenish decision, got %d", report.DecisionSummary["replenish"])
	}

	if report.RejectionReasons["threshold_too_low"] != 1 {
		t.Fatalf("expected 1 'threshold_too_low' rejection, got %d", report.RejectionReasons["threshold_too_low"])
	}
}

func TestAuditor_FailurePatterns(t *testing.T) {
	auditor := newTestAuditor(t)

	records := []DecisionRecord{
		{ID: "d1", AgentID: "A5", DecisionPoint: "stock_alert", WasCorrect: false, FailurePattern: "overconfidence"},
		{ID: "d2", AgentID: "A5", DecisionPoint: "stock_alert", WasCorrect: false, FailurePattern: "overconfidence"},
		{ID: "d3", AgentID: "A5", DecisionPoint: "replenish", WasCorrect: false, FailurePattern: "parse_error"},
		{ID: "d4", AgentID: "A5", DecisionPoint: "replenish", WasCorrect: true},
		{ID: "d5", AgentID: "A5", DecisionPoint: "stock_alert", WasCorrect: false, FailurePattern: "overconfidence"},
	}

	report := auditor.GenerateReport("A5", time.Now().Add(-7*24*time.Hour), time.Now(), records)

	if len(report.TopFailures) != 2 {
		t.Fatalf("expected 2 failure patterns, got %d", len(report.TopFailures))
	}

	if report.TopFailures[0].Pattern != "overconfidence" {
		t.Fatalf("expected first failure pattern 'overconfidence', got %q", report.TopFailures[0].Pattern)
	}
	if report.TopFailures[0].Count != 3 {
		t.Fatalf("expected overconfidence count 3, got %d", report.TopFailures[0].Count)
	}

	if len(report.TopFailures[0].ExampleIDs) != 3 {
		t.Fatalf("expected 3 example IDs for overconfidence, got %d", len(report.TopFailures[0].ExampleIDs))
	}
}

func TestAuditor_Recommendations(t *testing.T) {
	auditor := newTestAuditor(t)

	records := []DecisionRecord{
		{ID: "d1", AgentID: "A5", DecisionPoint: "stock_alert", WasAdopted: false, RejectionReason: "not_helpful", WasCorrect: false, FailurePattern: "overconfidence", Confidence: 0.9},
		{ID: "d2", AgentID: "A5", DecisionPoint: "stock_alert", WasAdopted: false, RejectionReason: "not_helpful", WasCorrect: false, FailurePattern: "overconfidence", Confidence: 0.85},
		{ID: "d3", AgentID: "A5", DecisionPoint: "replenish", WasAdopted: false, RejectionReason: "wrong_supplier", WasCorrect: false, FailurePattern: "overconfidence", Confidence: 0.95},
		{ID: "d4", AgentID: "A5", DecisionPoint: "stock_alert", WasAdopted: false, RejectionReason: "not_helpful", WasCorrect: false, FailurePattern: "parse_error", Confidence: 0.3},
		{ID: "d5", AgentID: "A5", DecisionPoint: "stock_alert", WasAdopted: true, WasCorrect: true, Confidence: 0.88},
	}

	report := auditor.GenerateReport("A5", time.Now().Add(-7*24*time.Hour), time.Now(), records)

	if len(report.Recommendations) < 2 {
		t.Fatalf("expected at least 2 recommendations, got %d", len(report.Recommendations))
	}

	foundOverconfidence := false
	for _, rec := range report.Recommendations {
		if contains(rec, "overconfidence") {
			foundOverconfidence = true
			break
		}
	}
	if !foundOverconfidence {
		t.Fatal("expected recommendation mentioning overconfidence")
	}
}

func TestAuditor_HighAdoptionNoFailures(t *testing.T) {
	auditor := newTestAuditor(t)

	records := make([]DecisionRecord, 100)
	for i := 0; i < 100; i++ {
		records[i] = DecisionRecord{
			ID:            itoa(i + 1),
			AgentID:       "A5",
			DecisionPoint: "stock_alert",
			WasAdopted:    true,
			WasCorrect:    true,
			Confidence:    0.9,
		}
	}

	report := auditor.GenerateReport("A5", time.Now().Add(-7*24*time.Hour), time.Now(), records)

	if report.AdoptionRate != 1.0 {
		t.Fatalf("expected adoption rate 1.0, got %f", report.AdoptionRate)
	}
	if len(report.TopFailures) != 0 {
		t.Fatalf("expected 0 failure patterns, got %d", len(report.TopFailures))
	}

	foundOK := false
	for _, rec := range report.Recommendations {
		if contains(rec, "No major issues") {
			foundOK = true
			break
		}
	}
	if !foundOK {
		t.Fatal("expected 'No major issues' recommendation")
	}
}

func TestAuditor_LowConfidenceCorrect(t *testing.T) {
	auditor := newTestAuditor(t)

	records := []DecisionRecord{
		{ID: "d1", AgentID: "A5", DecisionPoint: "stock_alert", WasAdopted: true, WasCorrect: true, Confidence: 0.15},
		{ID: "d2", AgentID: "A5", DecisionPoint: "stock_alert", WasAdopted: true, WasCorrect: true, Confidence: 0.20},
		{ID: "d3", AgentID: "A5", DecisionPoint: "stock_alert", WasAdopted: true, WasCorrect: true, Confidence: 0.90},
	}

	report := auditor.GenerateReport("A5", time.Now().Add(-7*24*time.Hour), time.Now(), records)

	foundCalibration := false
	for _, rec := range report.Recommendations {
		if contains(rec, "threshold optimizer") || contains(rec, "recalibrate") {
			foundCalibration = true
			break
		}
	}
	if !foundCalibration {
		t.Fatal("expected recommendation about threshold recalibration")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
