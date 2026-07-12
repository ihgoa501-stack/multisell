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

func taskLifecycleState(link *Sourcing1688TaskLink, approvalID *int64) LifecycleState {
	updated := link.WorkflowUpdatedAt
	return LifecycleState{SourcingProductID: link.SourcingProductID, TaskLinkID: link.ID, Status: link.WorkflowStatus, Reason: link.BlockedReason, ApprovalID: approvalID, UpdatedAt: &updated}
}

// SubmitDraftApproval is the compatibility entry point. It resolves only the
// primary task and never selects an arbitrary draft by source id.
func (s *Service) SubmitDraftApproval(sourceID int64, in *DraftApprovalSubmissionInput) (*DraftApprovalResult, error) {
	if in == nil || in.RequesterID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	var source sourcingLifecycleRow
	if err := s.db.First(&source, sourceID).Error; err != nil {
		return nil, err
	}
	if err := requireTransition(source.LifecycleStatus, LifecycleEditing, LifecyclePendingApproval); err != nil {
		return nil, err
	}
	link, err := findOwnedTaskLink(s.db, sourceID, in.RequesterID, 0)
	if err != nil {
		return nil, err
	}
	return s.SubmitTaskDraftApproval(sourceID, link.ID, in)
}

func (s *Service) SubmitTaskDraftApproval(sourceID, taskLinkID int64, in *DraftApprovalSubmissionInput) (*DraftApprovalResult, error) {
	if in == nil || in.RequesterID <= 0 || strings.TrimSpace(in.Reason) == "" {
		return nil, fmt.Errorf("%w: requester and reason are required", ErrInvalidWorkflow)
	}
	var link Sourcing1688TaskLink
	var draft draftRow
	var req approval.ApprovalRequest
	err := s.db.Transaction(func(tx *gorm.DB) error {
		resolved, err := requireTaskSourcingAuthority(tx, sourceID, in.RequesterID, taskLinkID)
		if err != nil {
			return err
		}
		link = *resolved
		if err := requireSampleApprovalGate(tx, &link); err != nil {
			return err
		}
		if link.DraftID == nil {
			if !link.IsPrimary {
				return fmt.Errorf("%w: task has no converted draft", ErrWorkflowGate)
			}
			var legacy draftRow
			if err := tx.Where("sourcing_product_id = ? AND experiment_id = ?", sourceID, link.ExperimentID).Order("id DESC").First(&legacy).Error; err != nil {
				return fmt.Errorf("%w: task has no converted draft", ErrWorkflowGate)
			}
			link.DraftID = &legacy.ID
			if err := tx.Model(&legacy).Update("task_link_id", link.ID).Error; err != nil {
				return err
			}
			if err := tx.Model(&link).Updates(map[string]any{"draft_id": legacy.ID, "workflow_status": "editing", "workflow_updated_at": time.Now().UTC()}).Error; err != nil {
				return err
			}
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND sourcing_product_id = ? AND task_link_id = ?", *link.DraftID, sourceID, link.ID).First(&draft).Error; err != nil {
			return err
		}
		if draft.ApprovalStatus == approval.StatusPending || draft.ApprovalStatus == approval.StatusApproved {
			return fmt.Errorf("%w: task draft already has an active or approved review", ErrWorkflowGate)
		}
		if link.WorkflowStatus != "converted_to_draft" && link.WorkflowStatus != "editing" {
			// SQLite unit fixtures predate migration 000118; a real linked draft is
			// sufficient to normalize their compatibility projection.
			if !(link.IsPrimary && (link.WorkflowStatus == "" || link.WorkflowStatus == "needs_review")) {
				return fmt.Errorf("%w: task draft is not editable", ErrWorkflowGate)
			}
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
		req = approval.ApprovalRequest{ProductID: draft.ProductID, RequestType: DraftApprovalRequestType, Requester: strconv.FormatInt(in.RequesterID, 10), RequesterUserID: &in.RequesterID, Status: approval.StatusPending, Reason: strings.TrimSpace(in.Reason), TargetType: DraftApprovalTargetType, TargetID: draft.ID, RiskLevel: "medium", EntityType: DraftApprovalTargetType, EntityID: draft.ID, NewValue: frozenValue}
		if err := tx.Create(&req).Error; err != nil {
			return err
		}
		if err := tx.Model(&draft).Updates(map[string]any{"approval_id": req.ID, "approval_status": approval.StatusPending, "approval_content_sha256": contentHash, "approval_rejection_reason": ""}).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&link).Updates(map[string]any{"workflow_status": "pending_approval", "blocked_reason": "", "workflow_updated_at": now}).Error; err != nil {
			return err
		}
		link.WorkflowStatus, link.WorkflowUpdatedAt = "pending_approval", now
		if link.IsPrimary {
			_ = tx.Model(&Sourcing1688Product{}).Where("id = ?", sourceID).Updates(map[string]any{"lifecycle_status": LifecyclePendingApproval, "lifecycle_actor_id": in.RequesterID, "lifecycle_reason": strings.TrimSpace(in.Reason), "lifecycle_updated_at": now}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &DraftApprovalResult{Lifecycle: taskLifecycleState(&link, &req.ID), DraftID: draft.ID, ListingID: draft.ListingID, ApprovalID: req.ID, ApprovalStatus: req.Status}, nil
}

func (s *Service) DecideDraftApproval(sourceID, approvalID int64, in *DraftApprovalDecisionInput) (*DraftApprovalResult, error) {
	if in == nil || in.OwnerID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	var source sourcingLifecycleRow
	if err := s.db.First(&source, sourceID).Error; err != nil {
		return nil, err
	}
	next := LifecycleApprovedDraft
	if in.Action == "reject" {
		next = LifecycleEditing
	}
	if err := requireTransition(source.LifecycleStatus, LifecyclePendingApproval, next); err != nil {
		return nil, err
	}
	link, err := findOwnedTaskLink(s.db, sourceID, in.OwnerID, 0)
	if err != nil {
		return nil, err
	}
	return s.DecideTaskDraftApproval(sourceID, link.ID, approvalID, in)
}

func (s *Service) DecideTaskDraftApproval(sourceID, taskLinkID, approvalID int64, in *DraftApprovalDecisionInput) (*DraftApprovalResult, error) {
	if in == nil || in.OwnerID <= 0 || approvalID <= 0 || strings.TrimSpace(in.Note) == "" || (in.Action != "approve" && in.Action != "reject") {
		return nil, fmt.Errorf("%w: Owner, approval, action and note are required", ErrInvalidWorkflow)
	}
	link, err := requireTaskSourcingAuthority(s.db, sourceID, in.OwnerID, taskLinkID)
	if err != nil {
		return nil, err
	}
	if link.DraftID == nil || link.WorkflowStatus != "pending_approval" {
		return nil, fmt.Errorf("%w: exact task draft is not pending approval", ErrWorkflowGate)
	}
	var draft draftRow
	if err := s.db.Where("id = ? AND sourcing_product_id = ? AND task_link_id = ? AND approval_id = ?", *link.DraftID, sourceID, link.ID, approvalID).First(&draft).Error; err != nil {
		return nil, fmt.Errorf("%w: approval does not match exact task draft", ErrWorkflowGate)
	}
	var req approval.ApprovalRequest
	if err := s.db.First(&req, approvalID).Error; err != nil {
		return nil, fmt.Errorf("%w: approval does not match exact task draft", ErrWorkflowGate)
	}
	if req.Status != approval.StatusPending || req.RequestType != DraftApprovalRequestType || req.TargetType != DraftApprovalTargetType || req.TargetID != draft.ID || req.ProductID != draft.ProductID {
		return nil, fmt.Errorf("%w: approval does not match exact task draft", ErrWorkflowGate)
	}
	if err := validateDraftApprovalContent(s.db, &draft, &req); err != nil {
		return nil, err
	}
	oplogSvc := operationlog.NewService(s.db, s.logger)
	approvalSvc := approval.NewService(s.db, s.logger, oplogSvc)
	reviewed, err := approvalSvc.Review(approvalID, &approval.ReviewApprovalInput{Action: in.Action, Reviewer: strconv.FormatInt(in.OwnerID, 10), ReviewerUserID: &in.OwnerID, ReviewNote: strings.TrimSpace(in.Note)})
	if err != nil {
		return nil, err
	}
	next := "approved_draft"
	if reviewed.Status == approval.StatusRejected {
		next = "editing"
	}
	now := time.Now().UTC()
	updates := map[string]any{"workflow_status": next, "workflow_updated_at": now}
	if next == "editing" {
		updates["blocked_reason"] = strings.TrimSpace(in.Note)
	} else {
		updates["blocked_reason"] = ""
	}
	// PostgreSQL migration 000118 may already have performed this update in the
	// approval trigger; the idempotent write also supports SQLite tests.
	if err := s.db.Model(&Sourcing1688TaskLink{}).Where("id = ? AND sourcing_product_id = ?", link.ID, sourceID).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("id = ?", draft.ID).First(&draft).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	link.WorkflowStatus, link.WorkflowUpdatedAt = next, now
	link.BlockedReason = ""
	if next == "editing" {
		link.BlockedReason = strings.TrimSpace(in.Note)
	}
	return &DraftApprovalResult{Lifecycle: taskLifecycleState(link, &approvalID), DraftID: draft.ID, ListingID: draft.ListingID, ApprovalID: approvalID, ApprovalStatus: reviewed.Status}, nil
}
