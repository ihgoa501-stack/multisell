package approval

import (
	"context"
	"time"
)

// ApprovalPolicyChecker implements command.PolicyChecker by querying the
// approval_request table. It verifies that an approval ID exists, is still
// valid (status = "approved"), and has not expired.
//
// It satisfies command.PolicyChecker structurally (IsApproved(int64) bool)
// without importing command to avoid an import cycle (command → approval → command).
type ApprovalPolicyChecker struct {
	svc *Service
}

// IsApprovedFor binds an approval to the exact executable action and target.
// Unknown mappings and unparseable constrained targets fail closed.
func (c *ApprovalPolicyChecker) AuthorizeFor(ctx context.Context, approvalID int64, actionType, targetType, targetID, key string) error {
	return c.svc.AuthorizeExecution(ctx, approvalID, actionType, targetType, targetID, key)
}

func (c *ApprovalPolicyChecker) ConsumeFor(ctx context.Context, approvalID int64, actionType, targetType, targetID, key string) error {
	return c.svc.ConsumeExecution(ctx, approvalID, actionType, targetType, targetID, key)
}

func (c *ApprovalPolicyChecker) CompleteFor(ctx context.Context, approvalID int64, key string) error {
	return c.svc.CompleteExecution(ctx, approvalID, key)
}

func (c *ApprovalPolicyChecker) FailFor(ctx context.Context, approvalID int64, key string, cause error) error {
	return c.svc.FailExecution(ctx, approvalID, key, cause)
}

// NewApprovalPolicyChecker creates a PolicyChecker backed by the approval
// domain service.
func NewApprovalPolicyChecker(svc *Service) *ApprovalPolicyChecker {
	return &ApprovalPolicyChecker{svc: svc}
}

// IsApproved returns true if the approval_id exists in the approval_request
// table with status "approved" and has not expired.
func (c *ApprovalPolicyChecker) IsApproved(approvalID int64) bool {
	req, err := c.svc.Get(approvalID)
	if err != nil {
		return false
	}
	// Approval must be explicitly approved, not pending, rejected, or otherwise.
	if req.Status != StatusApproved {
		return false
	}
	// Check expiration.
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}
