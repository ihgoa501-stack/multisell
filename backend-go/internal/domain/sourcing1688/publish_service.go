package sourcing1688

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/integrations"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PublishApprovalRequestType = "sourcing_1688_publish"
	PublishApprovalTargetType  = "sourcing_publish_attempt"
	PublishStatusPending       = "pending_approval"
	PublishStatusApproved      = "approved"
	PublishStatusRejected      = "rejected"
	PublishStatusExecuting     = "executing"
	PublishStatusSubmitted     = "submitted"
	PublishStatusReconcile     = "reconcile_required"
	PublishStatusSucceeded     = "succeeded"
	PublishStatusFailed        = "failed"
	publishApprovalTTL         = 24 * time.Hour
)

var (
	errPublishAdapterUnavailable = errors.New("publish adapter unavailable")
	errPublishCredentialsInvalid = errors.New("publish credentials invalid")
)

type publishAdapter interface {
	Publish(context.Context, *integrations.PublishInput) (*integrations.PublishResult, error)
	ValidateCredentials(context.Context, int64) (bool, error)
}

type publishAdapterResolver func(string) (publishAdapter, bool)

type PublishRequestInput struct {
	TaskLinkID        int64          `json:"task_link_id,omitempty"`
	RequesterID       int64          `json:"requester_id"`
	PlatformAccountID int64          `json:"platform_account_id" binding:"required"`
	IdempotencyKey    string         `json:"idempotency_key" binding:"required"`
	Reason            string         `json:"reason" binding:"required"`
	Inventories       map[string]int `json:"inventories" binding:"required"`
}

type PublishDecisionInput struct {
	TaskLinkID int64  `json:"task_link_id,omitempty"`
	OwnerID    int64  `json:"owner_id"`
	Action     string `json:"action" binding:"required"`
	Note       string `json:"note" binding:"required"`
}

type PublishReconcileInput struct {
	TaskLinkID     int64                      `json:"task_link_id,omitempty"`
	OwnerID        int64                      `json:"owner_id"`
	Outcome        string                     `json:"outcome" binding:"required"` // submitted or failed
	EvidenceURI    string                     `json:"evidence_uri" binding:"required"`
	ObservedAt     time.Time                  `json:"observed_at" binding:"required"`
	TruthStatus    string                     `json:"truth_status" binding:"required"`
	PlatformResult integrations.PublishResult `json:"platform_result"`
}

type reconciledPublishEvidence struct {
	PlatformResult integrations.PublishResult `json:"platform_result"`
	EvidenceURI    string                     `json:"evidence_uri"`
	ObservedAt     time.Time                  `json:"observed_at"`
	TruthStatus    string                     `json:"truth_status"`
}

type PublishAttempt struct {
	ID                    int64           `json:"id"`
	SourcingProductID     int64           `json:"sourcing_product_id"`
	DraftID               int64           `json:"draft_id"`
	TaskLinkID            int64           `json:"task_link_id"`
	ProductID             int64           `json:"product_id"`
	ListingID             int64           `json:"listing_id"`
	PlatformID            int64           `json:"platform_id"`
	PlatformAccountID     int64           `json:"platform_account_id"`
	ExperimentID          string          `json:"experiment_id"`
	IdempotencyKey        string          `json:"idempotency_key"`
	RequestSHA256         string          `json:"request_sha256"`
	Status                string          `json:"status"`
	ErrorMessage          string          `json:"error_message,omitempty"`
	ApprovalID            *int64          `json:"approval_id,omitempty"`
	RequestPayload        json.RawMessage `json:"request_payload"`
	AdapterRequestPayload json.RawMessage `json:"adapter_request_payload,omitempty"`
	ResponsePayload       json.RawMessage `json:"response_payload,omitempty"`
	ResponseSHA256        string          `json:"response_sha256,omitempty"`
	RequestedBy           int64           `json:"requested_by"`
	RequestedAt           time.Time       `json:"requested_at"`
	ApprovedBy            *int64          `json:"approved_by,omitempty"`
	ApprovedAt            *time.Time      `json:"approved_at,omitempty"`
	ExecutedAt            *time.Time      `json:"executed_at,omitempty"`
	CompletedAt           *time.Time      `json:"completed_at,omitempty"`
}

func (PublishAttempt) TableName() string { return "sourcing_publish_attempt" }

type platformAccountRow struct {
	ID, PlatformID int64
	Status         string
	ExecutionMode  int8
}

func (platformAccountRow) TableName() string { return "platform_integration_account" }

type publishRequestSnapshot struct {
	SourcingProductID int64          `json:"sourcing_product_id"`
	TaskLinkID        int64          `json:"task_link_id"`
	DraftID           int64          `json:"draft_id"`
	ProductID         int64          `json:"product_id"`
	ListingID         int64          `json:"listing_id"`
	PlatformID        int64          `json:"platform_id"`
	PlatformAccountID int64          `json:"platform_account_id"`
	Inventories       map[string]int `json:"inventories"`
}

type publishApprovalEnvelope struct {
	Snapshot       publishRequestSnapshot `json:"snapshot"`
	AdapterRequest json.RawMessage        `json:"adapter_request"`
}

func buildFrozenPublishInput(tx *gorm.DB, draft *draftRow, listing *listingRow, accountID int64, idempotencyKey string, inventoriesByCode map[string]int) (*integrations.PublishInput, error) {
	var product productRow
	if err := tx.First(&product, draft.ProductID).Error; err != nil {
		return nil, err
	}
	var skus []skuRow
	if err := tx.Where("product_id = ?", product.ID).Order("id").Find(&skus).Error; err != nil {
		return nil, err
	}
	if len(skus) == 0 || len(inventoriesByCode) != len(skus) {
		return nil, fmt.Errorf("%w: inventory snapshot must cover every current SKU", ErrWorkflowGate)
	}
	publishSKUs := make([]integrations.PublishSKU, 0, len(skus))
	prices, inventories := map[int64]string{}, map[int64]int{}
	for _, sku := range skus {
		quantity, ok := inventoriesByCode[sku.Code]
		if !ok {
			return nil, fmt.Errorf("%w: inventory snapshot missing SKU %s", ErrWorkflowGate, sku.Code)
		}
		publishSKUs = append(publishSKUs, integrations.PublishSKU{SkuID: sku.ID, SkuCode: sku.Code})
		prices[sku.ID] = strconv.FormatFloat(sku.Price, 'f', 2, 64)
		inventories[sku.ID] = quantity
	}
	var listingData struct {
		LocalizedTitle       string `json:"localized_title"`
		LocalizedDescription string `json:"localized_description"`
	}
	_ = json.Unmarshal(listing.PublishedData, &listingData)
	input := &integrations.PublishInput{ProductID: product.ID, PlatformID: listing.PlatformID, AccountID: accountID, SKUs: publishSKUs, Prices: prices, Inventories: inventories, IdempotencyKey: idempotencyKey, ProductName: listingData.LocalizedTitle, Description: listingData.LocalizedDescription, CategoryID: product.CategoryID, MainImage: product.MainImage}
	if input.ProductName == "" {
		input.ProductName = product.Name
	}
	if input.Description == "" {
		input.Description = product.Description
	}
	return input, nil
}

func validatePublishRequestInput(in *PublishRequestInput) error {
	if in == nil || in.RequesterID <= 0 || in.PlatformAccountID <= 0 || strings.TrimSpace(in.Reason) == "" || strings.TrimSpace(in.IdempotencyKey) == "" {
		return fmt.Errorf("%w: requester, platform account, reason and idempotency key are required", ErrInvalidWorkflow)
	}
	if len(in.IdempotencyKey) > 160 || len(in.Inventories) == 0 {
		return fmt.Errorf("%w: idempotency key or inventory snapshot is invalid", ErrInvalidWorkflow)
	}
	for sku, quantity := range in.Inventories {
		if strings.TrimSpace(sku) == "" || quantity < 0 {
			return fmt.Errorf("%w: inventory SKU and quantity are invalid", ErrInvalidWorkflow)
		}
	}
	return nil
}

func equalInventories(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for sku, quantity := range a {
		other, ok := b[sku]
		if !ok || other != quantity {
			return false
		}
	}
	return true
}

// requirePublishSourcingAuthority verifies the current, frozen Owner authority
// for this exact sourcing product. experimentID is only a trace key used to
// locate the task link; experiment state and its historical gate never grant
// publish authority.
func requirePublishSourcingAuthority(tx *gorm.DB, sourceID, ownerID, demandCaseID int64, experimentID string) error {
	var link Sourcing1688TaskLink
	if err := tx.Where(
		"sourcing_product_id = ? AND owner_id = ? AND demand_case_id = ? AND experiment_id = ? AND is_primary = ? AND authority_kind = ?",
		sourceID, ownerID, demandCaseID, experimentID, true, "product_opportunity",
	).First(&link).Error; err != nil || link.ProductOpportunityID == nil || link.OpportunityDecisionID == nil {
		return fmt.Errorf("%w: frozen product opportunity authority for this source is required", ErrWorkflowGate)
	}
	decision, err := requireSourcingAuthority(tx, ownerID, demandCaseID, *link.ProductOpportunityID)
	if err != nil {
		return err
	}
	if decision.ID != *link.OpportunityDecisionID {
		return fmt.Errorf("%w: frozen product opportunity approval no longer matches", ErrWorkflowGate)
	}
	return nil
}

func requireOnlyActivePlatformAccount(tx *gorm.DB, accountID, platformID int64) error {
	var activeCount int64
	if err := tx.Model(&platformAccountRow{}).Where("platform_id = ? AND status = ?", platformID, "active").Count(&activeCount).Error; err != nil {
		return err
	}
	if activeCount != 1 {
		return fmt.Errorf("%w: current adapters require exactly one active account for the selected platform", ErrWorkflowGate)
	}
	var account platformAccountRow
	if err := tx.First(&account, accountID).Error; err != nil {
		return err
	}
	if account.PlatformID != platformID || account.Status != "active" {
		return fmt.Errorf("%w: approved platform account is not the sole active account", ErrWorkflowGate)
	}
	return nil
}

// RequestPublish freezes the exact account and inventory request behind a new,
// high-risk approval. It never invokes an integration adapter.
func (s *Service) RequestPublish(sourceID int64, in *PublishRequestInput) (*PublishAttempt, error) {
	if err := validatePublishRequestInput(in); err != nil {
		return nil, err
	}
	var attempt PublishAttempt
	err := s.db.Transaction(func(tx *gorm.DB) error {
		task, err := requireTaskSourcingAuthority(tx, sourceID, in.RequesterID, in.TaskLinkID)
		if err != nil {
			return err
		}
		var replay PublishAttempt
		if err := tx.Where("idempotency_key = ?", strings.TrimSpace(in.IdempotencyKey)).First(&replay).Error; err == nil {
			var envelope publishApprovalEnvelope
			legacyPrimary := task.IsPrimary && replay.TaskLinkID == 0 && envelope.Snapshot.TaskLinkID == 0
			if replay.SourcingProductID != sourceID || (!legacyPrimary && replay.TaskLinkID != task.ID) || replay.RequestedBy != in.RequesterID || replay.PlatformAccountID != in.PlatformAccountID || json.Unmarshal(replay.RequestPayload, &envelope) != nil || (!legacyPrimary && envelope.Snapshot.TaskLinkID != task.ID) || !equalInventories(envelope.Snapshot.Inventories, in.Inventories) {
				return fmt.Errorf("%w: idempotency key is already bound to another publish request", ErrWorkflowGate)
			}
			attempt = replay
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var source sourcingLifecycleRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&source, sourceID).Error; err != nil {
			return err
		}
		if task.WorkflowStatus != "approved_draft" && !(task.IsPrimary && source.LifecycleStatus == LifecycleApprovedDraft && (task.WorkflowStatus == "" || task.WorkflowStatus == "needs_review")) {
			return fmt.Errorf("%w: exact task requires a separately approved internal draft", ErrWorkflowGate)
		}
		var activeAttempts int64
		if err := tx.Model(&PublishAttempt{}).
			Where("task_link_id = ? AND status IN ?", task.ID, []string{PublishStatusPending, PublishStatusApproved, PublishStatusExecuting, PublishStatusSubmitted, PublishStatusReconcile}).
			Count(&activeAttempts).Error; err != nil {
			return err
		}
		if activeAttempts > 0 {
			return fmt.Errorf("%w: an unresolved publish request already exists for this source", ErrWorkflowGate)
		}
		var draft draftRow
		if task.DraftID != nil {
			if err := tx.Where("id = ? AND sourcing_product_id = ? AND task_link_id = ?", *task.DraftID, sourceID, task.ID).First(&draft).Error; err != nil {
				return err
			}
		} else if !task.IsPrimary {
			return fmt.Errorf("%w: exact task draft identity is missing", ErrWorkflowGate)
		} else if err := tx.Where("sourcing_product_id = ? AND experiment_id = ? AND task_link_id IS NULL", sourceID, task.ExperimentID).First(&draft).Error; err != nil {
			return err
		}
		if draft.ApprovalID == nil || draft.ApprovalStatus != approval.StatusApproved {
			return fmt.Errorf("%w: draft approval is not valid", ErrWorkflowGate)
		}
		var draftApproval approval.ApprovalRequest
		if err := tx.First(&draftApproval, *draft.ApprovalID).Error; err != nil {
			return err
		}
		if draftApproval.Status != approval.StatusApproved || draftApproval.ReviewerUserID == nil || *draftApproval.ReviewerUserID != in.RequesterID {
			return fmt.Errorf("%w: draft approval Owner is invalid", ErrWorkflowGate)
		}
		if err := validateDraftApprovalContentLocked(tx, &draft, &draftApproval); err != nil {
			return err
		}
		var listing listingRow
		if err := tx.First(&listing, draft.ListingID).Error; err != nil {
			return err
		}
		if listing.Status != "draft" {
			return fmt.Errorf("%w: only an internal draft may request publication", ErrWorkflowGate)
		}
		var account platformAccountRow
		if err := tx.First(&account, in.PlatformAccountID).Error; err != nil {
			return err
		}
		if account.PlatformID != listing.PlatformID || account.Status != "active" || (account.ExecutionMode != int8(integrations.ExecutionModeApprovalRequired) && account.ExecutionMode != int8(integrations.ExecutionModeProduction)) {
			return fmt.Errorf("%w: active write-enabled platform account must match the approved channel", ErrWorkflowGate)
		}
		if draft.ExperimentID != task.ExperimentID || draft.DemandCaseID != task.DemandCaseID {
			return fmt.Errorf("%w: task and draft authority trace do not match", ErrWorkflowGate)
		}
		if err := requireOnlyActivePlatformAccount(tx, account.ID, listing.PlatformID); err != nil {
			return err
		}
		snapshot := publishRequestSnapshot{SourcingProductID: sourceID, TaskLinkID: task.ID, DraftID: draft.ID, ProductID: draft.ProductID, ListingID: draft.ListingID, PlatformID: listing.PlatformID, PlatformAccountID: account.ID, Inventories: in.Inventories}
		adapterInput, err := buildFrozenPublishInput(tx, &draft, &listing, account.ID, strings.TrimSpace(in.IdempotencyKey), in.Inventories)
		if err != nil {
			return err
		}
		adapterPayload, err := json.Marshal(adapterInput)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(publishApprovalEnvelope{Snapshot: snapshot, AdapterRequest: adapterPayload})
		if err != nil {
			return err
		}
		hash := sha256.Sum256(payload)
		requestHash := hex.EncodeToString(hash[:])
		var existing PublishAttempt
		findErr := tx.Where("idempotency_key = ?", strings.TrimSpace(in.IdempotencyKey)).First(&existing).Error
		if findErr == nil {
			if existing.SourcingProductID != sourceID || existing.RequestSHA256 != requestHash {
				return fmt.Errorf("%w: idempotency key is already bound to another publish request", ErrWorkflowGate)
			}
			attempt = existing
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		attempt = PublishAttempt{SourcingProductID: sourceID, TaskLinkID: task.ID, DraftID: draft.ID, ProductID: draft.ProductID, ListingID: draft.ListingID, PlatformID: listing.PlatformID, PlatformAccountID: account.ID, ExperimentID: draft.ExperimentID, IdempotencyKey: strings.TrimSpace(in.IdempotencyKey), RequestSHA256: requestHash, RequestPayload: payload, AdapterRequestPayload: adapterPayload, Status: PublishStatusPending, RequestedBy: in.RequesterID, RequestedAt: time.Now().UTC()}
		if err := tx.Create(&attempt).Error; err != nil {
			return err
		}
		expiresAt := time.Now().UTC().Add(publishApprovalTTL)
		req := approval.ApprovalRequest{ProductID: draft.ProductID, RequestType: PublishApprovalRequestType, Requester: strconv.FormatInt(in.RequesterID, 10), RequesterUserID: &in.RequesterID, Status: approval.StatusPending, NewValue: string(payload), Reason: strings.TrimSpace(in.Reason), TargetType: PublishApprovalTargetType, TargetID: attempt.ID, RiskLevel: "high", ExpiresAt: &expiresAt, EntityType: PublishApprovalTargetType, EntityID: attempt.ID}
		if err := tx.Create(&req).Error; err != nil {
			return err
		}
		attempt.ApprovalID = &req.ID
		if err := tx.Model(&attempt).Update("approval_id", req.ID).Error; err != nil {
			return err
		}
		if err := tx.Model(task).Updates(map[string]any{"workflow_status": "publish_pending", "workflow_updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		return tx.Create(&operationlog.OperationLog{Module: "sourcing1688", Action: "publish.request", ResourceID: strconv.FormatInt(attempt.ID, 10), Operator: strconv.FormatInt(in.RequesterID, 10), UserID: in.RequesterID, Content: fmt.Sprintf("publish_attempt=%d platform_account=%d", attempt.ID, account.ID), Result: PublishStatusPending, TriggerType: "manual", ApprovalID: &req.ID, EntityType: PublishApprovalTargetType, EntityID: attempt.ID}).Error
	})
	if err != nil {
		return nil, err
	}
	return &attempt, nil
}

func (s *Service) DecidePublish(sourceID, attemptID int64, in *PublishDecisionInput) (*PublishAttempt, error) {
	if in == nil || in.OwnerID <= 0 || strings.TrimSpace(in.Note) == "" || (in.Action != "approve" && in.Action != "reject") {
		return nil, fmt.Errorf("%w: Owner, approve/reject action and note are required", ErrInvalidWorkflow)
	}
	var attempt PublishAttempt
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND sourcing_product_id = ?", attemptID, sourceID).First(&attempt).Error; err != nil {
			return err
		}
		if attempt.Status != PublishStatusPending || attempt.ApprovalID == nil {
			return fmt.Errorf("%w: publish request is not pending approval", ErrWorkflowGate)
		}
		task, err := requireTaskSourcingAuthority(tx, sourceID, in.OwnerID, in.TaskLinkID)
		if err != nil {
			return err
		}
		if attempt.TaskLinkID != task.ID && !(task.IsPrimary && attempt.TaskLinkID == 0) {
			return fmt.Errorf("%w: publish request belongs to another task", ErrWorkflowGate)
		}
		var req approval.ApprovalRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&req, *attempt.ApprovalID).Error; err != nil {
			return err
		}
		if req.RequestType != PublishApprovalRequestType || req.TargetType != PublishApprovalTargetType || req.TargetID != attempt.ID || req.ProductID != attempt.ProductID || req.Status != approval.StatusPending {
			return fmt.Errorf("%w: approval does not match this publish request", ErrWorkflowGate)
		}
		oplogSvc := operationlog.NewService(tx, s.logger)
		approvalSvc := approval.NewService(tx, s.logger, oplogSvc)
		reviewed, err := approvalSvc.Review(req.ID, &approval.ReviewApprovalInput{Action: in.Action, Reviewer: strconv.FormatInt(in.OwnerID, 10), ReviewerUserID: &in.OwnerID, ReviewNote: strings.TrimSpace(in.Note)})
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		status := PublishStatusApproved
		updates := map[string]any{"status": status}
		if reviewed.Status == approval.StatusRejected {
			status = PublishStatusRejected
			updates["status"] = status
		} else {
			updates["approved_by"] = in.OwnerID
			updates["approved_at"] = now
		}
		updated := tx.Model(&PublishAttempt{}).Where("id = ? AND status = ?", attempt.ID, PublishStatusPending).Updates(updates)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return fmt.Errorf("%w: publish request was concurrently changed", ErrWorkflowGate)
		}
		taskStatus := "publish_approved"
		if status == PublishStatusRejected {
			taskStatus = "approved_draft"
		}
		if err := tx.Model(task).Updates(map[string]any{"workflow_status": taskStatus, "workflow_updated_at": now}).Error; err != nil {
			return err
		}
		return tx.First(&attempt, attempt.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &attempt, nil
}

// ExecutePublish is the only external-write seam. It atomically claims an
// approved idempotency key before the call and never retries failed/ambiguous
// calls automatically.
func (s *Service) ExecutePublish(ctx context.Context, sourceID, attemptID, ownerID int64) (*PublishAttempt, error) {
	link, err := findOwnedTaskLink(s.db, sourceID, ownerID, 0)
	if err != nil {
		return nil, err
	}
	return s.executePublishForTask(ctx, sourceID, link.ID, attemptID, ownerID)
}

func (s *Service) executePublishForTask(ctx context.Context, sourceID, taskLinkID, attemptID, ownerID int64) (*PublishAttempt, error) {
	var attempt PublishAttempt
	var input integrations.PublishInput
	var platformCode string
	var executionMode integrations.ExecutionMode
	shouldExecute := false
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND sourcing_product_id = ?", attemptID, sourceID).First(&attempt).Error; err != nil {
			return err
		}
		task, err := requireTaskSourcingAuthority(tx, sourceID, ownerID, attempt.TaskLinkID)
		if err != nil {
			return err
		}
		if attempt.Status == PublishStatusSubmitted || attempt.Status == PublishStatusReconcile || attempt.Status == PublishStatusSucceeded || attempt.Status == PublishStatusFailed || attempt.Status == PublishStatusExecuting {
			if task.ExperimentID != attempt.ExperimentID {
				return fmt.Errorf("%w: task authority trace does not match this publish request", ErrWorkflowGate)
			}
			return nil
		}
		if attempt.Status != PublishStatusApproved || attempt.ApprovalID == nil {
			return fmt.Errorf("%w: a valid second Owner publish approval is required", ErrWorkflowGate)
		}
		var source sourcingLifecycleRow
		if err := tx.First(&source, sourceID).Error; err != nil {
			return err
		}
		if task.WorkflowStatus != "publish_approved" || requireOwner(tx, &source, ownerID) != nil {
			return fmt.Errorf("%w: approved draft and selected market Owner are required", ErrWorkflowGate)
		}
		if err := requireCurrentCompliance(tx, sourceID, task.ID, ownerID, StandardPublishComplianceRequirementCodes, time.Now().UTC()); err != nil {
			return err
		}
		var req approval.ApprovalRequest
		if err := tx.First(&req, *attempt.ApprovalID).Error; err != nil {
			return err
		}
		if req.Status != approval.StatusApproved || req.RequestType != PublishApprovalRequestType || req.TargetType != PublishApprovalTargetType || req.TargetID != attempt.ID || req.ProductID != attempt.ProductID || req.ReviewerUserID == nil || *req.ReviewerUserID != ownerID || (req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now())) {
			return fmt.Errorf("%w: publish approval is expired or does not match", ErrWorkflowGate)
		}
		var draft draftRow
		if err := tx.First(&draft, attempt.DraftID).Error; err != nil {
			return err
		}
		if draft.SourcingProductID != sourceID || (draft.TaskLinkID == nil && !task.IsPrimary) || (draft.TaskLinkID != nil && *draft.TaskLinkID != task.ID) || draft.ProductID != attempt.ProductID || draft.ListingID != attempt.ListingID {
			return fmt.Errorf("%w: draft linkage changed after approval", ErrWorkflowGate)
		}
		if draft.ApprovalID == nil {
			return fmt.Errorf("%w: approved draft fingerprint is missing", ErrWorkflowGate)
		}
		var draftApproval approval.ApprovalRequest
		if err := tx.First(&draftApproval, *draft.ApprovalID).Error; err != nil {
			return err
		}
		if draftApproval.Status != approval.StatusApproved || draftApproval.ReviewerUserID == nil || *draftApproval.ReviewerUserID != ownerID {
			return fmt.Errorf("%w: approved draft Owner is invalid", ErrWorkflowGate)
		}
		if err := validateDraftApprovalContentLocked(tx, &draft, &draftApproval); err != nil {
			return err
		}
		if draft.ExperimentID != attempt.ExperimentID || task.ExperimentID != attempt.ExperimentID {
			return fmt.Errorf("%w: source, market and draft experiment do not match", ErrWorkflowGate)
		}
		var listing listingRow
		if err := tx.First(&listing, attempt.ListingID).Error; err != nil {
			return err
		}
		var account platformAccountRow
		if err := tx.First(&account, attempt.PlatformAccountID).Error; err != nil {
			return err
		}
		if listing.Status != "draft" || listing.PlatformID != attempt.PlatformID || account.PlatformID != attempt.PlatformID || account.Status != "active" || (account.ExecutionMode != int8(integrations.ExecutionModeApprovalRequired) && account.ExecutionMode != int8(integrations.ExecutionModeProduction)) {
			return fmt.Errorf("%w: draft or platform account changed after approval", ErrWorkflowGate)
		}
		if err := requireOnlyActivePlatformAccount(tx, account.ID, attempt.PlatformID); err != nil {
			return err
		}
		var platform platformRow
		if err := tx.First(&platform, attempt.PlatformID).Error; err != nil {
			return err
		}
		platformCode = platform.Code
		executionMode = integrations.ExecutionMode(account.ExecutionMode)
		var dc demandCaseRow
		if tx.First(&dc, task.DemandCaseID).Error != nil {
			return fmt.Errorf("%w: selected market is unavailable", ErrWorkflowGate)
		}
		channel := strings.ToLower(dc.SalesChannel)
		if !strings.Contains(channel, strings.ToLower(platform.Code)) && !strings.Contains(channel, strings.ToLower(platform.Name)) {
			return fmt.Errorf("%w: platform no longer matches selected sales channel", ErrWorkflowGate)
		}
		var envelope publishApprovalEnvelope
		hash := sha256.Sum256(attempt.RequestPayload)
		if json.Unmarshal(attempt.RequestPayload, &envelope) != nil || hex.EncodeToString(hash[:]) != attempt.RequestSHA256 || envelope.Snapshot.PlatformAccountID != account.ID || envelope.Snapshot.ListingID != listing.ID || envelope.Snapshot.ProductID != attempt.ProductID || string(envelope.AdapterRequest) != string(attempt.AdapterRequestPayload) {
			return fmt.Errorf("%w: frozen publish request is invalid", ErrWorkflowGate)
		}
		if err := json.Unmarshal(attempt.AdapterRequestPayload, &input); err != nil || input.ProductID != attempt.ProductID || input.PlatformID != attempt.PlatformID || input.AccountID != attempt.PlatformAccountID || input.IdempotencyKey != attempt.IdempotencyKey {
			return fmt.Errorf("%w: frozen adapter request is invalid", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		updated := tx.Model(&attempt).Where("status = ?", PublishStatusApproved).Updates(map[string]any{"status": PublishStatusExecuting, "executed_at": now})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return fmt.Errorf("%w: publish request was concurrently claimed", ErrWorkflowGate)
		}
		if err := tx.Model(task).Updates(map[string]any{"workflow_status": "publishing", "workflow_updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Create(&operationlog.OperationLog{Module: "sourcing1688", Action: "publish.execute.claimed", ResourceID: strconv.FormatInt(attempt.ID, 10), Operator: strconv.FormatInt(ownerID, 10), UserID: ownerID, Content: fmt.Sprintf("publish_attempt=%d platform_account=%d", attempt.ID, attempt.PlatformAccountID), Result: PublishStatusExecuting, TriggerType: "owner_approval", ApprovalID: attempt.ApprovalID, EntityType: PublishApprovalTargetType, EntityID: attempt.ID}).Error; err != nil {
			return err
		}
		shouldExecute = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !shouldExecute {
		return &attempt, nil
	}
	normalizedCode := strings.ToLower(strings.TrimSpace(platformCode))
	if normalizedCode == "sandbox" || normalizedCode == "mock" || strings.HasPrefix(normalizedCode, "mock-") || strings.HasSuffix(normalizedCode, "-sandbox") {
		return s.finishPublishAttempt(ctx, &attempt, nil, errPublishAdapterUnavailable)
	}
	adapter, ok := s.resolvePublisher(platformCode)
	if !ok || adapter == nil {
		return s.finishPublishAttempt(ctx, &attempt, nil, errPublishAdapterUnavailable)
	}
	callCtx, cancelCall := context.WithTimeout(ctx, 30*time.Second)
	defer cancelCall()
	callCtx = integrations.WithApprovalID(callCtx, *attempt.ApprovalID)
	callCtx = integrations.WithExecutionMode(callCtx, executionMode)
	valid, credentialErr := adapter.ValidateCredentials(callCtx, attempt.PlatformAccountID)
	if credentialErr != nil || !valid {
		if errors.Is(credentialErr, context.DeadlineExceeded) || errors.Is(credentialErr, context.Canceled) {
			return s.finishPublishAttempt(ctx, &attempt, nil, credentialErr)
		}
		return s.finishPublishAttempt(ctx, &attempt, nil, errPublishCredentialsInvalid)
	}
	result, publishErr := adapter.Publish(callCtx, &input)
	if publishErr == nil && (result == nil || strings.TrimSpace(result.PlatformProductID) == "") {
		publishErr = errors.New("platform did not confirm a product identifier")
	}
	return s.finishPublishAttempt(ctx, &attempt, result, publishErr)
}

func (s *Service) finishPublishAttempt(ctx context.Context, attempt *PublishAttempt, result *integrations.PublishResult, callErr error) (*PublishAttempt, error) {
	completedAt := time.Now().UTC()
	status, safeError := PublishStatusSubmitted, ""
	response, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		callErr = errors.New("platform result serialization failed")
		response = nil
	}
	if callErr != nil || result == nil {
		status = PublishStatusFailed
		if errors.Is(callErr, context.DeadlineExceeded) || errors.Is(callErr, context.Canceled) {
			status = PublishStatusReconcile
		}
		if callErr == nil {
			callErr = errors.New("platform returned no publish result")
		}
		safeError = classifyPublishError(callErr)
	}
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	err := s.db.WithContext(persistCtx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"status": status, "response_payload": response, "error_message": safeError, "completed_at": completedAt}
		responseHash := sha256.Sum256(response)
		updates["response_sha256"] = hex.EncodeToString(responseHash[:])
		updated := tx.Model(&PublishAttempt{}).Where("id = ? AND status = ?", attempt.ID, PublishStatusExecuting).Updates(updates)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return fmt.Errorf("publish attempt state changed while completing")
		}
		taskStatus := "publish_failed"
		if status == PublishStatusSubmitted {
			taskStatus = "submitted"
		} else if status == PublishStatusReconcile {
			taskStatus = "reconcile_required"
		}
		if err := tx.Model(&Sourcing1688TaskLink{}).Where("id = ? AND sourcing_product_id = ?", attempt.TaskLinkID, attempt.SourcingProductID).Updates(map[string]any{"workflow_status": taskStatus, "workflow_updated_at": completedAt}).Error; err != nil {
			return err
		}
		if status == PublishStatusSubmitted {
			updatedListing := tx.Model(&listingRow{}).Where("id = ? AND status = ?", attempt.ListingID, "draft").Updates(map[string]any{"status": "submitted", "published_data": response, "platform_product_id": result.PlatformProductID, "platform_url": result.PlatformURL, "sync_message": result.SyncMessage, "last_sync_at": completedAt})
			if updatedListing.Error != nil {
				return updatedListing.Error
			}
			if updatedListing.RowsAffected != 1 {
				return fmt.Errorf("listing changed while platform result was being recorded")
			}
		}
		return tx.Create(&operationlog.OperationLog{Module: "sourcing1688", Action: "publish.execute", ResourceID: strconv.FormatInt(attempt.ID, 10), Operator: strconv.FormatInt(attempt.RequestedBy, 10), UserID: attempt.RequestedBy, Content: fmt.Sprintf("publish_attempt=%d platform_account=%d", attempt.ID, attempt.PlatformAccountID), Result: status, TriggerType: "owner_approval", ApprovalID: attempt.ApprovalID, EntityType: PublishApprovalTargetType, EntityID: attempt.ID}).Error
	})
	if err != nil {
		return nil, err
	}
	if err := s.db.First(attempt, attempt.ID).Error; err != nil {
		return nil, err
	}
	if callErr != nil {
		return attempt, fmt.Errorf("platform publish failed; result retained in attempt %d", attempt.ID)
	}
	return attempt, nil
}

func classifyPublishError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, errPublishAdapterUnavailable):
		return "adapter_unavailable"
	case errors.Is(err, errPublishCredentialsInvalid):
		return "credentials_invalid"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "platform_publish_failed"
	}
}

func (s *Service) ListPublishAttempts(sourceID int64) ([]PublishAttempt, error) {
	var items []PublishAttempt
	err := s.db.Where("sourcing_product_id = ?", sourceID).Order("id DESC").Find(&items).Error
	return items, err
}

func (s *Service) ListTaskPublishAttempts(sourceID, taskLinkID, ownerID int64) ([]PublishAttempt, error) {
	if _, err := requireTaskSourcingAuthority(s.db, sourceID, ownerID, taskLinkID); err != nil {
		return nil, err
	}
	var items []PublishAttempt
	err := s.db.Where("sourcing_product_id = ? AND task_link_id = ?", sourceID, taskLinkID).Order("id DESC").Find(&items).Error
	return items, err
}

// ReconcilePublish records a platform-observed/manual result after an
// ambiguous timeout or interrupted persistence. It has no external side
// effect and can never claim succeeded; submitted still requires later status
// synchronization from the platform.
func (s *Service) ReconcilePublish(ctx context.Context, sourceID, attemptID int64, in *PublishReconcileInput) (*PublishAttempt, error) {
	if in == nil {
		return nil, ErrInvalidWorkflow
	}
	link, err := findOwnedTaskLink(s.db, sourceID, in.OwnerID, 0)
	if err != nil {
		return nil, err
	}
	return s.reconcilePublishForTask(ctx, sourceID, link.ID, attemptID, in)
}

func (s *Service) reconcilePublishForTask(ctx context.Context, sourceID, taskLinkID, attemptID int64, in *PublishReconcileInput) (*PublishAttempt, error) {
	if in == nil || in.OwnerID <= 0 || (in.Outcome != PublishStatusSubmitted && in.Outcome != PublishStatusFailed) || strings.TrimSpace(in.EvidenceURI) == "" || in.ObservedAt.IsZero() || in.ObservedAt.After(time.Now().Add(5*time.Minute)) || in.TruthStatus != "actual" {
		return nil, fmt.Errorf("%w: actual reconciliation evidence, observation time and submitted/failed outcome are required", ErrInvalidWorkflow)
	}
	if in.Outcome == PublishStatusSubmitted && strings.TrimSpace(in.PlatformResult.PlatformProductID) == "" {
		return nil, fmt.Errorf("%w: submitted reconciliation requires a platform product identifier", ErrInvalidWorkflow)
	}
	payload, err := json.Marshal(reconciledPublishEvidence{PlatformResult: in.PlatformResult, EvidenceURI: strings.TrimSpace(in.EvidenceURI), ObservedAt: in.ObservedAt.UTC(), TruthStatus: in.TruthStatus})
	if err != nil {
		return nil, fmt.Errorf("%w: reconciliation result is not serializable", ErrInvalidWorkflow)
	}
	hash := sha256.Sum256(payload)
	evidenceHash := sha256.Sum256([]byte(strings.TrimSpace(in.EvidenceURI)))
	var attempt PublishAttempt
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	err = s.db.WithContext(persistCtx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND sourcing_product_id = ?", attemptID, sourceID).First(&attempt).Error; err != nil {
			return err
		}
		if attempt.Status != PublishStatusReconcile && attempt.Status != PublishStatusExecuting {
			return fmt.Errorf("%w: only an ambiguous publish attempt may be reconciled", ErrWorkflowGate)
		}
		if in.TaskLinkID > 0 && in.TaskLinkID != attempt.TaskLinkID {
			return fmt.Errorf("%w: reconciliation belongs to another task", ErrWorkflowGate)
		}
		task, err := requireTaskSourcingAuthority(tx, sourceID, in.OwnerID, attempt.TaskLinkID)
		if err != nil {
			return err
		}
		if attempt.Status == PublishStatusExecuting && (attempt.ExecutedAt == nil || attempt.ExecutedAt.After(time.Now().UTC().Add(-35*time.Second))) {
			return fmt.Errorf("%w: active publish call cannot be reconciled before its timeout window", ErrWorkflowGate)
		}
		var source sourcingLifecycleRow
		if err := tx.First(&source, sourceID).Error; err != nil {
			return err
		}
		if err := requireOwner(tx, &source, in.OwnerID); err != nil {
			return err
		}
		if task.ExperimentID != attempt.ExperimentID {
			return fmt.Errorf("%w: task authority trace does not match this reconciliation", ErrWorkflowGate)
		}
		if err := requireCurrentCompliance(tx, sourceID, task.ID, in.OwnerID, StandardPublishComplianceRequirementCodes, time.Now().UTC()); err != nil {
			return err
		}
		completedAt := time.Now().UTC()
		updates := map[string]any{"status": in.Outcome, "response_payload": payload, "response_sha256": hex.EncodeToString(hash[:]), "error_message": "", "completed_at": completedAt}
		if in.Outcome == PublishStatusFailed {
			updates["error_message"] = "platform_reconciled_failed"
		}
		updated := tx.Model(&PublishAttempt{}).Where("id = ? AND status IN ?", attempt.ID, []string{PublishStatusExecuting, PublishStatusReconcile}).Updates(updates)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return fmt.Errorf("%w: publish attempt was concurrently reconciled", ErrWorkflowGate)
		}
		if in.Outcome == PublishStatusSubmitted {
			listingUpdate := tx.Model(&listingRow{}).Where("id = ? AND status = ?", attempt.ListingID, "draft").Updates(map[string]any{"status": "submitted", "published_data": payload, "platform_product_id": in.PlatformResult.PlatformProductID, "platform_url": in.PlatformResult.PlatformURL, "sync_message": in.PlatformResult.SyncMessage, "last_sync_at": completedAt})
			if listingUpdate.Error != nil {
				return listingUpdate.Error
			}
			if listingUpdate.RowsAffected != 1 {
				return fmt.Errorf("%w: linked draft changed before reconciliation", ErrWorkflowGate)
			}
		}
		taskStatus := "publish_failed"
		if in.Outcome == PublishStatusSubmitted {
			taskStatus = "submitted"
		}
		if err := tx.Model(task).Updates(map[string]any{"workflow_status": taskStatus, "workflow_updated_at": completedAt}).Error; err != nil {
			return err
		}
		if err := tx.Create(&operationlog.OperationLog{Module: "sourcing1688", Action: "publish.reconcile", ResourceID: strconv.FormatInt(attempt.ID, 10), Operator: strconv.FormatInt(in.OwnerID, 10), UserID: in.OwnerID, Content: fmt.Sprintf("publish_attempt=%d evidence_sha256=%s", attempt.ID, hex.EncodeToString(evidenceHash[:])), Result: in.Outcome, TriggerType: "owner_approval", ApprovalID: attempt.ApprovalID, EntityType: PublishApprovalTargetType, EntityID: attempt.ID}).Error; err != nil {
			return err
		}
		return tx.First(&attempt, attempt.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &attempt, nil
}
