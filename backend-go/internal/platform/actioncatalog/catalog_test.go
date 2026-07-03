package actioncatalog

import "testing"

func TestDefault_CoversKnownActions(t *testing.T) {
	c := Default()
	known := []string{
		"stock_alert", "system_health", "dashboard_overview",
		"listing_optimize", "compliance_check", "replenish", "discount_risk_check",
		"price_update", "price_review", "listing_publish", "inventory_change",
		"auto_publish", "auto_reply", "profit_watch",
	}
	for _, name := range known {
		if _, ok := c.Lookup(name); !ok {
			t.Errorf("Default catalog missing action: %s", name)
		}
	}
}

func TestDefault_L1ActionsNoApproval(t *testing.T) {
	c := Default()
	l1 := []string{"stock_alert", "system_health", "dashboard_overview", "auto_reply", "profit_watch"}
	for _, name := range l1 {
		entry, ok := c.Lookup(name)
		if !ok {
			t.Fatalf("missing entry: %s", name)
		}
		if entry.Level != Level1 {
			t.Errorf("%s: expected Level1, got %s", name, entry.Level)
		}
		if entry.RequireApproval {
			t.Errorf("%s: L1 action should not RequireApproval", name)
		}
	}
}

func TestDefault_L2ActionsNoApproval(t *testing.T) {
	c := Default()
	l2 := []string{"listing_optimize", "compliance_check", "replenish", "discount_risk_check"}
	for _, name := range l2 {
		entry, ok := c.Lookup(name)
		if !ok {
			t.Fatalf("missing entry: %s", name)
		}
		if entry.Level != Level2 {
			t.Errorf("%s: expected Level2, got %s", name, entry.Level)
		}
		if entry.RequireApproval {
			t.Errorf("%s: L2 action should not RequireApproval", name)
		}
	}
}

func TestDefault_L3ActionsRequireApproval(t *testing.T) {
	c := Default()
	l3 := []string{"price_update", "price_review", "listing_publish", "inventory_change"}
	for _, name := range l3 {
		entry, ok := c.Lookup(name)
		if !ok {
			t.Fatalf("missing entry: %s", name)
		}
		if entry.Level != Level3 {
			t.Errorf("%s: expected Level3, got %s", name, entry.Level)
		}
		if !entry.RequireApproval {
			t.Errorf("%s: L3 action must RequireApproval", name)
		}
	}
}

func TestDefault_L4Blocked(t *testing.T) {
	c := Default()
	entry, ok := c.Lookup("auto_publish")
	if !ok {
		t.Fatal("missing entry: auto_publish")
	}
	if entry.Level != Level4 {
		t.Errorf("auto_publish: expected Level4, got %s", entry.Level)
	}
	if !entry.AutonomousBlocked {
		t.Error("auto_publish: must be AutonomousBlocked")
	}
}

func TestUnknownActionFailsValidation(t *testing.T) {
	c := Default()
	err := c.ValidateProduction("nonexistent_action", RiskLow, false)
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
	if !IsUnknownAction(err) {
		t.Errorf("expected ErrUnknownAction, got: %v", err)
	}
}

func TestL4BlockedInValidateProduction(t *testing.T) {
	c := Default()
	err := c.ValidateProduction("auto_publish", RiskHigh, true)
	if err == nil {
		t.Fatal("expected error for L4 blocked action")
	}
	if !IsAutonomousBlocked(err) {
		t.Errorf("expected ErrAutonomousBlocked, got: %v", err)
	}
}

func TestL3RequiresApprovalInValidateProduction(t *testing.T) {
	c := Default()
	// Without approval → should fail
	err := c.ValidateProduction("price_update", RiskHigh, false)
	if err == nil {
		t.Fatal("expected error for L3 action without approval")
	}
	if !IsApprovalRequired(err) {
		t.Errorf("expected ErrApprovalRequired, got: %v", err)
	}

	// With approval → should pass
	err = c.ValidateProduction("price_update", RiskHigh, true)
	if err != nil {
		t.Errorf("expected no error for approved L3 action, got: %v", err)
	}
}

func TestL1ValidateProductionPasses(t *testing.T) {
	c := Default()
	err := c.ValidateProduction("stock_alert", RiskLow, false)
	if err != nil {
		t.Errorf("L1 action should pass production validation: %v", err)
	}
}

func TestValidateProduction_PriceReviewRequiresApproval(t *testing.T) {
	c := Default()
	err := c.ValidateProduction("price_review", RiskHigh, false)
	if !IsApprovalRequired(err) {
		t.Errorf("price_review should require approval, got: %v", err)
	}

	err = c.ValidateProduction("price_review", RiskHigh, true)
	if err != nil {
		t.Errorf("price_review with approval should pass: %v", err)
	}
}

func TestNewPanicsOnDuplicate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate entry")
		}
	}()
	New([]Entry{
		{ActionType: "dup", Name: "first"},
		{ActionType: "dup", Name: "second"},
	})
}

func TestHasAction(t *testing.T) {
	c := Default()
	if !c.HasAction("stock_alert") {
		t.Error("expected HasAction true for stock_alert")
	}
	if c.HasAction("non_existent") {
		t.Error("expected HasAction false for non_existent")
	}
}

// ---------------------------------------------------------------------------
// Error helpers
// ---------------------------------------------------------------------------

func IsUnknownAction(err error) bool     { return err != nil && isError(err, ErrUnknownAction) }
func IsAutonomousBlocked(err error) bool { return err != nil && isError(err, ErrAutonomousBlocked) }
func IsApprovalRequired(err error) bool  { return err != nil && isError(err, ErrApprovalRequired) }

func isError(err, target error) bool {
	for e := err; e != nil; e = unwrap(e) {
		if e == target {
			return true
		}
	}
	return false
}

func unwrap(err error) error {
	u, ok := err.(interface{ Unwrap() error })
	if !ok {
		return nil
	}
	return u.Unwrap()
}
