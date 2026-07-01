package trustscore

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestDB(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &TrustScore{})
	return NewService(db, zap.NewNop())
}

func setupRouter(t *testing.T) (*Service, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &TrustScore{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)

	r := gin.New()
	rg := r.Group("/api/v1")

	// Register /trust-scores/summary BEFORE /trust-scores/:agent_id
	// to avoid route conflict with the wildcard.
	rg.GET("/trust-scores", h.List)
	rg.GET("/trust-scores/summary", h.Summary)
	rg.GET("/trust-scores/:agent_id", h.GetByAgent)
	rg.POST("/trust-scores/recalculate", h.Recalculate)
	rg.POST("/trust-scores/eligible", h.Eligible)
	rg.PUT("/trust-scores/:agent_id/level", h.UpdateLevel)
	rg.POST("/trust-scores/auto-upgrade", h.AutoUpgrade)
	return svc, r
}

func findScore(scores []TrustScore, agentID string) *TrustScore {
	for i := range scores {
		if scores[i].AgentID == agentID {
			return &scores[i]
		}
	}
	return nil
}

// approx checks that |got - want| < eps.
func approx(got, want, eps float64) bool {
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	return diff < eps
}

// ---------------------------------------------------------------------------
// Pure function tests
// ---------------------------------------------------------------------------

func TestClamp01(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{input: -1.0, expected: 0},
		{input: -0.5, expected: 0},
		{input: -0.001, expected: 0},
		{input: 0, expected: 0},
		{input: 0.001, expected: 0.001},
		{input: 0.5, expected: 0.5},
		{input: 0.999, expected: 0.999},
		{input: 1, expected: 1},
		{input: 1.001, expected: 1},
		{input: 1.5, expected: 1},
		{input: 100, expected: 1},
	}
	for _, tt := range tests {
		got := clamp01(tt.input)
		if got != tt.expected {
			t.Errorf("clamp01(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestMaxInt(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{a: 5, b: 3, expected: 5},
		{a: 2, b: 7, expected: 7},
		{a: 4, b: 4, expected: 4},
		{a: 0, b: 5, expected: 5},
		{a: -1, b: 3, expected: 3},
		{a: -5, b: -2, expected: -2},
	}
	for _, tt := range tests {
		got := maxInt(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("maxInt(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestDetermineTargetLevel(t *testing.T) {
	// advisory with 0.0 score -> stays advisory
	t.Run("advisory with 0.0", func(t *testing.T) {
		got := determineTargetLevel(0.0, "advisory")
		if got != "advisory" {
			t.Errorf("determineTargetLevel(0.0, advisory) = %v, want advisory", got)
		}
	})

	// guided with 0.35 score -> stays guided because 0.35 < 0.55 (supervised threshold)
	t.Run("guided with 0.35", func(t *testing.T) {
		got := determineTargetLevel(0.35, "guided")
		if got != "guided" {
			t.Errorf("determineTargetLevel(0.35, guided) = %v, want guided", got)
		}
	})

	// one-step-upgrade constraint: advisory with very high score
	// upgrades to guided (one step up)
	t.Run("advisory high (one-step constraint)", func(t *testing.T) {
		got := determineTargetLevel(0.90, "advisory")
		if got != "guided" {
			t.Errorf("determineTargetLevel(0.90, advisory) = %v, want guided (one step up)", got)
		}
	})

	t.Run("supervised with 0.55", func(t *testing.T) {
		got := determineTargetLevel(0.55, "supervised")
		if got != "supervised" {
			t.Errorf("determineTargetLevel(0.55, supervised) = %v, want supervised", got)
		}
	})

	t.Run("autonomous stays autonomous", func(t *testing.T) {
		got := determineTargetLevel(0.10, "autonomous")
		if got != "autonomous" {
			t.Errorf("determineTargetLevel(0.10, autonomous) = %v, want autonomous", got)
		}
	})

	t.Run("guided with 0.0", func(t *testing.T) {
		got := determineTargetLevel(0.0, "guided")
		if got != "guided" {
			t.Errorf("determineTargetLevel(0.0, guided) = %v, want guided", got)
		}
	})
}

// ---------------------------------------------------------------------------
// Service tests
// ---------------------------------------------------------------------------

func TestList_Empty(t *testing.T) {
	svc := newTestDB(t)
	scores, err := svc.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(scores) != 0 {
		t.Errorf("expected empty list, got %d items", len(scores))
	}
}

func TestList_WithScores(t *testing.T) {
	svc := newTestDB(t)

	s1 := NewTrustScore("A1", "Product Scout", "autonomous")
	s1.TrustScore = 0.80
	s1.AutonomyLevel = "supervised"
	if err := svc.db.Create(s1).Error; err != nil {
		t.Fatalf("create A1: %v", err)
	}

	s2 := NewTrustScore("A2", "Listing Optimizer", "autonomous")
	s2.TrustScore = 0.30
	if err := svc.db.Create(s2).Error; err != nil {
		t.Fatalf("create A2: %v", err)
	}

	s3 := NewTrustScore("A3", "Ad Advice", "autonomous")
	s3.TrustScore = 0.90
	s3.AutonomyLevel = "autonomous"
	if err := svc.db.Create(s3).Error; err != nil {
		t.Fatalf("create A3: %v", err)
	}

	scores, err := svc.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(scores))
	}

	// Must be ordered by trust_score DESC: A3 (0.90), A1 (0.80), A2 (0.30)
	if scores[0].AgentID != "A3" {
		t.Errorf("expected first A3, got %s", scores[0].AgentID)
	}
	if scores[0].TrustScore != 0.90 {
		t.Errorf("expected first score 0.90, got %f", scores[0].TrustScore)
	}
	if scores[1].AgentID != "A1" {
		t.Errorf("expected second A1, got %s", scores[1].AgentID)
	}
	if scores[2].AgentID != "A2" {
		t.Errorf("expected third A2, got %s", scores[2].AgentID)
	}
}

func TestGetByAgent_Found(t *testing.T) {
	svc := newTestDB(t)
	score := NewTrustScore("A1", "Product Scout", "autonomous")
	if err := svc.db.Create(score).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := svc.GetByAgent("A1")
	if err != nil {
		t.Fatalf("GetByAgent error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil score")
	}
	if got.AgentID != "A1" {
		t.Errorf("expected A1, got %s", got.AgentID)
	}
	if got.AgentName != "Product Scout" {
		t.Errorf("expected Product Scout, got %s", got.AgentName)
	}
	if got.AutonomyLevel != "advisory" {
		t.Errorf("expected advisory, got %s", got.AutonomyLevel)
	}
}

func TestGetByAgent_NotFound(t *testing.T) {
	svc := newTestDB(t)
	got, err := svc.GetByAgent("NONEXISTENT")
	if err != nil {
		t.Fatalf("GetByAgent error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestRecalculate(t *testing.T) {
	svc := newTestDB(t)

	// Create external tables referenced by Recalculate SQL queries.
	svc.db.Exec(`CREATE TABLE unified_action (
		id INTEGER, agent_id TEXT, status TEXT,
		rejected_by TEXT, approved_by TEXT, proposed_at TIMESTAMP
	)`)
	svc.db.Exec(`CREATE TABLE ai_trace (
		id INTEGER, agent_id TEXT, confidence REAL
	)`)

	now := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)

	// ---- Insert data for A1 ----
	// 7 approved (3 auto / policy, 4 manual)
	for i := 0; i < 3; i++ {
		svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, approved_by, proposed_at) VALUES (?, 'A1', 'approved', 'policy', ?)`,
			i+1, now)
	}
	for i := 0; i < 4; i++ {
		svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, approved_by, proposed_at) VALUES (?, 'A1', 'approved', 'human', ?)`,
			i+4, now)
	}
	// 1 failed
	svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, proposed_at) VALUES (8, 'A1', 'failed', ?)`, now)
	// 2 rejected by human (not policy) -> counts as rejected
	svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, rejected_by, proposed_at) VALUES (9, 'A1', 'rejected', 'human', ?)`, now)
	svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, rejected_by, proposed_at) VALUES (10, 'A1', 'rejected', 'human', ?)`, now)
	// AI traces for A1: avg = (0.85 + 0.90 + 0.80) / 3 = 0.85
	svc.db.Exec(`INSERT INTO ai_trace (id, agent_id, confidence) VALUES (1, 'A1', 0.85)`)
	svc.db.Exec(`INSERT INTO ai_trace (id, agent_id, confidence) VALUES (2, 'A1', 0.90)`)
	svc.db.Exec(`INSERT INTO ai_trace (id, agent_id, confidence) VALUES (3, 'A1', 0.80)`)

	// ---- Insert data for A2 ----
	for i := 0; i < 5; i++ {
		svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, approved_by, proposed_at) VALUES (?, 'A2', 'approved', 'policy', ?)`,
			i+11, now)
	}
	svc.db.Exec(`INSERT INTO ai_trace (id, agent_id, confidence) VALUES (4, 'A2', 0.95)`)

	// ---- A3: a single rejected-by-policy action (should NOT count as human-rejected) ----
	svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, rejected_by, proposed_at) VALUES (21, 'A3', 'rejected', 'policy', ?)`, now)

	// ---- First Recalculate: creates new records with default values ----
	if err := svc.Recalculate(); err != nil {
		t.Fatalf("first Recalculate() error: %v", err)
	}

	allAgents := len(defaultAgentList())
	scores, err := svc.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(scores) != allAgents {
		t.Errorf("expected %d scores, got %d", allAgents, len(scores))
	}

	// First call creates records with computed values from batch data
	a1First, _ := svc.GetByAgent("A1")
	if a1First.TotalActions != 10 {
		t.Errorf("first call: expected total_actions=10, got %d", a1First.TotalActions)
	}
	if !approx(a1First.TrustScore, 0.640, 0.0005) {
		t.Errorf("first call: expected trust_score=~0.640, got %f", a1First.TrustScore)
	}

	// ---- Second Recalculate: updates existing records with computed values ----
	if err := svc.Recalculate(); err != nil {
		t.Fatalf("second Recalculate() error: %v", err)
	}

	// --- Verify A1 ---
	a1, _ := svc.GetByAgent("A1")
	if a1.TotalActions != 10 {
		t.Errorf("total_actions: want 10, got %d", a1.TotalActions)
	}
	if a1.AdoptedActions != 7 {
		t.Errorf("adopted: want 7, got %d", a1.AdoptedActions)
	}
	if a1.RejectedActions != 2 {
		t.Errorf("rejected (human): want 2, got %d", a1.RejectedActions)
	}
	if a1.FailedActions != 1 {
		t.Errorf("failed: want 1, got %d", a1.FailedActions)
	}
	if a1.AutoApproved != 3 {
		t.Errorf("auto_approved: want 3, got %d", a1.AutoApproved)
	}
	if !approx(a1.AdoptionRate, 0.7, 0.0005) {
		t.Errorf("adoption_rate: want ~0.700, got %f", a1.AdoptionRate)
	}
	if !approx(a1.ExecutionSuccess, 0.9, 0.0005) {
		t.Errorf("execution_success: want ~0.900, got %f", a1.ExecutionSuccess)
	}
	if !approx(a1.AvgConfidence, 0.85, 0.0005) {
		t.Errorf("avg_confidence: want ~0.850, got %f", a1.AvgConfidence)
	}
	// trustScore = 0.7*0.4 + 0.9*0.3 + 0.85*0.3 = 0.28 + 0.27 + 0.255 = 0.640
	if !approx(a1.TrustScore, 0.640, 0.0005) {
		t.Errorf("trust_score: want ~0.640, got %f", a1.TrustScore)
	}
	if a1.AutonomyLevel != "advisory" {
		t.Errorf("autonomy_level: want advisory, got %s", a1.AutonomyLevel)
	}
	if a1.TargetLevel != "guided" {
		t.Errorf("target_level: want guided, got %s", a1.TargetLevel)
	}

	// --- Verify A2 ---
	a2, _ := svc.GetByAgent("A2")
	if a2.TotalActions != 5 {
		t.Errorf("A2 total_actions: want 5, got %d", a2.TotalActions)
	}
	if a2.AdoptedActions != 5 {
		t.Errorf("A2 adopted: want 5, got %d", a2.AdoptedActions)
	}
	if !approx(a2.TrustScore, 0.790, 0.0005) {
		t.Errorf("A2 trust_score: want ~0.790, got %f", a2.TrustScore)
	}

	// --- Verify A3: policy-only rejected, zero adopted -> adoption=0, execSuccess=1.0, conf=0
	a3, _ := svc.GetByAgent("A3")
	if a3.TotalActions != 1 {
		t.Errorf("A3 total_actions: want 1, got %d", a3.TotalActions)
	}
	if a3.RejectedActions != 0 {
		t.Errorf("A3 rejected (policy should be excluded): want 0, got %d", a3.RejectedActions)
	}
	// trust=0*0.35 + 1.0*0.25 + 0*0.20 + 0*0.20 = 0.250
	if !approx(a3.TrustScore, 0.250, 0.0005) {
		t.Errorf("A3 trust_score: want ~0.250, got %f", a3.TrustScore)
	}

	// --- Verify an agent with NO data ---
	a4, _ := svc.GetByAgent("A4")
	// trust=0*0.35 + 1.0*0.25 + 0*0.20 + 0*0.20 = 0.250
	if !approx(a4.TrustScore, 0.250, 0.0005) {
		t.Errorf("A4 trust_score (no data): want ~0.250, got %f", a4.TrustScore)
	}
}

func TestRecalculateForAgent(t *testing.T) {
	svc := newTestDB(t)

	svc.db.Exec(`CREATE TABLE unified_action (
		id INTEGER, agent_id TEXT, status TEXT,
		rejected_by TEXT, approved_by TEXT, proposed_at TIMESTAMP
	)`)
	svc.db.Exec(`CREATE TABLE ai_trace (
		id INTEGER, agent_id TEXT, confidence REAL
	)`)

	now := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)

	// A5: 5 approved (all auto), 1 failed
	for i := 0; i < 5; i++ {
		svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, approved_by, proposed_at) VALUES (?, 'A5', 'approved', 'policy', ?)`,
			i+1, now)
	}
	svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, proposed_at) VALUES (6, 'A5', 'failed', ?)`, now)
	svc.db.Exec(`INSERT INTO ai_trace (id, agent_id, confidence) VALUES (1, 'A5', 0.95)`)

	// First call creates new record with default values
	if err := svc.RecalculateForAgent("A5", "Inventory Alert", "autonomous"); err != nil {
		t.Fatalf("first RecalculateForAgent error: %v", err)
	}
	a5First, _ := svc.GetByAgent("A5")
	if a5First.TotalActions != 0 {
		t.Errorf("first call: total_actions=0 (new record), got %d", a5First.TotalActions)
	}

	// Second call updates existing record
	if err := svc.RecalculateForAgent("A5", "Inventory Alert", "autonomous"); err != nil {
		t.Fatalf("second RecalculateForAgent error: %v", err)
	}

	a5, _ := svc.GetByAgent("A5")
	if a5.TotalActions != 6 {
		t.Errorf("total_actions: want 6, got %d", a5.TotalActions)
	}
	if a5.AdoptedActions != 5 {
		t.Errorf("adopted: want 5, got %d", a5.AdoptedActions)
	}
	if a5.FailedActions != 1 {
		t.Errorf("failed: want 1, got %d", a5.FailedActions)
	}
	if a5.AutoApproved != 5 {
		t.Errorf("auto_approved: want 5, got %d", a5.AutoApproved)
	}
	// adoptionRate=5/6≈0.8333, execSuccess=1-1/6≈0.8333, avgConf=0.95
	// trustScore=0.8333*0.35+0.8333*0.25+0.95*0.20+0*0.20=0.2917+0.2083+0.1900=0.690
	if !approx(a5.AdoptionRate, 0.8333, 0.001) {
		t.Errorf("adoption_rate: want ~0.8333, got %f", a5.AdoptionRate)
	}
	if !approx(a5.ExecutionSuccess, 0.8333, 0.001) {
		t.Errorf("execution_success: want ~0.8333, got %f", a5.ExecutionSuccess)
	}
	if !approx(a5.TrustScore, 0.690, 0.001) {
		t.Errorf("trust_score: want ~0.690, got %f", a5.TrustScore)
	}

	// Agent with no existing record gets created
	if err := svc.RecalculateForAgent("NONEXISTENT", "Ghost", "none"); err != nil {
		t.Fatalf("RecalculateForAgent new agent error: %v", err)
	}
	ghost, _ := svc.GetByAgent("NONEXISTENT")
	if ghost == nil {
		t.Fatal("expected ghost agent to be created")
	}
}

func TestGetEligibleForUpgrade(t *testing.T) {
	svc := newTestDB(t)

	// A1: trust=0.70, guided -> eligible
	e1 := NewTrustScore("A1", "Scout", "autonomous")
	e1.TrustScore = 0.70
	e1.AutonomyLevel = "guided"
	svc.db.Create(e1)

	// A2: trust=0.20, advisory -> NOT eligible (trust < 0.55)
	e2 := NewTrustScore("A2", "Optimizer", "autonomous")
	e2.TrustScore = 0.20
	svc.db.Create(e2)

	// A3: trust=0.90, autonomous -> NOT eligible (is autonomous)
	e3 := NewTrustScore("A3", "Advisor", "autonomous")
	e3.TrustScore = 0.90
	e3.AutonomyLevel = "autonomous"
	svc.db.Create(e3)

	// A4: trust=0.55, guided -> eligible (exactly at threshold)
	e4 := NewTrustScore("A4", "Service", "autonomous")
	e4.TrustScore = 0.55
	e4.AutonomyLevel = "guided"
	svc.db.Create(e4)

	// A5: trust=0.55, advisory -> eligible (advisory != autonomous)
	e5 := NewTrustScore("A5", "Alerter", "autonomous")
	e5.TrustScore = 0.60
	e5.AutonomyLevel = "advisory"
	svc.db.Create(e5)

	scores, err := svc.GetEligibleForUpgrade()
	if err != nil {
		t.Fatalf("GetEligibleForUpgrade() error: %v", err)
	}
	if len(scores) != 3 {
		t.Fatalf("expected 3 eligible scores, got %d", len(scores))
	}

	ids := make(map[string]bool)
	for _, s := range scores {
		ids[s.AgentID] = true
	}
	if !ids["A1"] {
		t.Error("expected A1 in eligible")
	}
	if !ids["A4"] {
		t.Error("expected A4 in eligible")
	}
	if !ids["A5"] {
		t.Error("expected A5 in eligible")
	}
	if ids["A2"] {
		t.Error("A2 (trust=0.20) should NOT be eligible")
	}
	if ids["A3"] {
		t.Error("A3 (autonomous) should NOT be eligible")
	}
}

// ---------------------------------------------------------------------------
// Upgrader tests
// ---------------------------------------------------------------------------

func TestUpgradeEligible(t *testing.T) {
	svc := newTestDB(t)
	upgrader := NewUpgraderFromSvc(svc)

	// A1: eligible (trust>=0.55, not autonomous) and target != current -> upgraded
	score := NewTrustScore("A1", "Scout", "autonomous")
	score.TrustScore = 0.60
	score.AutonomyLevel = "advisory"
	score.TargetLevel = "guided"
	if err := svc.db.Create(score).Error; err != nil {
		t.Fatalf("create A1: %v", err)
	}

	// A2: eligible but target == current -> skipped
	score2 := NewTrustScore("A2", "Optimizer", "autonomous")
	score2.TrustScore = 0.70
	score2.AutonomyLevel = "advisory"
	score2.TargetLevel = "advisory"
	svc.db.Create(score2)

	// A3: already autonomous -> not returned by GetEligibleForUpgrade
	score3 := NewTrustScore("A3", "Advisor", "autonomous")
	score3.TrustScore = 0.90
	score3.AutonomyLevel = "autonomous"
	score3.TargetLevel = "autonomous"
	svc.db.Create(score3)

	// A4: eligible but target_level empty string -> skipped (target == "")
	score4 := NewTrustScore("A4", "Service", "autonomous")
	score4.TrustScore = 0.60
	score4.AutonomyLevel = "advisory"
	score4.TargetLevel = ""
	svc.db.Create(score4)

	results, err := upgrader.UpgradeEligible()
	if err != nil {
		t.Fatalf("UpgradeEligible() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 upgrade, got %d", len(results))
	}

	r := results[0]
	if r.AgentID != "A1" {
		t.Errorf("expected A1, got %s", r.AgentID)
	}
	if r.FromLevel != "advisory" {
		t.Errorf("expected from advisory, got %s", r.FromLevel)
	}
	if r.ToLevel != "guided" {
		t.Errorf("expected to guided, got %s", r.ToLevel)
	}
	if !approx(r.TrustScore, 0.60, 0.0005) {
		t.Errorf("trust_score: want ~0.60, got %f", r.TrustScore)
	}

	// Verify the DB was actually updated
	updated, _ := svc.GetByAgent("A1")
	if updated.AutonomyLevel != "guided" {
		t.Errorf("DB autonomy_level: want guided, got %s", updated.AutonomyLevel)
	}
}

func TestAutoUpgrade(t *testing.T) {
	svc := newTestDB(t)
	upgrader := NewUpgraderFromSvc(svc)

	// Create external tables so Recalculate can run
	svc.db.Exec(`CREATE TABLE unified_action (
		id INTEGER, agent_id TEXT, status TEXT,
		rejected_by TEXT, approved_by TEXT, proposed_at TIMESTAMP
	)`)
	svc.db.Exec(`CREATE TABLE ai_trace (
		id INTEGER, agent_id TEXT, confidence REAL
	)`)

	now := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 10; i++ {
		svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, approved_by, proposed_at) VALUES (?, 'A1', 'approved', 'policy', ?)`,
			i+1, now)
	}
	svc.db.Exec(`INSERT INTO ai_trace (id, agent_id, confidence) VALUES (1, 'A1', 0.95)`)

	// AutoUpgrade calls Recalculate then UpgradeEligible.
	// A1 has trust=0.790 (advisory -> guided), so one upgrade is expected.
	results, err := upgrader.AutoUpgrade()
	if err != nil {
		t.Fatalf("AutoUpgrade() error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 upgrade (A1 advisory->guided), got %d", len(results))
	}

	// Verify Recalculate created all agent records
	scores, _ := svc.List()
	if len(scores) != len(defaultAgentList()) {
		t.Errorf("expected %d agent records after Recalculate, got %d", len(defaultAgentList()), len(scores))
	}
}

// ---------------------------------------------------------------------------
// Handler tests
// ---------------------------------------------------------------------------

func TestHandler_List(t *testing.T) {
	svc, r := setupRouter(t)

	// Create a score so the list is non-empty
	score := NewTrustScore("A1", "Scout", "autonomous")
	svc.db.Create(score)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/trust-scores", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result response.Result
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Code != 0 {
		t.Errorf("expected code 0, got %d", result.Code)
	}
	if result.Message != "ok" {
		t.Errorf("expected 'ok', got %s", result.Message)
	}
	if result.Data == nil {
		t.Fatal("expected data, got nil")
	}
}

func TestHandler_GetByAgent_Found(t *testing.T) {
	svc, r := setupRouter(t)

	score := NewTrustScore("A1", "Scout", "autonomous")
	if err := svc.db.Create(score).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/trust-scores/A1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result response.Result
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Code != 0 {
		t.Errorf("expected code 0, got %d", result.Code)
	}
	if result.Data == nil {
		t.Error("expected data, got nil")
	}
}

func TestHandler_GetByAgent_NotFound(t *testing.T) {
	_, r := setupRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/trust-scores/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var result response.Result
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Code != 404 {
		t.Errorf("expected code 404, got %d", result.Code)
	}
	if result.Message != "trust score not found" {
		t.Errorf("expected 'trust score not found', got %s", result.Message)
	}
}

func TestHandler_Recalculate(t *testing.T) {
	svc, r := setupRouter(t)

	svc.db.Exec(`CREATE TABLE unified_action (
		id INTEGER, agent_id TEXT, status TEXT,
		rejected_by TEXT, approved_by TEXT, proposed_at TIMESTAMP
	)`)
	svc.db.Exec(`CREATE TABLE ai_trace (
		id INTEGER, agent_id TEXT, confidence REAL
	)`)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/trust-scores/recalculate", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Code != 0 {
		t.Errorf("expected code 0, got %d", result.Code)
	}
	if result.Data.Status != "ok" {
		t.Errorf("expected 'ok', got %s", result.Data.Status)
	}
}

func TestHandler_Eligible(t *testing.T) {
	svc, r := setupRouter(t)

	// Create an eligible agent
	score := NewTrustScore("A1", "Scout", "autonomous")
	score.TrustScore = 0.70
	score.AutonomyLevel = "guided"
	svc.db.Create(score)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/trust-scores/eligible", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result response.Result
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Code != 0 {
		t.Errorf("expected code 0, got %d", result.Code)
	}

	// Unmarshal data as []TrustScore
	dataJSON, _ := json.Marshal(result.Data)
	var scores []TrustScore
	if err := json.Unmarshal(dataJSON, &scores); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if len(scores) != 1 || scores[0].AgentID != "A1" {
		t.Errorf("expected [A1], got %+v", scores)
	}
}

func TestHandler_UpdateLevel(t *testing.T) {
	svc, r := setupRouter(t)

	score := NewTrustScore("A1", "Scout", "autonomous")
	svc.db.Create(score)

	body := `{"level":"supervised"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/trust-scores/A1/level", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Code != 0 {
		t.Errorf("expected code 0, got %d", result.Code)
	}
	if result.Data.Status != "ok" {
		t.Errorf("expected 'ok', got %s", result.Data.Status)
	}

	// Verify DB was updated
	updated, _ := svc.GetByAgent("A1")
	if updated.AutonomyLevel != "supervised" {
		t.Errorf("DB autonomy_level: want supervised, got %s", updated.AutonomyLevel)
	}
}

func TestHandler_UpdateLevel_MissingBody(t *testing.T) {
	_, r := setupRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/trust-scores/A1/level", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_AutoUpgrade(t *testing.T) {
	svc, r := setupRouter(t)

	// Create external tables so Recalculate doesn't error
	svc.db.Exec(`CREATE TABLE unified_action (
		id INTEGER, agent_id TEXT, status TEXT,
		rejected_by TEXT, approved_by TEXT, proposed_at TIMESTAMP
	)`)
	svc.db.Exec(`CREATE TABLE ai_trace (
		id INTEGER, agent_id TEXT, confidence REAL
	)`)
	svc.db.Exec(`CREATE TABLE listing_recommendation (
		id INTEGER, product_id INTEGER, triggered_by TEXT, feedback_status TEXT
	)`)

	_ = svc

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/trust-scores/auto-upgrade", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result response.Result
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Code != 0 {
		t.Errorf("expected code 0, got %d", result.Code)
	}
}

func TestHandler_Summary(t *testing.T) {
	svc, r := setupRouter(t)

	score := NewTrustScore("A1", "Product Scout", "autonomous")
	if err := svc.db.Create(score).Error; err != nil {
		t.Fatalf("create A1: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/trust-scores/summary", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result response.Result
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Code != 0 {
		t.Errorf("expected code 0, got %d", result.Code)
	}
	if result.Data == nil {
		t.Fatal("expected data, got nil")
	}

	// Verify data contains populated summary items
	dataJSON, _ := json.Marshal(result.Data)
	var items []AutonomySummaryItem
	if err := json.Unmarshal(dataJSON, &items); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one summary item")
	}
	if items[0].AgentID != "A1" {
		t.Errorf("expected A1, got %s", items[0].AgentID)
	}
	// A1 exists in the registry -> Description should be populated
	if items[0].Description == "" {
		t.Error("expected non-empty Description from registry for A1")
	}
	if items[0].TotalActions != 0 {
		t.Errorf("expected 0 total actions, got %d", items[0].TotalActions)
	}
}

// ---------------------------------------------------------------------------
// RecordAgentFeedback tests
// ---------------------------------------------------------------------------

func TestRecordAgentFeedback_NoData(t *testing.T) {
	svc := newTestDB(t)

	// Create listing_recommendation table
	svc.db.Exec(`CREATE TABLE listing_recommendation (
		id INTEGER, product_id INTEGER, triggered_by TEXT, feedback_status TEXT
	)`)
	svc.db.Exec(`CREATE TABLE unified_action (
		id INTEGER, agent_id TEXT, status TEXT,
		rejected_by TEXT, approved_by TEXT, proposed_at TIMESTAMP
	)`)
	svc.db.Exec(`CREATE TABLE ai_trace (
		id INTEGER, agent_id TEXT, confidence REAL
	)`)

	// Record feedback for A1 with no listing feedback data
	if err := svc.RecordAgentFeedback("A1"); err != nil {
		t.Fatalf("RecordAgentFeedback(A1) error: %v", err)
	}

	score, err := svc.GetByAgent("A1")
	if err != nil {
		t.Fatalf("GetByAgent error: %v", err)
	}
	if score == nil {
		t.Fatal("expected score to be created")
	}
	if score.FeedbackAdopted != 0 {
		t.Errorf("feedback_adopted: want 0, got %d", score.FeedbackAdopted)
	}
	if score.FeedbackRejected != 0 {
		t.Errorf("feedback_rejected: want 0, got %d", score.FeedbackRejected)
	}
	// No listing feedback data -> listing_feedback_rate = 0
	if score.ListingFeedbackRate != 0 {
		t.Errorf("listing_feedback_rate: want 0, got %f", score.ListingFeedbackRate)
	}
}

func TestRecordAgentFeedback_WithData(t *testing.T) {
	svc := newTestDB(t)

	// Create listing_recommendation table
	svc.db.Exec(`CREATE TABLE listing_recommendation (
		id INTEGER, product_id INTEGER, triggered_by TEXT, feedback_status TEXT
	)`)
	svc.db.Exec(`CREATE TABLE unified_action (
		id INTEGER, agent_id TEXT, status TEXT,
		rejected_by TEXT, approved_by TEXT, proposed_at TIMESTAMP
	)`)
	svc.db.Exec(`CREATE TABLE ai_trace (
		id INTEGER, agent_id TEXT, confidence REAL
	)`)

	now := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)

	// Insert listing recommendations with feedback for A1
	// 3 adopted, 1 rejected
	svc.db.Exec(`INSERT INTO listing_recommendation (id, product_id, triggered_by, feedback_status) VALUES (1, 1, 'A1', 'adopted')`)
	svc.db.Exec(`INSERT INTO listing_recommendation (id, product_id, triggered_by, feedback_status) VALUES (2, 2, 'A1', 'adopted')`)
	svc.db.Exec(`INSERT INTO listing_recommendation (id, product_id, triggered_by, feedback_status) VALUES (3, 3, 'A1', 'adopted')`)
	svc.db.Exec(`INSERT INTO listing_recommendation (id, product_id, triggered_by, feedback_status) VALUES (4, 4, 'A1', 'rejected')`)

	// Insert some action and trace data so the trust score is recalculated
	for i := 0; i < 5; i++ {
		svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, approved_by, proposed_at) VALUES (?, 'A1', 'approved', 'policy', ?)`, i+1, now)
	}
	svc.db.Exec(`INSERT INTO ai_trace (id, agent_id, confidence) VALUES (1, 'A1', 0.90)`)

	// RecordAgentFeedback for A1 (first call creates record with default values)
	if err := svc.RecordAgentFeedback("A1"); err != nil {
		t.Fatalf("first RecordAgentFeedback(A1) error: %v", err)
	}

	score, err := svc.GetByAgent("A1")
	if err != nil {
		t.Fatalf("GetByAgent error: %v", err)
	}
	if score == nil {
		t.Fatal("expected score to exist")
	}

	// First call creates record with zero defaults
	if score.FeedbackAdopted != 0 {
		t.Logf("first call: feedback_adopted=%d (expected 0 as new record)", score.FeedbackAdopted)
	}

	// Call again to trigger the update with computed values
	if err := svc.RecordAgentFeedback("A1"); err != nil {
		t.Fatalf("second RecordAgentFeedback(A1) error: %v", err)
	}

	score, _ = svc.GetByAgent("A1")

	// Verify feedback counts
	if score.FeedbackAdopted != 3 {
		t.Errorf("feedback_adopted: want 3, got %d", score.FeedbackAdopted)
	}
	if score.FeedbackRejected != 1 {
		t.Errorf("feedback_rejected: want 1, got %d", score.FeedbackRejected)
	}

	// listing_feedback_rate = 3 / (3+1) = 0.75
	if !approx(score.ListingFeedbackRate, 0.75, 0.0005) {
		t.Errorf("listing_feedback_rate: want ~0.75, got %f", score.ListingFeedbackRate)
	}

	// Trust score includes listing feedback rate factor (20%)
	// adoptionRate=1.0*0.35 + execSuccess=1.0*0.25 + avgConf=0.90*0.20 + listingFeedbackRate=0.75*0.20
	// = 0.35 + 0.25 + 0.18 + 0.15 = 0.93
	if !approx(score.TrustScore, 0.93, 0.0005) {
		t.Errorf("trust_score: want ~0.93, got %f", score.TrustScore)
	}
}

func TestRecalculate_WithListingFeedback(t *testing.T) {
	svc := newTestDB(t)

	// Create external tables
	svc.db.Exec(`CREATE TABLE unified_action (
		id INTEGER, agent_id TEXT, status TEXT,
		rejected_by TEXT, approved_by TEXT, proposed_at TIMESTAMP
	)`)
	svc.db.Exec(`CREATE TABLE ai_trace (
		id INTEGER, agent_id TEXT, confidence REAL
	)`)
	svc.db.Exec(`CREATE TABLE listing_recommendation (
		id INTEGER, product_id INTEGER, triggered_by TEXT, feedback_status TEXT
	)`)

	now := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)

	// A1: 3 adopted, 1 rejected in listing feedback
	svc.db.Exec(`INSERT INTO listing_recommendation (id, product_id, triggered_by, feedback_status) VALUES (1, 1, 'A1', 'adopted')`)
	svc.db.Exec(`INSERT INTO listing_recommendation (id, product_id, triggered_by, feedback_status) VALUES (2, 2, 'A1', 'adopted')`)
	svc.db.Exec(`INSERT INTO listing_recommendation (id, product_id, triggered_by, feedback_status) VALUES (3, 3, 'A1', 'adopted')`)
	svc.db.Exec(`INSERT INTO listing_recommendation (id, product_id, triggered_by, feedback_status) VALUES (4, 4, 'A1', 'rejected')`)
	for i := 0; i < 10; i++ {
		svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, approved_by, proposed_at) VALUES (?, 'A1', 'approved', 'policy', ?)`, i+1, now)
	}
	svc.db.Exec(`INSERT INTO ai_trace (id, agent_id, confidence) VALUES (1, 'A1', 0.95)`)

	// A2: no listing feedback
	for i := 0; i < 5; i++ {
		svc.db.Exec(`INSERT INTO unified_action (id, agent_id, status, approved_by, proposed_at) VALUES (?, 'A2', 'approved', 'policy', ?)`, i+11, now)
	}
	svc.db.Exec(`INSERT INTO ai_trace (id, agent_id, confidence) VALUES (2, 'A2', 0.90)`)

	// Run Recalculate
	if err := svc.Recalculate(); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	// Verify A1 with listing feedback
	a1, _ := svc.GetByAgent("A1")
	if a1.FeedbackAdopted != 3 {
		t.Errorf("A1 feedback_adopted: want 3, got %d", a1.FeedbackAdopted)
	}
	if a1.FeedbackRejected != 1 {
		t.Errorf("A1 feedback_rejected: want 1, got %d", a1.FeedbackRejected)
	}
	if !approx(a1.ListingFeedbackRate, 0.75, 0.0005) {
		t.Errorf("A1 listing_feedback_rate: want ~0.75, got %f", a1.ListingFeedbackRate)
	}

	// Verify A2 has no listing feedback
	a2, _ := svc.GetByAgent("A2")
	if a2.FeedbackAdopted != 0 {
		t.Errorf("A2 feedback_adopted: want 0, got %d", a2.FeedbackAdopted)
	}
	if a2.FeedbackRejected != 0 {
		t.Errorf("A2 feedback_rejected: want 0, got %d", a2.FeedbackRejected)
	}
	if a2.ListingFeedbackRate != 0 {
		t.Errorf("A2 listing_feedback_rate: want 0, got %f", a2.ListingFeedbackRate)
	}
}
