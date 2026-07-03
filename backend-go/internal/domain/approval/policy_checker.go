package approval

import (
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
