package evolution

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
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
