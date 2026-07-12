package sourcing1688

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidLifecycleTransition = errors.New("invalid sourcing lifecycle transition")

func lifecycleState(row *sourcingLifecycleRow, approvalID *int64) LifecycleState {
	return LifecycleState{SourcingProductID: row.ID, Status: row.LifecycleStatus, ActorID: row.LifecycleActorID, Reason: row.LifecycleReason, ApprovalID: approvalID, UpdatedAt: row.LifecycleUpdatedAt}
}

func (s *Service) GetLifecycle(id int64) (*LifecycleState, error) {
	var row sourcingLifecycleRow
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	var draft draftRow
	var approvalID *int64
	if err := s.db.Where("sourcing_product_id = ?", id).First(&draft).Error; err == nil {
		approvalID = draft.ApprovalID
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	state := lifecycleState(&row, approvalID)
	return &state, nil
}

func requireTransition(current, expected, next string) error {
	if current != expected {
		return fmt.Errorf("%w: %s -> %s requires current state %s", ErrInvalidLifecycleTransition, current, next, expected)
	}
	return nil
}

func requireOwner(tx *gorm.DB, row *sourcingLifecycleRow, actorID int64) error {
	if actorID == 0 || row.DemandCaseID == nil {
		return fmt.Errorf("%w: Owner identity and demand case are required", ErrWorkflowGate)
	}
	var dc demandCaseRow
	if err := tx.First(&dc, *row.DemandCaseID).Error; err != nil {
		return err
	}
	if dc.OwnerID != actorID {
		return fmt.Errorf("%w: only the approved market Owner may perform this action", ErrWorkflowGate)
	}
	if dc.Status != "experiment_ready" {
		return fmt.Errorf("%w: the approved market is no longer experiment_ready", ErrWorkflowGate)
	}
	return nil
}

func updateLifecycle(tx *gorm.DB, row *sourcingLifecycleRow, next string, actorID int64, reason string) error {
	now := time.Now().UTC()
	reason = strings.TrimSpace(reason)
	if err := tx.Model(row).Updates(map[string]any{
		"lifecycle_status":     next,
		"lifecycle_actor_id":   actorID,
		"lifecycle_reason":     reason,
		"lifecycle_updated_at": now,
	}).Error; err != nil {
		return err
	}
	row.LifecycleStatus, row.LifecycleActorID, row.LifecycleReason, row.LifecycleUpdatedAt = next, &actorID, reason, &now
	return nil
}

// MarkCaptureFailed records a failed controlled capture. A reason is mandatory
// so a failed item cannot silently disappear from the Owner's review queue.
func (s *Service) MarkCaptureFailed(id int64, in *CaptureFailureInput) (*LifecycleState, error) {
	if in == nil || in.ActorID == 0 || strings.TrimSpace(in.Reason) == "" {
		return nil, fmt.Errorf("%w: actor and failure reason are required", ErrInvalidWorkflow)
	}
	var row sourcingLifecycleRow
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			return err
		}
		if err := requireTransition(row.LifecycleStatus, LifecyclePendingReview, LifecycleCaptureFailed); err != nil {
			return err
		}
		if err := requireOwner(tx, &row, in.ActorID); err != nil {
			return err
		}
		if err := tx.Model(&Sourcing1688Product{}).Where("id = ?", id).Update("status", LifecycleCaptureFailed).Error; err != nil {
			return err
		}
		return updateLifecycle(tx, &row, LifecycleCaptureFailed, in.ActorID, in.Reason)
	})
	if err != nil {
		return nil, err
	}
	state := lifecycleState(&row, nil)
	return &state, nil
}

// DecideSourceReview is the only lifecycle transition out of pending_review.
// Approval means ready_for_product; rejection persists the Owner's reason.
func (s *Service) DecideSourceReview(id int64, in *SourceReviewDecisionInput) (*LifecycleState, error) {
	if in == nil || in.OwnerID == 0 || strings.TrimSpace(in.Notes) == "" || (in.Action != "approve" && in.Action != "reject") {
		return nil, fmt.Errorf("%w: Owner, approve/reject action and notes are required", ErrInvalidWorkflow)
	}
	var row sourcingLifecycleRow
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			return err
		}
		next := LifecycleReadyForProduct
		legacy := StatusReviewed
		if in.Action == "reject" {
			next, legacy = LifecycleRejected, LifecycleRejected
		}
		if err := requireTransition(row.LifecycleStatus, LifecyclePendingReview, next); err != nil {
			return err
		}
		if err := requireOwner(tx, &row, in.OwnerID); err != nil {
			return err
		}
		if row.SnapshotID == nil {
			return fmt.Errorf("%w: immutable source snapshot is required before Owner review", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		if err := tx.Model(&Sourcing1688Product{}).Where("id = ?", id).Updates(map[string]any{
			"status": legacy, "reviewed_by": in.OwnerID, "reviewed_at": now, "review_notes": strings.TrimSpace(in.Notes),
		}).Error; err != nil {
			return err
		}
		return updateLifecycle(tx, &row, next, in.OwnerID, in.Notes)
	})
	if err != nil {
		return nil, err
	}
	state := lifecycleState(&row, nil)
	return &state, nil
}

// SubmitDraftApproval creates an approval_request and freezes the lifecycle at
// pending_approval. It never invokes a platform adapter.
func (s *Service) SubmitDraftApproval(id int64, in *DraftApprovalSubmissionInput) (*DraftApprovalResult, error) {
	if in == nil || in.RequesterID == 0 || strings.TrimSpace(in.Reason) == "" {
		return nil, fmt.Errorf("%w: requester and reason are required", ErrInvalidWorkflow)
	}
	var row sourcingLifecycleRow
	var draft draftRow
	var req approval.ApprovalRequest
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			return err
		}
		if err := requireTransition(row.LifecycleStatus, LifecycleEditing, LifecyclePendingApproval); err != nil {
			return err
		}
		if err := requireOwner(tx, &row, in.RequesterID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("sourcing_product_id = ?", id).First(&draft).Error; err != nil {
			return err
		}
		var listing listingRow
		if err := tx.First(&listing, draft.ListingID).Error; err != nil {
			return err
		}
		if listing.Status != "draft" {
			return fmt.Errorf("%w: only an internal draft may be submitted", ErrWorkflowGate)
		}
		contentHash, err := calculateDraftContentSHA256Locked(tx, &draft)
		if err != nil {
			return err
		}
		frozenValue, err := marshalDraftApprovalNewValue(contentHash)
		if err != nil {
			return err
		}
		req = approval.ApprovalRequest{
			ProductID: draft.ProductID, RequestType: DraftApprovalRequestType,
			Requester: strconv.FormatInt(in.RequesterID, 10), RequesterUserID: &in.RequesterID,
			Status: approval.StatusPending, Reason: strings.TrimSpace(in.Reason),
			TargetType: DraftApprovalTargetType, TargetID: draft.ID, RiskLevel: "medium",
			EntityType: DraftApprovalTargetType, EntityID: draft.ID,
			NewValue: frozenValue,
		}
		if err := tx.Create(&req).Error; err != nil {
			return err
		}
		if err := tx.Model(&draft).Updates(map[string]any{"approval_id": req.ID, "approval_status": approval.StatusPending, "approval_content_sha256": contentHash, "approval_rejection_reason": ""}).Error; err != nil {
			return err
		}
		return updateLifecycle(tx, &row, LifecyclePendingApproval, in.RequesterID, in.Reason)
	})
	if err != nil {
		return nil, err
	}
	return &DraftApprovalResult{Lifecycle: lifecycleState(&row, &req.ID), DraftID: draft.ID, ListingID: draft.ListingID, ApprovalID: req.ID, ApprovalStatus: req.Status}, nil
}

// DecideDraftApproval closes the exact approval attached to the draft. An
// approval produces approved_draft only; the listing status must remain draft.
// Rejection returns the item to editing and persists the rejection reason.
func (s *Service) DecideDraftApproval(id, approvalID int64, in *DraftApprovalDecisionInput) (*DraftApprovalResult, error) {
	if in == nil || in.OwnerID == 0 || approvalID == 0 || strings.TrimSpace(in.Note) == "" || (in.Action != "approve" && in.Action != "reject") {
		return nil, fmt.Errorf("%w: Owner, approval, approve/reject action and note are required", ErrInvalidWorkflow)
	}
	// Read-only preflight provides deterministic domain errors. The PostgreSQL
	// trigger repeats these checks atomically when approval.Service.Review writes.
	var preflight sourcingLifecycleRow
	if err := s.db.First(&preflight, id).Error; err != nil {
		return nil, err
	}
	expected := LifecycleApprovedDraft
	if in.Action == "reject" {
		expected = LifecycleEditing
	}
	if err := requireTransition(preflight.LifecycleStatus, LifecyclePendingApproval, expected); err != nil {
		return nil, err
	}
	if err := requireOwner(s.db, &preflight, in.OwnerID); err != nil {
		return nil, err
	}
	var preflightDraft draftRow
	if err := s.db.Where("sourcing_product_id = ?", id).First(&preflightDraft).Error; err != nil {
		return nil, err
	}
	if preflightDraft.ApprovalID == nil || *preflightDraft.ApprovalID != approvalID {
		return nil, fmt.Errorf("%w: approval does not belong to this draft", ErrWorkflowGate)
	}
	var preflightReq approval.ApprovalRequest
	if err := s.db.First(&preflightReq, approvalID).Error; err != nil {
		return nil, err
	}
	if preflightReq.Status != approval.StatusPending || preflightReq.RequestType != DraftApprovalRequestType || preflightReq.TargetType != DraftApprovalTargetType || preflightReq.TargetID != preflightDraft.ID || preflightReq.ProductID != preflightDraft.ProductID {
		return nil, fmt.Errorf("%w: pending approval does not match the draft", ErrWorkflowGate)
	}
	if err := validateDraftApprovalContent(s.db, &preflightDraft, &preflightReq); err != nil {
		return nil, err
	}
	var preflightListing listingRow
	if err := s.db.First(&preflightListing, preflightDraft.ListingID).Error; err != nil {
		return nil, err
	}
	if preflightListing.Status != "draft" {
		return nil, fmt.Errorf("%w: approval target is no longer an internal draft", ErrWorkflowGate)
	}
	// The canonical approval service owns the approval decision, structured
	// operation_log entry and lifecycle event. Migration 000085 atomically
	// validates Owner/draft linkage and applies the sourcing lifecycle state in
	// the same approval_request transaction.
	oplogSvc := operationlog.NewService(s.db, s.logger)
	approvalSvc := approval.NewService(s.db, s.logger, oplogSvc)
	req, err := approvalSvc.Review(approvalID, &approval.ReviewApprovalInput{
		Action: in.Action, Reviewer: strconv.FormatInt(in.OwnerID, 10),
		ReviewerUserID: &in.OwnerID, ReviewNote: strings.TrimSpace(in.Note),
	})
	if err != nil {
		return nil, err
	}
	var row sourcingLifecycleRow
	if err := s.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	var draft draftRow
	if err := s.db.Where("sourcing_product_id = ? AND approval_id = ?", id, approvalID).First(&draft).Error; err != nil {
		return nil, err
	}
	if row.LifecycleStatus != expected {
		return nil, fmt.Errorf("%w: approval completed without expected lifecycle state", ErrWorkflowGate)
	}
	return &DraftApprovalResult{Lifecycle: lifecycleState(&row, &approvalID), DraftID: draft.ID, ListingID: draft.ListingID, ApprovalID: approvalID, ApprovalStatus: req.Status}, nil
}
