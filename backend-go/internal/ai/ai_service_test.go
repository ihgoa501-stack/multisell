package ai

import (
	"errors"
	"testing"
)

// ── CreateAction ─────────────────────────────────────────────────────────────

func TestService_CreateAction_DefaultStatus(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_st_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "status test",
		ProposedBy: "agent:A6",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	if a.Status != "suggested" {
		t.Errorf("status = %q, want %q", a.Status, "suggested")
	}
	if !a.RequiresApproval {
		t.Error("RequiresApproval should be true by default")
	}
}

// ── ApproveAction ────────────────────────────────────────────────────────────

func TestService_ApproveAction_Normal(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_app_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "approve test",
		RiskLevel: "medium", ProposedBy: "agent:A6",
	})

	uid := int64(100)
	approved, err := svc.ApproveAction(a.ID, "bob", &uid, "")
	if err != nil {
		t.Fatalf("ApproveAction: %v", err)
	}
	if approved.Status != "approved" {
		t.Errorf("status = %q, want %q", approved.Status, "approved")
	}
	if approved.ApprovedBy != "bob" {
		t.Errorf("approved_by = %q, want %q", approved.ApprovedBy, "bob")
	}
	if approved.ApprovedByUserID == nil || *approved.ApprovedByUserID != 100 {
		t.Errorf("approved_by_user_id = %v, want 100", approved.ApprovedByUserID)
	}
	if approved.ApprovedAt == nil {
		t.Error("expected approved_at to be set")
	}
}

func TestService_ApproveAction_FromRejected(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_appr_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "reject then approve",
		RiskLevel: "medium", ProposedBy: "agent:A6",
	})

	// Reject first.
	if _, err := svc.RejectAction(a.ID, "alice", nil, "not needed"); err != nil {
		t.Fatalf("RejectAction: %v", err)
	}

	// Try to approve from rejected — should fail.
	_, err := svc.ApproveAction(a.ID, "bob", nil, "")
	if err == nil {
		t.Fatal("expected error when approving a rejected action")
	}
	var invErr *InvalidTransitionError
	if !errors.As(err, &invErr) {
		t.Fatalf("expected InvalidTransitionError, got %T: %v", err, err)
	}
}

// ── RejectAction ─────────────────────────────────────────────────────────────

func TestService_RejectAction_Normal(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_rej_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "reject test",
		RiskLevel: "medium", ProposedBy: "agent:A6",
	})

	rejected, err := svc.RejectAction(a.ID, "alice", nil, "low confidence")
	if err != nil {
		t.Fatalf("RejectAction: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Errorf("status = %q, want %q", rejected.Status, "rejected")
	}
	if rejected.RejectedBy != "alice" {
		t.Errorf("rejected_by = %q, want %q", rejected.RejectedBy, "alice")
	}
	if rejected.RejectionReason != "low confidence" {
		t.Errorf("rejection_reason = %q, want %q", rejected.RejectionReason, "low confidence")
	}
	if rejected.RejectedAt == nil {
		t.Error("expected rejected_at to be set")
	}
}

func TestService_RejectAction_EmptyReason(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_rej_2", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "empty reason",
		RiskLevel: "medium", ProposedBy: "agent:A6",
	})

	rejected, err := svc.RejectAction(a.ID, "alice", nil, "")
	if err != nil {
		t.Fatalf("RejectAction: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Errorf("status = %q, want %q", rejected.Status, "rejected")
	}
	// Empty reason should still be persisted as empty string.
	if rejected.RejectionReason != "" {
		t.Errorf("expected empty rejection_reason, got %q", rejected.RejectionReason)
	}
}

// ── ExecuteAction ────────────────────────────────────────────────────────────

func TestService_ExecuteAction_FullChain(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	noApproval := false
	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_full_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "full chain",
		RiskLevel: "low", ProposedBy: "agent:A2", RequiresApproval: &noApproval,
	})

	executed, err := svc.ExecuteAction(a.ID, nil, "alice", "")
	if err != nil {
		t.Fatalf("ExecuteAction: %v", err)
	}
	if executed.Status != "executed" {
		t.Errorf("status = %q, want %q", executed.Status, "executed")
	}
	if executed.ExecutedAt == nil {
		t.Error("expected executed_at to be set")
	}
}

func TestService_ExecuteAction_NotApproved(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_exna_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "no approval",
		RiskLevel: "high", ProposedBy: "agent:A6",
	})
	// RequiresApproval defaults to true.

	_, err := svc.ExecuteAction(a.ID, nil, "alice", "")
	if err == nil {
		t.Fatal("expected error for unapproved action")
	}
	if !errors.Is(err, ErrApprovalRequired) {
		t.Errorf("expected ErrApprovalRequired, got %v", err)
	}
}

func TestService_ExecuteAction_OperatorAndUserID(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	noApproval := false
	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_exuid_2", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "exec operator",
		RiskLevel: "low", ProposedBy: "agent:A2", RequiresApproval: &noApproval,
	})

	uid := int64(42)
	executed, err := svc.ExecuteAction(a.ID, &uid, "system", "")
	if err != nil {
		t.Fatalf("ExecuteAction: %v", err)
	}
	if executed.ExecutedBy != "system" {
		t.Errorf("executed_by = %q, want %q", executed.ExecutedBy, "system")
	}
	if executed.ExecutedByUserID == nil || *executed.ExecutedByUserID != 42 {
		t.Errorf("executed_by_user_id = %v, want 42", executed.ExecutedByUserID)
	}
}

// ── FailAction ───────────────────────────────────────────────────────────────

func TestService_FailAction(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_fail_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "fail test",
		RiskLevel: "medium", ProposedBy: "agent:A6",
	})

	// Manually set to "executing" since ExecuteAction cannot leave it there
	// (no command dispatcher configured — it atomically goes to executed/failed).
	db.Model(&UnifiedAction{}).Where("id = ?", a.ID).Update("status", "executing")

	failed, err := svc.FailAction(a.ID, "execution timeout")
	if err != nil {
		t.Fatalf("FailAction: %v", err)
	}
	if failed.Status != "failed" {
		t.Errorf("status = %q, want %q", failed.Status, "failed")
	}
	if failed.RejectionReason != "execution timeout" {
		t.Errorf("rejection_reason = %q, want %q", failed.RejectionReason, "execution timeout")
	}
	if failed.FailedAt == nil {
		t.Error("expected failed_at to be set")
	}
}

// ── ReviewAction ─────────────────────────────────────────────────────────────

func TestService_ReviewAction(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	noApproval := false
	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_rev_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "review test",
		RiskLevel: "low", ProposedBy: "agent:A2", RequiresApproval: &noApproval,
	})

	// Execute first — review only allows from "executed" or "failed".
	if _, err := svc.ExecuteAction(a.ID, nil, "alice", ""); err != nil {
		t.Fatalf("ExecuteAction: %v", err)
	}

	reviewed, err := svc.ReviewAction(a.ID)
	if err != nil {
		t.Fatalf("ReviewAction: %v", err)
	}
	if reviewed.Status != "reviewed" {
		t.Errorf("status = %q, want %q", reviewed.Status, "reviewed")
	}
	if reviewed.ReviewedAt == nil {
		t.Error("expected reviewed_at to be set")
	}
}

// ── InvalidTransitionError ───────────────────────────────────────────────────

func TestService_InvalidTransitionError(t *testing.T) {
	// Test the error message format.
	transitionErr := &InvalidTransitionError{From: "approved", To: "rejected"}
	want := "invalid action transition: approved → rejected"
	if transitionErr.Error() != want {
		t.Errorf("Error() = %q, want %q", transitionErr.Error(), want)
	}

	// Verify it's returned at runtime by attempting an illegal transition.
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_inv_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "invalid transition",
		RiskLevel: "medium", ProposedBy: "agent:A6",
	})

	// Reject first (allowed from "suggested").
	if _, err := svc.RejectAction(a.ID, "alice", nil, "no"); err != nil {
		t.Fatalf("RejectAction: %v", err)
	}

	// Reject again (now in "rejected", not allowed).
	_, rejectErr := svc.RejectAction(a.ID, "bob", nil, "still no")
	if rejectErr == nil {
		t.Fatal("expected error for double reject")
	}
	var invErr *InvalidTransitionError
	if !errors.As(rejectErr, &invErr) {
		t.Fatalf("expected InvalidTransitionError, got %T: %v", rejectErr, rejectErr)
	}
	if invErr.From != "rejected" {
		t.Errorf("From = %q, want %q", invErr.From, "rejected")
	}
	if invErr.To != "rejected" {
		t.Errorf("To = %q, want %q", invErr.To, "rejected")
	}
}

// ── riskLevelToInt ───────────────────────────────────────────────────────────

func TestRiskLevelToInt(t *testing.T) {
	low := riskLevelToInt("low")
	medium := riskLevelToInt("medium")
	high := riskLevelToInt("high")

	if !(low < medium && medium < high) {
		t.Errorf("risk levels not ordered: low=%d, medium=%d, high=%d", low, medium, high)
	}
	// Default (empty or unknown) should equal medium.
	if riskLevelToInt("") != medium {
		t.Error("default risk level should equal medium")
	}
	if riskLevelToInt("unknown") != medium {
		t.Error("unknown risk level should default to medium")
	}
}
