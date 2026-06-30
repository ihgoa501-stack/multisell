package consolidation

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// ---------------------------------------------------------------------------
// CalculateDiscountPct (pure function tests)
// ---------------------------------------------------------------------------

func TestCalculateDiscountPct_Below50(t *testing.T) {
	t.Parallel()

	pct := CalculateDiscountPct(30)
	if pct != 0 {
		t.Errorf("expected 0%% for 30kg, got %.2f%%", pct)
	}
}

func TestCalculateDiscountPct_Above50(t *testing.T) {
	t.Parallel()

	pct := CalculateDiscountPct(50)
	if pct != 5.0 {
		t.Errorf("expected 5%% for 50kg, got %.2f%%", pct)
	}
}

func TestCalculateDiscountPct_Above100(t *testing.T) {
	t.Parallel()

	pct := CalculateDiscountPct(100)
	if pct != 10.0 {
		t.Errorf("expected 10%% for 100kg, got %.2f%%", pct)
	}
}

func TestCalculateDiscountPct_Above500(t *testing.T) {
	t.Parallel()

	pct := CalculateDiscountPct(500)
	if pct != 20.0 {
		t.Errorf("expected 20%% for 500kg, got %.2f%%", pct)
	}
}

func TestCalculateDiscountPct_MidRangeWeight(t *testing.T) {
	t.Parallel()

	if pct := CalculateDiscountPct(75); pct != 5.0 {
		t.Errorf("expected 5%% for 75kg, got %.2f%%", pct)
	}
	if pct := CalculateDiscountPct(250); pct != 10.0 {
		t.Errorf("expected 10%% for 250kg, got %.2f%%", pct)
	}
	if pct := CalculateDiscountPct(1000); pct != 20.0 {
		t.Errorf("expected 20%% for 1000kg, got %.2f%%", pct)
	}
}

func TestDiscountLabel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		weight float64
		want   string
	}{
		{10, "0% (<50kg)"},
		{50, "5% (>=50kg)"},
		{100, "10% (>=100kg)"},
		{500, "20% (>=500kg)"},
		{999, "20% (>=500kg)"},
	}

	for _, tc := range cases {
		got := DiscountLabel(tc.weight)
		if got != tc.want {
			t.Errorf("DiscountLabel(%.0f) = %q, want %q", tc.weight, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// CreateGroup
// ---------------------------------------------------------------------------

func TestCreateGroup_Success(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, err := svc.CreateGroup("RU", 72)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	if group.ID == 0 {
		t.Error("expected non-zero group ID")
	}
	if group.Status != GroupStatusOpen {
		t.Errorf("expected status open, got %s", group.Status)
	}
	if group.Destination != "RU" {
		t.Errorf("expected destination RU, got %s", group.Destination)
	}
}

func TestCreateGroup_EmptyDestination(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	_, err := svc.CreateGroup("", 72)
	if err == nil {
		t.Fatal("expected error for empty destination")
	}
}

func TestCreateGroup_DefaultTimeWindow(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, err := svc.CreateGroup("KZ", 0) // 0 → default
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	if group.ID == 0 {
		t.Error("expected non-zero group ID")
	}
}

// ---------------------------------------------------------------------------
// AddItem
// ---------------------------------------------------------------------------

func TestAddItem_Success(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)

	item, err := svc.AddItem(group.ID, 1001, 10.5, 0.05, "RU")
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	if item.ID == 0 {
		t.Error("expected non-zero item ID")
	}
	if item.GroupID != group.ID {
		t.Errorf("expected group_id %d, got %d", group.ID, item.GroupID)
	}
	if item.Status != ItemStatusPending {
		t.Errorf("expected status pending, got %s", item.Status)
	}

	// Group totals should have been updated.
	updated, _ := svc.GetGroup(group.ID)
	if updated.TotalWeightKg != 10.5 {
		t.Errorf("expected total weight 10.5, got %.2f", updated.TotalWeightKg)
	}
}

func TestAddItem_ZeroWeight(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)

	_, err := svc.AddItem(group.ID, 1001, 0, 0.05, "RU")
	if err == nil {
		t.Fatal("expected error for zero weight")
	}
}

func TestAddItem_EmptyDestination(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)

	_, err := svc.AddItem(group.ID, 1001, 10, 0, "")
	if err == nil {
		t.Fatal("expected error for empty destination")
	}
}

func TestAddItem_DestinationMismatch(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)

	_, err := svc.AddItem(group.ID, 1001, 10, 0, "KZ")
	if err == nil {
		t.Fatal("expected error for destination mismatch")
	}
}

func TestAddItem_GroupNotFound(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	_, err := svc.AddItem(999, 1001, 10, 0, "RU")
	if err == nil {
		t.Fatal("expected error for non-existent group")
	}
}

// ---------------------------------------------------------------------------
// RemoveItem
// ---------------------------------------------------------------------------

func TestRemoveItem_Success(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)
	item, _ := svc.AddItem(group.ID, 1001, 10, 0, "RU")

	if err := svc.RemoveItem(item.ID); err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}

	// Verify item status.
	items, _ := svc.GetItemsByGroup(group.ID)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Status != ItemStatusRemoved {
		t.Errorf("expected status removed, got %s", items[0].Status)
	}

	// Verify group totals updated (excludes removed items).
	updated, _ := svc.GetGroup(group.ID)
	if updated.TotalWeightKg != 0 {
		t.Errorf("expected total weight 0 after removing only item, got %.2f", updated.TotalWeightKg)
	}
}

func TestRemoveItem_AlreadyRemoved(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)
	item, _ := svc.AddItem(group.ID, 1001, 10, 0, "RU")

	_ = svc.RemoveItem(item.ID)
	err := svc.RemoveItem(item.ID)
	if err == nil {
		t.Fatal("expected error for already removed item")
	}
}

func TestRemoveItem_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	err := svc.RemoveItem(999)
	if err == nil {
		t.Fatal("expected error for non-existent item")
	}
}

// ---------------------------------------------------------------------------
// Negotiate
// ---------------------------------------------------------------------------

func TestNegotiate_NoDiscountBelow50kg(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)
	_, _ = svc.AddItem(group.ID, 1001, 30, 0, "RU")

	result, err := svc.Negotiate(group.ID)
	if err != nil {
		t.Fatalf("Negotiate failed: %v", err)
	}

	if result.DiscountRate != 0 {
		t.Errorf("expected 0%% discount for 30kg, got %.2f%%", result.DiscountRate)
	}
	if result.Status != GroupStatusNegotiating {
		t.Errorf("expected status negotiating, got %s", result.Status)
	}
}

func TestNegotiate_5PercentDiscount(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)
	_, _ = svc.AddItem(group.ID, 1001, 30, 0, "RU")
	_, _ = svc.AddItem(group.ID, 1002, 25, 0, "RU")

	result, err := svc.Negotiate(group.ID)
	if err != nil {
		t.Fatalf("Negotiate failed: %v", err)
	}

	if result.DiscountRate != 5.0 {
		t.Errorf("expected 5%% discount for 55kg, got %.2f%%", result.DiscountRate)
	}
	if result.TotalWeightKg != 55 {
		t.Errorf("expected total weight 55, got %.2f", result.TotalWeightKg)
	}
}

func TestNegotiate_10PercentDiscount(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)
	_, _ = svc.AddItem(group.ID, 1001, 100, 0, "RU")

	result, err := svc.Negotiate(group.ID)
	if err != nil {
		t.Fatalf("Negotiate failed: %v", err)
	}

	if result.DiscountRate != 10.0 {
		t.Errorf("expected 10%% discount for 100kg, got %.2f%%", result.DiscountRate)
	}
}

func TestNegotiate_20PercentDiscount(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)
	_, _ = svc.AddItem(group.ID, 1001, 500, 0, "RU")

	result, err := svc.Negotiate(group.ID)
	if err != nil {
		t.Fatalf("Negotiate failed: %v", err)
	}

	if result.DiscountRate != 20.0 {
		t.Errorf("expected 20%% discount for 500kg, got %.2f%%", result.DiscountRate)
	}
}

func TestNegotiate_GroupNotOpen(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)
	_, _ = svc.AddItem(group.ID, 1001, 100, 0, "RU")

	// First negotiate transitions to "negotiating".
	_, err := svc.Negotiate(group.ID)
	if err != nil {
		t.Fatalf("first Negotiate failed: %v", err)
	}

	// Second negotiate should work (negotiating → negotiating is allowed).
	_, err = svc.Negotiate(group.ID)
	if err != nil {
		t.Fatalf("second Negotiate should succeed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetGroup / ListGroups / GetItemsByGroup
// ---------------------------------------------------------------------------

func TestGetGroup_Success(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	created, _ := svc.CreateGroup("RU", 72)

	fetched, err := svc.GetGroup(created.ID)
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}

	if fetched.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, fetched.ID)
	}
	if fetched.Destination != "RU" {
		t.Errorf("expected destination RU, got %s", fetched.Destination)
	}
}

func TestGetGroup_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	_, err := svc.GetGroup(999)
	if err == nil {
		t.Fatal("expected error for non-existent group")
	}
}

func TestListGroups(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	// No groups yet.
	groups, err := svc.ListGroups()
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}

	// Create two groups.
	_, _ = svc.CreateGroup("RU", 72)
	_, _ = svc.CreateGroup("KZ", 72)

	groups, err = svc.ListGroups()
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestGetItemsByGroup(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)

	// No items yet.
	items, err := svc.GetItemsByGroup(group.ID)
	if err != nil {
		t.Fatalf("GetItemsByGroup failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}

	// Add two items.
	_, _ = svc.AddItem(group.ID, 1001, 10, 0, "RU")
	_, _ = svc.AddItem(group.ID, 1002, 20, 0, "RU")

	items, err = svc.GetItemsByGroup(group.ID)
	if err != nil {
		t.Fatalf("GetItemsByGroup failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

// ---------------------------------------------------------------------------
// Integration: full flow
// ---------------------------------------------------------------------------

func TestConsolidation_FullFlow(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	// 1. Create group for Russia.
	group, err := svc.CreateGroup("RU", 72)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	// 2. Add items totaling 150kg across multiple SKUs.
	_, _ = svc.AddItem(group.ID, 1001, 50, 0, "RU")
	_, _ = svc.AddItem(group.ID, 1002, 50, 0, "RU")
	_, _ = svc.AddItem(group.ID, 1003, 50, 0, "RU")

	// 3. Verify group totals.
	group, _ = svc.GetGroup(group.ID)
	if group.TotalWeightKg != 150 {
		t.Errorf("expected total weight 150, got %.2f", group.TotalWeightKg)
	}

	// 4. Remove one item before negotiating.
	items, _ := svc.GetItemsByGroup(group.ID)
	if err := svc.RemoveItem(items[0].ID); err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}

	// 5. Verify weight adjusted.
	group, _ = svc.GetGroup(group.ID)
	if group.TotalWeightKg != 100 {
		t.Errorf("expected total weight 100 after remove, got %.2f", group.TotalWeightKg)
	}

	// 6. Negotiate -- should get 10% discount (100kg >= tier 2).
	result, err := svc.Negotiate(group.ID)
	if err != nil {
		t.Fatalf("Negotiate failed: %v", err)
	}
	if result.DiscountRate != 10.0 {
		t.Errorf("expected 10%% discount for 100kg, got %.2f%%", result.DiscountRate)
	}
}

func TestConsolidation_RemoveAfterNegotiateFails(t *testing.T) {
	db := dbtest.NewDB(t, &ConsolidationGroup{}, &ConsolidationItem{})
	svc := NewConsolidationService(db, dbtest.NewLogger(t))

	group, _ := svc.CreateGroup("RU", 72)
	item, _ := svc.AddItem(group.ID, 1001, 50, 0, "RU")

	// Negotiate transitions to "negotiating".
	_, _ = svc.Negotiate(group.ID)

	// RemoveItem should fail because group is not "open".
	err := svc.RemoveItem(item.ID)
	if err == nil {
		t.Fatal("expected error when removing item from non-open group")
	}
}
