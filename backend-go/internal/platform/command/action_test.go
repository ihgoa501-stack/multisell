package command

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// ---------------------------------------------------------------------------
// AgentAction creation — read-only/suggestion actions can exist without
// execution. This test proves that constructing an AgentAction is not itself
// an execution.
// ---------------------------------------------------------------------------

func TestAgentActionCreation_NoExecution(t *testing.T) {
	// An action can be created with dry_run mode and never dispatched.
	// This proves the action struct is a data envelope, not an execution.
	action := AgentAction{
		ActionType:       "stock_alert",
		Version:          "1.0.0",
		AgentID:          "A5",
		Actor:            "system",
		RiskLevel:        RiskLow,
		ApprovalRequired: false,
		Mode:             ModeDryRun,
		TargetType:       "sku",
		TargetID:         "SKU001",
		Input:            map[string]interface{}{"stock_status": "low"},
	}
	if action.ActionType != "stock_alert" {
		t.Errorf("expected stock_alert, got %s", action.ActionType)
	}
	if action.Mode != ModeDryRun {
		t.Errorf("expected dry_run mode, got %v", action.Mode)
	}
}

// ---------------------------------------------------------------------------
// DispatchSafe — dry_run validates but does not execute.
// ---------------------------------------------------------------------------

func TestDispatchSafe_DryRun_NoExecution(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	var executed bool
	d.Register("stock_alert", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		executed = true
		return &Result{Success: true}, nil
	})

	action := AgentAction{
		ActionType: "stock_alert",
		Mode:       ModeDryRun,
		RiskLevel:  RiskLow,
	}
	result, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil {
		t.Fatalf("DispatchSafe dry_run returned error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true for dry run")
	}
	if executed {
		t.Error("handler was executed despite dry_run mode")
	}
}

func TestDispatchSafe_DryRun_UnregisteredHandler(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	action := AgentAction{
		ActionType: "nonexistent",
		Mode:       ModeDryRun,
	}
	_, err := d.DispatchSafe(context.Background(), action, nil)
	if !IsHandlerNotFound(err) {
		t.Errorf("expected HandlerNotFoundError for unregistered action, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DispatchSafe — production high-risk actions require approval.
// ---------------------------------------------------------------------------

func TestDispatchSafe_Production_HighRisk_RequiresApproval(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	d.Register("price_update", okHandler)

	// High-risk action without approval — should be blocked.
	action := AgentAction{
		ActionType:       "price_update",
		Mode:             ModeProduction,
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		ApprovalID:       nil,
		Input:            map[string]interface{}{"sku_code": "SKU001", "suggested_price": float64(29.99)},
	}
	_, err := d.DispatchSafe(context.Background(), action, nil)
	if err != ErrApprovalRequired {
		t.Errorf("expected ErrApprovalRequired for high-risk action without approval, got %v", err)
	}
}

func TestDispatchSafe_Production_HighRisk_Approved(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	d.Register("price_update", okHandler)

	approvalID := int64(42)
	action := AgentAction{
		ActionType:       "price_update",
		Mode:             ModeProduction,
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		ApprovalID:       &approvalID,
	}
	mockPolicy := &mockPolicyChecker{approved: true}
	result, err := d.DispatchSafe(context.Background(), action, mockPolicy)
	if err != nil {
		t.Fatalf("DispatchSafe with valid approval returned error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true for approved high-risk action")
	}
}

func TestDispatchSafe_Production_ApprovalNotInPolicy(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	d.Register("price_update", okHandler)

	approvalID := int64(99)
	action := AgentAction{
		ActionType:       "price_update",
		Mode:             ModeProduction,
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		ApprovalID:       &approvalID,
	}
	mockPolicy := &mockPolicyChecker{approved: false}
	_, err := d.DispatchSafe(context.Background(), action, mockPolicy)
	if err != ErrApprovalRequired {
		t.Errorf("expected ErrApprovalRequired when policy rejects approval, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DispatchSafe — sandbox mode allows execution without approval.
// ---------------------------------------------------------------------------

func TestDispatchSafe_Sandbox_HighRisk_NoApprovalNeeded(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	var executed bool
	d.Register("price_update", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		executed = true
		return &Result{Success: true}, nil
	})

	action := AgentAction{
		ActionType:       "price_update",
		Mode:             ModeSandbox,
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
	}
	result, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil {
		t.Fatalf("DispatchSafe sandbox returned error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true for sandbox execution")
	}
	if !executed {
		t.Error("handler was not executed in sandbox mode")
	}
}

// ---------------------------------------------------------------------------
// DispatchSafe — low-risk production actions do not require approval.
// ---------------------------------------------------------------------------

func TestDispatchSafe_Production_LowRisk_NoApproval(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)

	var executed bool
	d.Register("stock_alert", func(_ context.Context, _ map[string]interface{}) (*Result, error) {
		executed = true
		return &Result{Success: true}, nil
	})

	action := AgentAction{
		ActionType: "stock_alert",
		Mode:       ModeProduction,
		RiskLevel:  RiskLow,
	}
	result, err := d.DispatchSafe(context.Background(), action, nil)
	if err != nil {
		t.Fatalf("DispatchSafe low-risk returned error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if !executed {
		t.Error("low-risk action should execute without approval")
	}
}

// ---------------------------------------------------------------------------
// Audit context test — AgentAction.AuditContext produces structured data.
// ---------------------------------------------------------------------------

func TestAgentAction_AuditContext(t *testing.T) {
	action := AgentAction{
		ActionType:       "price_update",
		Version:          "1.0.0",
		AgentID:          "A6",
		Actor:            "system",
		TargetType:       "sku",
		TargetID:         "SKU001",
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		Mode:             ModeProduction,
	}
	ctx := action.AuditContext()
	if ctx["action_type"] != "price_update" {
		t.Errorf("expected price_update, got %v", ctx["action_type"])
	}
	if ctx["agent_id"] != "A6" {
		t.Errorf("expected A6, got %v", ctx["agent_id"])
	}
	if ctx["risk_level"] != "high" {
		t.Errorf("expected high, got %v", ctx["risk_level"])
	}
	if ctx["mode"] != "production" {
		t.Errorf("expected production, got %v", ctx["mode"])
	}
	if ctx["approval_required"] != true {
		t.Error("expected approval_required=true")
	}
}

// ---------------------------------------------------------------------------
// RiskLevel parsing.
// ---------------------------------------------------------------------------

func TestParseRiskLevel(t *testing.T) {
	tests := []struct {
		input string
		want  RiskLevel
	}{
		{"none", RiskNone},
		{"low", RiskLow},
		{"medium", RiskMedium},
		{"high", RiskHigh},
		{"critical", RiskHigh},
		{"unknown", RiskLow},
		{"", RiskLow},
	}
	for _, tc := range tests {
		got := ParseRiskLevel(tc.input)
		if got != tc.want {
			t.Errorf("ParseRiskLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Mock policy checker.
// ---------------------------------------------------------------------------

type mockPolicyChecker struct {
	approved bool
}

func (m *mockPolicyChecker) IsApproved(_ int64) bool {
	return m.approved
}
