package sourcing1688

import "time"

const (
	LifecycleUnverifiedLead  = "unverified_lead"
	LifecycleNeedsReview     = "needs_review"
	LifecycleCaptureFailed   = "capture_failed"
	LifecyclePendingReview   = "pending_review"
	LifecycleRejected        = "rejected"
	LifecycleReadyForProduct = "ready_for_product"
	LifecycleEditing         = "editing"
	LifecyclePendingApproval = "pending_approval"
	LifecycleApprovedDraft   = "approved_draft"
	LifecycleArchived        = "archived"
)

const (
	DraftApprovalRequestType = "sourcing_1688_draft"
	DraftApprovalTargetType  = "sourcing_listing_draft"
)

// LifecycleState is the Owner-facing state of one controlled 1688 item.
// Approval never means published: approved_draft remains an internal draft.
type LifecycleState struct {
	SourcingProductID int64      `json:"sourcing_product_id"`
	TaskLinkID        int64      `json:"task_link_id,omitempty"`
	Status            string     `json:"status"`
	ActorID           *int64     `json:"actor_id,omitempty"`
	Reason            string     `json:"reason,omitempty"`
	ApprovalID        *int64     `json:"approval_id,omitempty"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

type CaptureFailureInput struct {
	ActorID int64  `json:"actor_id"`
	Reason  string `json:"reason" binding:"required"`
}

type SourceReviewDecisionInput struct {
	OwnerID int64  `json:"owner_id"`
	Action  string `json:"action" binding:"required"` // approve or reject
	Notes   string `json:"notes" binding:"required"`
}

type DraftApprovalSubmissionInput struct {
	RequesterID int64  `json:"requester_id"`
	Reason      string `json:"reason" binding:"required"`
}

type DraftApprovalDecisionInput struct {
	OwnerID int64  `json:"owner_id"`
	Action  string `json:"action" binding:"required"` // approve or reject
	Note    string `json:"note" binding:"required"`
}

type DraftApprovalResult struct {
	Lifecycle      LifecycleState `json:"lifecycle"`
	DraftID        int64          `json:"draft_id"`
	ListingID      int64          `json:"listing_id"`
	ApprovalID     int64          `json:"approval_id"`
	ApprovalStatus string         `json:"approval_status"`
}

type sourcingLifecycleRow struct {
	ID                 int64      `gorm:"column:id;primaryKey"`
	LifecycleStatus    string     `gorm:"column:lifecycle_status"`
	LifecycleActorID   *int64     `gorm:"column:lifecycle_actor_id"`
	LifecycleReason    string     `gorm:"column:lifecycle_reason"`
	LifecycleUpdatedAt *time.Time `gorm:"column:lifecycle_updated_at"`
	DemandCaseID       *int64     `gorm:"column:demand_case_id"`
	ExperimentID       *string    `gorm:"column:experiment_id"`
	ProductID          *int64     `gorm:"column:product_id"`
	SnapshotID         *int64     `gorm:"column:snapshot_id"`
}

func (sourcingLifecycleRow) TableName() string { return "sourcing_1688_product" }
