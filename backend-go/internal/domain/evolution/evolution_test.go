package evolution

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
)

func TestService_NudgeCRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Nudge{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create a nudge directly since EvaluateNudges needs trustscore table + AI registry
	db.Create(&Nudge{
		UserID:       1,
		AgentID:      "A5",
		CurrentLevel: "guided",
		TargetLevel:  "supervised",
		TrustScore:   0.85,
		Status:       "pending",
		Message:      "test nudge",
	})

	// ListNudges
	nudges, err := svc.ListNudges(nil, "A5", "")
	if err != nil {
		t.Fatalf("ListNudges: %v", err)
	}
	if len(nudges) != 1 {
		t.Fatalf("nudges = %d (expected 1)", len(nudges))
	}
	if nudges[0].TargetLevel != "supervised" {
		t.Fatalf("TargetLevel = %s", nudges[0].TargetLevel)
	}

	// ListNudges with status filter
	nudges, err = svc.ListNudges(nil, "A5", "pending")
	if err != nil {
		t.Fatalf("ListNudges pending: %v", err)
	}
	if len(nudges) != 1 {
		t.Fatalf("pending nudges = %d", len(nudges))
	}

	// AcceptNudge should fail since there's no trustscore table
	// (The transaction inside AcceptNudge tries to update trustscore.TrustScore.)
	// So let's just test that AcceptNudge correctly validates status

	// DismissNudge
	err = svc.DismissNudge(nudges[0].ID)
	if err != nil {
		t.Fatalf("DismissNudge: %v", err)
	}
	nudges, _ = svc.ListNudges(nil, "A5", "dismissed")
	if len(nudges) != 1 {
		t.Fatalf("dismissed nudges = %d", len(nudges))
	}
}

func TestService_AcceptNudge_NotPending(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Nudge{})
	svc := NewService(db, dbtest.NewLogger(t))

	now := time.Now()
	db.Create(&Nudge{
		UserID: 1, AgentID: "A5",
		CurrentLevel: "guided", TargetLevel: "supervised",
		Status: "dismissed", DecidedAt: &now,
	})

	err := svc.AcceptNudge(1)
	if err == nil {
		t.Fatal("expected error for non-pending nudge")
	}
}

func TestService_ListByUser(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Nudge{})
	svc := NewService(db, dbtest.NewLogger(t))

	userID := int64(1)
	db.Create(&Nudge{UserID: 1, AgentID: "A5", CurrentLevel: "advisory", TargetLevel: "guided", Status: "pending"})
	db.Create(&Nudge{UserID: 2, AgentID: "A5", CurrentLevel: "guided", TargetLevel: "supervised", Status: "pending"})

	nudges, err := svc.ListNudges(&userID, "", "pending")
	if err != nil {
		t.Fatalf("ListNudges: %v", err)
	}
	if len(nudges) != 1 {
		t.Fatalf("nudges for user 1 = %d", len(nudges))
	}
}

// --- nextAutonomyLevel tests ---

func TestNextAutonomyLevel_Advisory(t *testing.T) {
	t.Parallel()
	got := nextAutonomyLevel("advisory")
	if got != "guided" {
		t.Fatalf("nextAutonomyLevel(advisory) = %q; want %q", got, "guided")
	}
}

func TestNextAutonomyLevel_Guided(t *testing.T) {
	t.Parallel()
	got := nextAutonomyLevel("guided")
	if got != "supervised" {
		t.Fatalf("nextAutonomyLevel(guided) = %q; want %q", got, "supervised")
	}
}

func TestNextAutonomyLevel_Supervised(t *testing.T) {
	t.Parallel()
	got := nextAutonomyLevel("supervised")
	if got != "autonomous" {
		t.Fatalf("nextAutonomyLevel(supervised) = %q; want %q", got, "autonomous")
	}
}

func TestNextAutonomyLevel_Autonomous(t *testing.T) {
	t.Parallel()
	got := nextAutonomyLevel("autonomous")
	if got != "" {
		t.Fatalf("nextAutonomyLevel(autonomous) = %q; want empty string", got)
	}
}

func TestNextAutonomyLevel_Unknown(t *testing.T) {
	t.Parallel()
	got := nextAutonomyLevel("unknown")
	if got != "" {
		t.Fatalf("nextAutonomyLevel(unknown) = %q; want empty string", got)
	}
}

// --- upgradeMessage tests ---

func TestUpgradeMessage(t *testing.T) {
	t.Parallel()

	msg := upgradeMessage("A5", "库存Agent", "advisory", "guided", 0.35)
	want := "库存Agent (A5) 的信任分已达 35.0%，建议从「观察」升级为「引导」—— 可创建待审批的动作"
	if msg != want {
		t.Fatalf("upgradeMessage = %q; want %q", msg, want)
	}

	msg = upgradeMessage("G3", "折扣风险", "guided", "supervised", 0.62)
	want = "折扣风险 (G3) 的信任分已达 62.0%，建议从「引导」升级为「监督」—— 动作自动创建但仍需审批"
	if msg != want {
		t.Fatalf("upgradeMessage = %q; want %q", msg, want)
	}

	msg = upgradeMessage("A2", "Listing优化", "supervised", "autonomous", 0.91)
	want = "Listing优化 (A2) 的信任分已达 91.0%，建议从「监督」升级为「自主」—— 低风险动作可自动执行"
	if msg != want {
		t.Fatalf("upgradeMessage = %q; want %q", msg, want)
	}

	// Unknown target level uses fallback template.
	msg = upgradeMessage("X1", "未知", "advisory", "unknown_level", 0.50)
	want = "未知 (X1) 的信任分已达 50.0%，建议升级自主度"
	if msg != want {
		t.Fatalf("upgradeMessage(unknown) = %q; want %q", msg, want)
	}
}

// --- AcceptNudge success path ---

func TestAcceptNudge_Success(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Nudge{}, &trustscore.TrustScore{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&trustscore.TrustScore{
		AgentID:       "A5",
		AgentName:     "库存Agent",
		AutonomyLevel: "advisory",
		TrustScore:    0.35,
	})
	db.Create(&Nudge{
		UserID: 1, AgentID: "A5",
		CurrentLevel: "advisory", TargetLevel: "guided",
		TrustScore: 0.35, Status: "pending",
	})

	if err := svc.AcceptNudge(1); err != nil {
		t.Fatalf("AcceptNudge: %v", err)
	}

	// Verify nudge status updated.
	var n Nudge
	db.First(&n, 1)
	if n.Status != "accepted" {
		t.Fatalf("nudge status = %q; want %q", n.Status, "accepted")
	}
	if n.DecidedAt == nil {
		t.Fatal("DecidedAt should be set after acceptance")
	}

	// Verify trust score autonomy level upgraded.
	var ts trustscore.TrustScore
	db.Where("agent_id = ?", "A5").First(&ts)
	if ts.AutonomyLevel != "guided" {
		t.Fatalf("autonomy level = %q; want %q", ts.AutonomyLevel, "guided")
	}
}

// --- DismissNudge success path ---

func TestDismissNudge_Success(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Nudge{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Nudge{
		UserID: 1, AgentID: "A5",
		CurrentLevel: "advisory", TargetLevel: "guided",
		Status: "pending",
	})

	if err := svc.DismissNudge(1); err != nil {
		t.Fatalf("DismissNudge: %v", err)
	}

	var n Nudge
	db.First(&n, 1)
	if n.Status != "dismissed" {
		t.Fatalf("nudge status = %q; want %q", n.Status, "dismissed")
	}
	if n.DecidedAt == nil {
		t.Fatal("DecidedAt should be set after dismissal")
	}
}

// --- CreateNudge default status ---

func TestCreateNudge_DefaultPending(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Nudge{})

	// Create without explicit status — GORM should apply default:pending.
	n := Nudge{
		UserID: 1, AgentID: "A5",
		CurrentLevel: "advisory", TargetLevel: "guided",
		TrustScore: 0.40, Message: "test",
	}
	if err := db.Create(&n).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}

	var saved Nudge
	db.First(&saved, n.ID)
	if saved.Status != "pending" {
		t.Fatalf("default status = %q; want %q", saved.Status, "pending")
	}
}

// --- ListNudges filter by status only (no agentID) ---

func TestListNudges_FilterByStatusOnly(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Nudge{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Nudge{UserID: 1, AgentID: "A5", CurrentLevel: "advisory", TargetLevel: "guided", Status: "pending"})
	db.Create(&Nudge{UserID: 2, AgentID: "G3", CurrentLevel: "guided", TargetLevel: "supervised", Status: "pending"})
	db.Create(&Nudge{UserID: 3, AgentID: "A6", CurrentLevel: "supervised", TargetLevel: "autonomous", Status: "dismissed"})

	nudges, err := svc.ListNudges(nil, "", "pending")
	if err != nil {
		t.Fatalf("ListNudges: %v", err)
	}
	if len(nudges) != 2 {
		t.Fatalf("pending nudges = %d; want 2", len(nudges))
	}

	nudges, err = svc.ListNudges(nil, "", "dismissed")
	if err != nil {
		t.Fatalf("ListNudges: %v", err)
	}
	if len(nudges) != 1 {
		t.Fatalf("dismissed nudges = %d; want 1", len(nudges))
	}
}
