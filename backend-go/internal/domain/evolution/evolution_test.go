package evolution

import (
	"strings"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Nudge{}, &trustscore.TrustScore{})
}

func newService(t *testing.T, db *gorm.DB) *Service {
	t.Helper()
	return NewService(db, dbtest.NewLogger(t))
}

func createNudge(t *testing.T, db *gorm.DB, agentID, status, current, target string) *Nudge {
	t.Helper()
	n := &Nudge{
		UserID:       1,
		AgentID:      agentID,
		CurrentLevel: current,
		TargetLevel:  target,
		TrustScore:   0.6,
		Status:       status,
		Message:      "test nudge",
	}
	if err := db.Create(n).Error; err != nil {
		t.Fatalf("createNudge failed: %v", err)
	}
	return n
}

func createTrustScore(t *testing.T, db *gorm.DB, agentID, level string, score float64) {
	t.Helper()
	ts := &trustscore.TrustScore{
		AgentID:       agentID,
		AgentName:     "Test Agent",
		SquadID:       "test",
		AutonomyLevel: level,
		TrustScore:    score,
		TotalActions:  100,
	}
	if err := db.Create(ts).Error; err != nil {
		t.Fatalf("createTrustScore failed: %v", err)
	}
}

func TestEvolution_ListNudges_All(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	createNudge(t, db, "A1", "pending", "advisory", "guided")
	createNudge(t, db, "A2", "accepted", "guided", "supervised")
	createNudge(t, db, "A1", "dismissed", "advisory", "guided")

	nudges, err := svc.ListNudges(nil, "", "")
	if err != nil {
		t.Fatalf("ListNudges failed: %v", err)
	}
	if len(nudges) != 3 {
		t.Fatalf("ListNudges returned %d, want 3", len(nudges))
	}
}

func TestEvolution_ListNudges_FilterAgent(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	createNudge(t, db, "A1", "pending", "advisory", "guided")
	createNudge(t, db, "A2", "pending", "guided", "supervised")

	nudges, err := svc.ListNudges(nil, "A1", "")
	if err != nil {
		t.Fatalf("ListNudges failed: %v", err)
	}
	if len(nudges) != 1 {
		t.Fatalf("expected 1 nudge, got %d", len(nudges))
	}
	if nudges[0].AgentID != "A1" {
		t.Fatalf("AgentID = %q, want %q", nudges[0].AgentID, "A1")
	}
}

func TestEvolution_ListNudges_FilterStatus(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	createNudge(t, db, "A1", "pending", "advisory", "guided")
	createNudge(t, db, "A2", "accepted", "guided", "supervised")

	nudges, err := svc.ListNudges(nil, "", "pending")
	if err != nil {
		t.Fatalf("ListNudges failed: %v", err)
	}
	if len(nudges) != 1 {
		t.Fatalf("expected 1 pending nudge, got %d", len(nudges))
	}
	if nudges[0].Status != "pending" {
		t.Fatalf("Status = %q, want %q", nudges[0].Status, "pending")
	}
}

func TestEvolution_ListNudges_FilterUser(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	n1 := &Nudge{UserID: 1, AgentID: "A1", CurrentLevel: "advisory", TargetLevel: "guided", Status: "pending", Message: "m1"}
	n2 := &Nudge{UserID: 2, AgentID: "A1", CurrentLevel: "advisory", TargetLevel: "guided", Status: "pending", Message: "m2"}
	db.Create(n1)
	db.Create(n2)

	uid := int64(1)
	nudges, err := svc.ListNudges(&uid, "", "")
	if err != nil {
		t.Fatalf("ListNudges failed: %v", err)
	}
	if len(nudges) != 1 {
		t.Fatalf("expected 1 nudge for user 1, got %d", len(nudges))
	}
}

func TestEvolution_AcceptNudge(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	createTrustScore(t, db, "A1", "advisory", 0.4)
	n := createNudge(t, db, "A1", "pending", "advisory", "guided")

	if err := svc.AcceptNudge(n.ID); err != nil {
		t.Fatalf("AcceptNudge failed: %v", err)
	}

	// Verify nudge status updated
	var updated Nudge
	db.First(&updated, n.ID)
	if updated.Status != "accepted" {
		t.Fatalf("nudge Status = %q, want %q", updated.Status, "accepted")
	}
	if updated.DecidedAt == nil {
		t.Fatal("expected non-nil DecidedAt after accept")
	}

	// Verify trust score autonomy level upgraded
	var score trustscore.TrustScore
	db.Where("agent_id = ?", "A1").First(&score)
	if score.AutonomyLevel != "guided" {
		t.Fatalf("autonomy level = %q, want %q", score.AutonomyLevel, "guided")
	}
}

func TestEvolution_AcceptNudge_NotPending(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	n := createNudge(t, db, "A1", "accepted", "advisory", "guided")

	if err := svc.AcceptNudge(n.ID); err == nil {
		t.Fatal("expected error for non-pending nudge")
	}
}

func TestEvolution_AcceptNudge_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	if err := svc.AcceptNudge(999); err == nil {
		t.Fatal("expected error for non-existent nudge")
	}
}

func TestEvolution_DismissNudge(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	n := createNudge(t, db, "A1", "pending", "advisory", "guided")

	if err := svc.DismissNudge(n.ID); err != nil {
		t.Fatalf("DismissNudge failed: %v", err)
	}

	var updated Nudge
	db.First(&updated, n.ID)
	if updated.Status != "dismissed" {
		t.Fatalf("Status = %q, want %q", updated.Status, "dismissed")
	}
	if updated.DecidedAt == nil {
		t.Fatal("expected non-nil DecidedAt after dismiss")
	}
}

func TestEvolution_DismissNudge_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newService(t, db)

	// DismissNudge uses GORM Updates with WHERE clause; SQLite doesn't error
	// when no rows match. Verify no panic and nudge list remains empty.
	err := svc.DismissNudge(999)
	if err != nil {
		t.Logf("DismissNudge returned error for non-existent ID (acceptable): %v", err)
	}
	nudges, _ := svc.ListNudges(nil, "", "dismissed")
	if len(nudges) != 0 {
		t.Fatalf("expected 0 dismissed nudges, got %d", len(nudges))
	}
}

func TestEvolution_NextAutonomyLevel(t *testing.T) {
	tests := []struct {
		current  string
		expected string
	}{
		{"advisory", "guided"},
		{"guided", "supervised"},
		{"supervised", "autonomous"},
		{"autonomous", ""},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.current, func(t *testing.T) {
			got := nextAutonomyLevel(tt.current)
			if got != tt.expected {
				t.Fatalf("nextAutonomyLevel(%q) = %q, want %q", tt.current, got, tt.expected)
			}
		})
	}
}

func TestEvolution_UpgradeMessage(t *testing.T) {
	msg := upgradeMessage("A1", "Scout", "advisory", "guided", 0.45)
	if msg == "" {
		t.Fatal("expected non-empty message")
	}
	if !strings.Contains(msg, "Scout") {
		t.Fatalf("message should contain agent name, got: %s", msg)
	}
	if !strings.Contains(msg, "A1") {
		t.Fatalf("message should contain agent ID, got: %s", msg)
	}
}

func TestEvolution_UpgradeMessage_AllLevels(t *testing.T) {
	levels := []struct {
		from, to string
		score    float64
	}{
		{"advisory", "guided", 0.4},
		{"guided", "supervised", 0.6},
		{"supervised", "autonomous", 0.85},
	}
	for _, l := range levels {
		msg := upgradeMessage("A1", "Agent", l.from, l.to, l.score)
		if msg == "" {
			t.Fatalf("empty message for %s -> %s", l.from, l.to)
		}
	}
}
