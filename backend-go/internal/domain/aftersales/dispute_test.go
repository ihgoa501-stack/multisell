package aftersales

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

// mockDeliveredChecker always reports as delivered.
type mockDeliveredChecker struct{}

func (m *mockDeliveredChecker) IsDelivered(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

// mockNotDeliveredChecker always reports as not delivered.
type mockNotDeliveredChecker struct{}

func (m *mockNotDeliveredChecker) IsDelivered(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func TestDisputeService_CreateCase(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})
	ds := NewDisputeService(db, testLogger(), nil)

	dc, err := ds.CreateCase(context.Background(), &CreateDisputeInput{
		TransactionID: "TXN-12345",
		Platform:      "shopee",
		ClaimType:     "not_received",
		Amount:        35.00,
		Evidence:      `{"tracking":"TRK001","buyer_note":"haven't received"}`,
	})
	if err != nil {
		t.Fatalf("CreateCase failed: %v", err)
	}
	if dc.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if dc.Status != "pending" {
		t.Errorf("expected default status pending, got %s", dc.Status)
	}
	if dc.TransactionID != "TXN-12345" {
		t.Errorf("expected TransactionID TXN-12345, got %s", dc.TransactionID)
	}
	if dc.Platform != "shopee" {
		t.Errorf("expected Platform shopee, got %s", dc.Platform)
	}
}

func TestDisputeService_GetCase_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})
	ds := NewDisputeService(db, testLogger(), nil)

	_, err := ds.GetCase(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for not found dispute case")
	}
}

func TestDisputeService_ListCases(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})
	ds := NewDisputeService(db, testLogger(), nil)

	for i := 0; i < 3; i++ {
		platform := "shopee"
		claimType := "not_received"
		if i == 1 {
			platform = "lazada"
			claimType = "damaged"
		}
		_, err := ds.CreateCase(context.Background(), &CreateDisputeInput{
			TransactionID: "TXN-" + itoa(int64(i+100)),
			Platform:      platform,
			ClaimType:     claimType,
			Amount:        50.0,
		})
		if err != nil {
			t.Fatalf("CreateCase failed: %v", err)
		}
	}

	// Test unfiltered list
	items, total, err := ds.ListCases(context.Background(), &common.Pagination{Page: 1, Size: 10}, nil)
	if err != nil {
		t.Fatalf("ListCases failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}

	// Test filter by platform
	items, total, err = ds.ListCases(context.Background(), &common.Pagination{Page: 1, Size: 10}, &DisputeListFilter{Platform: "lazada"})
	if err != nil {
		t.Fatalf("ListCases filter by platform failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 lazada dispute, got %d", total)
	}
}

func TestDisputeService_EvaluateCase_NotReceivedDelivered(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})

	// Use the mock that says delivered
	deliveredChecker := &mockDeliveredChecker{}
	ds := NewDisputeService(db, testLogger(), deliveredChecker)

	dc, err := ds.CreateCase(context.Background(), &CreateDisputeInput{
		TransactionID: "TXN-DLV-001",
		Platform:      "shopee",
		ClaimType:     "not_received",
		Amount:        25.00,
	})
	if err != nil {
		t.Fatalf("CreateCase failed: %v", err)
	}

	result, err := ds.EvaluateCase(context.Background(), dc.ID)
	if err != nil {
		t.Fatalf("EvaluateCase failed: %v", err)
	}

	// baseline 50
	// claim_type not_received -> +5 -> 55
	// delivered -> -30 -> 25
	// amount < 50 -> +20 -> 45
	expectedScore := 45.0
	if result.Score != expectedScore {
		t.Errorf("expected Score %.0f, got %.0f", expectedScore, result.Score)
	}

	// Score 45 is between 25 and 75 => manual_review
	if result.Decision != "manual_review" {
		t.Errorf("expected decision manual_review for score %.0f, got %s", result.Score, result.Decision)
	}

	// Verify persistence
	updated, err := ds.GetCase(context.Background(), dc.ID)
	if err != nil {
		t.Fatalf("GetCase failed: %v", err)
	}
	if updated.DecisionScore != expectedScore {
		t.Errorf("expected persisted DecisionScore %.0f, got %.0f", expectedScore, updated.DecisionScore)
	}
	if updated.DecisionSource != "rule" {
		t.Errorf("expected DecisionSource rule, got %s", updated.DecisionSource)
	}
}

func TestDisputeService_EvaluateCase_DamagedLowAmount(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})
	ds := NewDisputeService(db, testLogger(), nil)

	dc, err := ds.CreateCase(context.Background(), &CreateDisputeInput{
		TransactionID: "TXN-DMG-001",
		Platform:      "lazada",
		ClaimType:     "damaged",
		Amount:        20.00,
	})
	if err != nil {
		t.Fatalf("CreateCase failed: %v", err)
	}

	result, err := ds.EvaluateCase(context.Background(), dc.ID)
	if err != nil {
		t.Fatalf("EvaluateCase failed: %v", err)
	}

	// baseline 50
	// amount < 50 -> +20 -> 70
	// claim_type "damaged" -> +10 -> 80
	if result.Score != 80 {
		t.Errorf("expected Score 80, got %.0f", result.Score)
	}
	if result.Decision != "approved" {
		t.Errorf("expected decision approved, got %s", result.Decision)
	}
}

func TestDisputeService_EvaluateCase_HighAmountChangeOfMind(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})
	ds := NewDisputeService(db, testLogger(), nil)

	dc, err := ds.CreateCase(context.Background(), &CreateDisputeInput{
		TransactionID: "TXN-HGH-001",
		Platform:      "ozon",
		ClaimType:     "change_of_mind",
		Amount:        600.00,
	})
	if err != nil {
		t.Fatalf("CreateCase failed: %v", err)
	}

	result, err := ds.EvaluateCase(context.Background(), dc.ID)
	if err != nil {
		t.Fatalf("EvaluateCase failed: %v", err)
	}

	// baseline 50
	// amount > 500 -> -10 -> 40
	// claim_type "change_of_mind" -> -10 -> 30
	if result.Score != 30 {
		t.Errorf("expected Score 30, got %.0f", result.Score)
	}
	if result.Decision != "manual_review" {
		t.Errorf("expected decision manual_review for score 30, got %s", result.Decision)
	}
}

func TestDisputeService_AutoDecide_ApprovesLowRisk(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})
	ds := NewDisputeService(db, testLogger(), nil)

	dc, err := ds.CreateCase(context.Background(), &CreateDisputeInput{
		TransactionID: "TXN-AUTO-001",
		Platform:      "shopee",
		ClaimType:     "wrong_item", // +15
		Amount:        20.00,        // +20 (low amount)
	})
	if err != nil {
		t.Fatalf("CreateCase failed: %v", err)
	}

	result, err := ds.AutoDecide(context.Background(), dc.ID)
	if err != nil {
		t.Fatalf("AutoDecide failed: %v", err)
	}

	// baseline 50 + wrong_item 15 + low amount 20 = 85
	if result.Decision != "approved" {
		t.Errorf("expected decision approved, got %s, score=%.0f", result.Decision, result.Score)
	}
	if result.Dispute.Status != "approved" {
		t.Errorf("expected dispute Status approved, got %s", result.Dispute.Status)
	}
}

func TestDisputeService_AutoDecide_RejectsHighRisk(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})

	// Mock that says delivered.
	deliveredChecker := &mockDeliveredChecker{}
	ds := NewDisputeService(db, testLogger(), deliveredChecker)

	dc, err := ds.CreateCase(context.Background(), &CreateDisputeInput{
		TransactionID: "TXN-REJ-001",
		Platform:      "shopee",
		ClaimType:     "not_received",
		// Amount=0 to avoid the low-amount bonus, so:
		// baseline 50 - 30 (delivered penalty) + 5 (not_received) = 25
	})
	if err != nil {
		t.Fatalf("CreateCase failed: %v", err)
	}

	result, err := ds.AutoDecide(context.Background(), dc.ID)
	if err != nil {
		t.Fatalf("AutoDecide failed: %v", err)
	}

	// baseline 50
	// delivered + not_received -> -30 -> 20
	// amount=0 -> no amount rule -> unchanged at 20
	// claim_type "not_received" -> +5 -> 25
	// Score 25 <= 25 => rejected
	expectedScore := 25.0
	if result.Score != expectedScore {
		t.Errorf("expected Score %.0f for auto-reject case, got %.0f", expectedScore, result.Score)
	}
	if result.Decision != "rejected" {
		t.Errorf("expected decision rejected for score %.0f, got %s", result.Score, result.Decision)
	}
	if result.Dispute.Status != "rejected" {
		t.Errorf("expected dispute Status rejected, got %s", result.Dispute.Status)
	}
}

func TestDisputeService_AutoDecide_FlagsForManualReview(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})
	ds := NewDisputeService(db, testLogger(), nil)

	dc, err := ds.CreateCase(context.Background(), &CreateDisputeInput{
		TransactionID: "TXN-REV-001",
		Platform:      "shopee",
		ClaimType:     "defective",
		Amount:        150.00, // within normal range 50-500
	})
	if err != nil {
		t.Fatalf("CreateCase failed: %v", err)
	}

	result, err := ds.AutoDecide(context.Background(), dc.ID)
	if err != nil {
		t.Fatalf("AutoDecide failed: %v", err)
	}

	// baseline 50
	// defective -> +10 -> 60
	// amount 150 (between 50 and 500) -> no change, score still 60
	if result.Decision != "manual_review" {
		t.Errorf("expected decision manual_review for score %.0f, got %s", result.Score, result.Decision)
	}
	if result.Dispute.Status != "manual_review" {
		t.Errorf("expected dispute Status manual_review, got %s", result.Dispute.Status)
	}
}

func TestDisputeService_EvaluateCase_RuleBreakdown(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})
	ds := NewDisputeService(db, testLogger(), nil)

	dc, err := ds.CreateCase(context.Background(), &CreateDisputeInput{
		TransactionID: "TXN-BRK-001",
		Platform:      "shopee",
		ClaimType:     "wrong_item",
		Amount:        30.00,
	})
	if err != nil {
		t.Fatalf("CreateCase failed: %v", err)
	}

	result, err := ds.EvaluateCase(context.Background(), dc.ID)
	if err != nil {
		t.Fatalf("EvaluateCase failed: %v", err)
	}

	if len(result.RuleBreakdown) < 2 {
		t.Errorf("expected at least 2 rule breakdown items, got %d", len(result.RuleBreakdown))
	}

	ruleNames := make(map[string]bool)
	for _, item := range result.RuleBreakdown {
		if item.Rule == "" {
			t.Error("found empty rule name in breakdown")
		}
		ruleNames[item.Rule] = true
	}
	if !ruleNames["claim_type"] {
		t.Error("expected claim_type rule in breakdown")
	}
	if !ruleNames["amount_threshold"] {
		t.Error("expected amount_threshold rule in breakdown")
	}
}

func TestDisputeService_UpdateDisputeStatus(t *testing.T) {
	db := dbtest.NewDB(t, &DisputeCase{})
	ds := NewDisputeService(db, testLogger(), nil)

	dc, err := ds.CreateCase(context.Background(), &CreateDisputeInput{
		TransactionID: "TXN-UPD-001",
		Platform:      "shopee",
		ClaimType:     "defective",
		Amount:        80.00,
	})
	if err != nil {
		t.Fatalf("CreateCase failed: %v", err)
	}

	updated, err := ds.UpdateDisputeStatus(context.Background(), dc.ID, "approved", "manual override")
	if err != nil {
		t.Fatalf("UpdateDisputeStatus failed: %v", err)
	}
	if updated.Status != "approved" {
		t.Errorf("expected Status approved, got %s", updated.Status)
	}
	if updated.AiReason != "manual override" {
		t.Errorf("expected AiReason 'manual override', got %s", updated.AiReason)
	}
}
