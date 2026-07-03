package command

import (
	"context"
	"errors"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/actioncatalog"
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
		AgentID:    "A5",
		Actor:      "system",
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
		AgentID:    "A5",
		Actor:      "system",
		Mode:       ModeDryRun,
		RiskLevel:  RiskLow,
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
		AgentID:          "A5",
		Actor:            "system",
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
		AgentID:          "A5",
		Actor:            "system",
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
		AgentID:          "A5",
		Actor:            "system",
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
		AgentID:          "A5",
		Actor:            "system",
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
		AgentID:    "A5",
		Actor:      "system",
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
// AgentAction.Validate — required identity, mode, risk checks enforcement.
// ---------------------------------------------------------------------------

func TestAgentActionValidate_MissingActionType(t *testing.T) {
	action := AgentAction{
		AgentID:    "A5",
		Actor:      "system",
		RiskLevel:  RiskLow,
		Mode:       ModeDryRun,
	}
	err := action.Validate()
	if err == nil {
		t.Fatal("expected error for missing action_type")
	}
	if !errors.Is(err, ErrActionValidation) {
		t.Errorf("expected ErrActionValidation, got %v", err)
	}
}

func TestAgentActionValidate_MissingAgentID(t *testing.T) {
	action := AgentAction{
		ActionType: "stock_alert",
		Actor:      "system",
		RiskLevel:  RiskLow,
		Mode:       ModeDryRun,
	}
	err := action.Validate()
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
}

func TestAgentActionValidate_MissingActor(t *testing.T) {
	action := AgentAction{
		ActionType: "stock_alert",
		AgentID:    "A5",
		RiskLevel:  RiskLow,
		Mode:       ModeDryRun,
	}
	err := action.Validate()
	if err == nil {
		t.Fatal("expected error for missing actor")
	}
}

func TestAgentActionValidate_InvalidMode(t *testing.T) {
	action := AgentAction{
		ActionType: "stock_alert",
		AgentID:    "A5",
		Actor:      "system",
		RiskLevel:  RiskLow,
		Mode:       ActionMode(99),
	}
	err := action.Validate()
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestAgentActionValidate_InvalidRiskLevel(t *testing.T) {
	action := AgentAction{
		ActionType: "stock_alert",
		AgentID:    "A5",
		Actor:      "system",
		RiskLevel:  RiskLevel(99),
		Mode:       ModeDryRun,
	}
	err := action.Validate()
	if err == nil {
		t.Fatal("expected error for invalid risk_level")
	}
}

// ---------------------------------------------------------------------------
// TestAgentActionValidate_HighRiskDefaultApprovalRequired — high risk action
// without approval_required=true must be rejected.
// ---------------------------------------------------------------------------

func TestAgentActionValidate_HighRiskDefaultApprovalRequired(t *testing.T) {
	action := AgentAction{
		ActionType:       "price_update",
		AgentID:          "A5",
		Actor:            "system",
		RiskLevel:        RiskHigh,
		ApprovalRequired: false, // violation — high risk must set approval_required=true
		Mode:             ModeProduction,
	}
	err := action.Validate()
	if err == nil {
		t.Fatal("expected error: high risk action must set approval_required=true")
	}
}

func TestAgentActionValidate_HighRiskWithApprovalRequiredPasses(t *testing.T) {
	approvalID := int64(42)
	action := AgentAction{
		ActionType:       "price_update",
		AgentID:          "A5",
		Actor:            "system",
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		ApprovalID:       &approvalID,
		Mode:             ModeProduction,
		Input:            map[string]interface{}{"price": 29.99},
	}
	err := action.Validate()
	if err != nil {
		t.Fatalf("expected no error for valid high-risk action, got: %v", err)
	}
}

func TestAgentActionValidate_DryRunWithApprovalID(t *testing.T) {
	approvalID := int64(1)
	action := AgentAction{
		ActionType:       "price_update",
		AgentID:          "A5",
		Actor:            "system",
		RiskLevel:        RiskHigh,
		ApprovalRequired: true,
		ApprovalID:       &approvalID,
		Mode:             ModeDryRun, // dry_run must not carry approval
	}
	err := action.Validate()
	if err == nil {
		t.Fatal("expected error: dry_run must not carry approval_id")
	}
}

func TestAgentActionValidate_ValidLowRisk(t *testing.T) {
	action := AgentAction{
		ActionType: "stock_alert",
		AgentID:    "A5",
		Actor:      "system",
		RiskLevel:  RiskLow,
		Mode:       ModeProduction,
	}
	err := action.Validate()
	if err != nil {
		t.Fatalf("expected no error for valid low-risk action, got: %v", err)
	}
}

func TestAgentActionValidate_ValidDryRun(t *testing.T) {
	action := AgentAction{
		ActionType: "stock_alert",
		AgentID:    "A5",
		Actor:      "system",
		RiskLevel:  RiskLow,
		Mode:       ModeDryRun,
	}
	err := action.Validate()
	if err != nil {
		t.Fatalf("expected no error for valid dry_run action, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ActionStatus.String — all statuses produce correct labels.
// ---------------------------------------------------------------------------

func TestActionStatus_String(t *testing.T) {
	tests := []struct {
		status ActionStatus
		want   string
	}{
		{StatusSuggested, "suggested"},
		{StatusPendingApproval, "pending_approval"},
		{StatusApproved, "approved"},
		{StatusRejected, "rejected"},
		{StatusExecuting, "executing"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusBlocked, "blocked"},
	}
	for _, tc := range tests {
		if got := tc.status.String(); got != tc.want {
			t.Errorf("ActionStatus(%d).String() = %q, want %q", tc.status, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// HighRiskActions returns the canonical list.
// ---------------------------------------------------------------------------

func TestHighRiskActions(t *testing.T) {
	actions := HighRiskActions()
	// Now derived from actioncatalog.DefaultEntries() — the catalog is the
	// source of truth, so verify all catalog RiskHigh entries are present.
	catalogEntries := actioncatalog.DefaultEntries()
	for _, e := range catalogEntries {
		if e.RiskLevel >= actioncatalog.RiskHigh {
			found := false
			for _, a := range actions {
				if a == e.ActionType {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("HighRiskActions() missing catalog entry: %s", e.ActionType)
			}
		}
	}
	// Also verify no stale entries (actions not in catalog).
	catalogTypes := make(map[string]bool, len(catalogEntries))
	for _, e := range catalogEntries {
		catalogTypes[e.ActionType] = true
	}
	for _, a := range actions {
		if !catalogTypes[a] {
			t.Errorf("HighRiskActions() has stale entry not in catalog: %s", a)
		}
	}
}

// ---------------------------------------------------------------------------
// Idempotency key — record as TODO. The current system does not have an
// idempotency store, so duplicate execution is not yet prevented at the
// platform layer. This test documents the known gap.
// ---------------------------------------------------------------------------

func TestIdempotencyKey_NotYetImplemented(t *testing.T) {
	// ponytail: idempotency storage is not implemented yet. The AgentAction
	// carries the idempotency_key field so callers can generate a key, but
	// duplicate detection requires a persisted store (e.g. Redis or DB table).
	// Add when dedup is a measurable requirement.
	t.Skip("TODO: idempotency dedup not yet implemented — requires persisted store")
}

// ---------------------------------------------------------------------------
// DispatchSafe — structural validation integrated with dispatch.
// ---------------------------------------------------------------------------

func TestDispatchSafe_MissingAgentID_Rejected(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)
	d.Register("stock_alert", okHandler)

	action := AgentAction{
		ActionType: "stock_alert",
		Actor:      "system",
		RiskLevel:  RiskLow,
		Mode:       ModeProduction,
		// missing AgentID
	}
	_, err := d.DispatchSafe(context.Background(), action, nil)
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
}

func TestDispatchSafe_MissingActor_Rejected(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)
	d.Register("stock_alert", okHandler)

	action := AgentAction{
		ActionType: "stock_alert",
		AgentID:    "A5",
		RiskLevel:  RiskLow,
		Mode:       ModeProduction,
		// missing Actor
	}
	_, err := d.DispatchSafe(context.Background(), action, nil)
	if err == nil {
		t.Fatal("expected error for missing actor")
	}
}

// ---------------------------------------------------------------------------
// DispatchSafe — high-risk production action without approval is blocked
// (also tested via Validate path).
// ---------------------------------------------------------------------------

func TestDispatchSafe_Production_HighRisk_DefaultApprovalRequired(t *testing.T) {
	logger := dbtest.NewLogger(t)
	d := NewDispatcher(logger)
	d.Register("order_cancel", okHandler)

	action := AgentAction{
		ActionType:       "order_cancel",
		AgentID:          "A5",
		Actor:            "system",
		RiskLevel:        RiskHigh,
		ApprovalRequired: false, // violation — must be true for high risk
		Mode:             ModeProduction,
	}
	_, err := d.DispatchSafe(context.Background(), action, nil)
	if err == nil {
		t.Fatal("expected error: high risk action without approval_required should be rejected")
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
