package sourcing1688

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PublishTerminalSourcePlatformReceipt          = "platform_receipt"
	PublishTerminalSourceControlledReconciliation = "controlled_reconciliation"
)

// PublishTerminalObservationInput records an externally observed terminal
// platform state. ReceiptPayload is evidence, not an instruction to call the
// platform. It is hashed byte-for-byte and cannot be changed after insertion.
type PublishTerminalObservationInput struct {
	OwnerID           int64           `json:"owner_id,omitempty"`
	TaskLinkID        int64           `json:"task_link_id,omitempty"`
	Outcome           string          `json:"outcome" binding:"required"`
	SourceType        string          `json:"source_type" binding:"required"`
	EvidenceID        string          `json:"evidence_id" binding:"required"`
	ExternalReceiptID string          `json:"external_receipt_id" binding:"required"`
	ObservedAt        time.Time       `json:"observed_at" binding:"required"`
	PlatformProductID string          `json:"platform_product_id,omitempty"`
	FailureCode       string          `json:"failure_code,omitempty"`
	FailureMessage    string          `json:"failure_message,omitempty"`
	ReceiptPayload    json.RawMessage `json:"receipt_payload" binding:"required"`
}

type PublishTerminalEvidence struct {
	ID                int64           `json:"id"`
	OwnerID           int64           `json:"owner_id"`
	SourcingProductID int64           `json:"sourcing_product_id"`
	TaskLinkID        int64           `json:"task_link_id"`
	PublishAttemptID  int64           `json:"publish_attempt_id"`
	PlatformID        int64           `json:"platform_id"`
	PlatformAccountID int64           `json:"platform_account_id"`
	Outcome           string          `json:"outcome"`
	SourceType        string          `json:"source_type"`
	EvidenceID        string          `json:"evidence_id"`
	ExternalReceiptID string          `json:"external_receipt_id"`
	ObservedAt        time.Time       `json:"observed_at"`
	PlatformProductID string          `json:"platform_product_id,omitempty"`
	FailureCode       string          `json:"failure_code,omitempty"`
	FailureMessage    string          `json:"failure_message,omitempty"`
	ReceiptPayload    json.RawMessage `json:"receipt_payload"`
	ReceiptSHA256     string          `json:"receipt_sha256"`
	RecordedAt        time.Time       `json:"recorded_at"`
}

func (PublishTerminalEvidence) TableName() string { return "sourcing_publish_terminal_evidence" }

func validateTerminalObservation(in *PublishTerminalObservationInput) error {
	if in == nil || in.OwnerID <= 0 || in.TaskLinkID <= 0 ||
		(in.Outcome != PublishStatusSucceeded && in.Outcome != PublishStatusFailed) ||
		(in.SourceType != PublishTerminalSourcePlatformReceipt && in.SourceType != PublishTerminalSourceControlledReconciliation) ||
		strings.TrimSpace(in.EvidenceID) == "" || strings.TrimSpace(in.ExternalReceiptID) == "" ||
		in.ObservedAt.IsZero() || in.ObservedAt.After(time.Now().UTC().Add(5*time.Minute)) ||
		len(in.ReceiptPayload) == 0 || !json.Valid(in.ReceiptPayload) {
		return fmt.Errorf("%w: valid terminal outcome, source, immutable receipt identity, observation time and JSON receipt are required", ErrInvalidWorkflow)
	}
	if len(in.EvidenceID) > 200 || len(in.ExternalReceiptID) > 200 || len(in.ReceiptPayload) > 1024*1024 {
		return fmt.Errorf("%w: terminal receipt exceeds allowed size", ErrInvalidWorkflow)
	}
	if in.Outcome == PublishStatusSucceeded && strings.TrimSpace(in.PlatformProductID) == "" {
		return fmt.Errorf("%w: succeeded receipt requires the platform product identifier", ErrInvalidWorkflow)
	}
	if in.Outcome == PublishStatusFailed && (strings.TrimSpace(in.FailureCode) == "" || strings.TrimSpace(in.FailureMessage) == "") {
		return fmt.Errorf("%w: failed receipt requires an explainable failure code and message", ErrInvalidWorkflow)
	}
	return nil
}

// ObserveTaskPublishTerminal is side-effect free with respect to the external
// platform. It may only close an already submitted or ambiguous exact-task
// attempt from immutable platform evidence or a controlled reconciliation.
func (s *Service) ObserveTaskPublishTerminal(ctx context.Context, sourceID, taskLinkID, attemptID int64, in *PublishTerminalObservationInput) (*PublishTerminalEvidence, error) {
	if in == nil {
		return nil, ErrInvalidWorkflow
	}
	in.TaskLinkID = taskLinkID
	if err := validateTerminalObservation(in); err != nil {
		return nil, err
	}
	receiptHash := sha256.Sum256(in.ReceiptPayload)
	now := time.Now().UTC()
	evidence := PublishTerminalEvidence{
		OwnerID: in.OwnerID, SourcingProductID: sourceID, TaskLinkID: taskLinkID, PublishAttemptID: attemptID,
		Outcome: in.Outcome, SourceType: in.SourceType, EvidenceID: strings.TrimSpace(in.EvidenceID),
		ExternalReceiptID: strings.TrimSpace(in.ExternalReceiptID), ObservedAt: in.ObservedAt.UTC(),
		PlatformProductID: strings.TrimSpace(in.PlatformProductID), FailureCode: strings.TrimSpace(in.FailureCode),
		FailureMessage: strings.TrimSpace(in.FailureMessage), ReceiptPayload: append(json.RawMessage(nil), in.ReceiptPayload...),
		ReceiptSHA256: hex.EncodeToString(receiptHash[:]), RecordedAt: now,
	}
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	err := s.db.WithContext(persistCtx).Transaction(func(tx *gorm.DB) error {
		var attempt PublishAttempt
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND sourcing_product_id = ? AND task_link_id = ?", attemptID, sourceID, taskLinkID).First(&attempt).Error; err != nil {
			return fmt.Errorf("%w: publish attempt belongs to another Owner or task", ErrWorkflowGate)
		}
		task, err := requireTaskSourcingAuthority(tx, sourceID, in.OwnerID, taskLinkID)
		if err != nil {
			return err
		}
		if attempt.PlatformID <= 0 || attempt.PlatformAccountID <= 0 || attempt.ExperimentID != task.ExperimentID {
			return fmt.Errorf("%w: publish attempt authority does not match the task", ErrWorkflowGate)
		}
		if err := requireCurrentCompliance(tx, sourceID, task.ID, in.OwnerID, StandardPublishComplianceRequirementCodes, time.Now().UTC()); err != nil {
			return err
		}
		evidence.PlatformID, evidence.PlatformAccountID = attempt.PlatformID, attempt.PlatformAccountID

		var replay PublishTerminalEvidence
		if err := tx.Where("owner_id = ? AND platform_id = ? AND evidence_id = ?", in.OwnerID, attempt.PlatformID, evidence.EvidenceID).First(&replay).Error; err == nil {
			if replay.PublishAttemptID != attemptID || replay.TaskLinkID != taskLinkID || replay.Outcome != evidence.Outcome || replay.SourceType != evidence.SourceType || replay.ExternalReceiptID != evidence.ExternalReceiptID || replay.ObservedAt.UTC() != evidence.ObservedAt.UTC() || replay.PlatformProductID != evidence.PlatformProductID || replay.FailureCode != evidence.FailureCode || replay.FailureMessage != evidence.FailureMessage || replay.ReceiptSHA256 != evidence.ReceiptSHA256 {
				return fmt.Errorf("%w: evidence id is already bound to different terminal facts", ErrWorkflowGate)
			}
			evidence = replay
			return nil
		} else if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if attempt.Status != PublishStatusSubmitted && attempt.Status != PublishStatusReconcile {
			return fmt.Errorf("%w: only submitted or reconciliation-required attempts may receive a terminal observation", ErrWorkflowGate)
		}
		if evidence.Outcome == PublishStatusSucceeded && attempt.ResponsePayload != nil {
			var direct struct {
				PlatformProductID string `json:"platform_product_id"`
			}
			var reconciled reconciledPublishEvidence
			_ = json.Unmarshal(attempt.ResponsePayload, &direct)
			_ = json.Unmarshal(attempt.ResponsePayload, &reconciled)
			known := strings.TrimSpace(direct.PlatformProductID)
			if known == "" {
				known = strings.TrimSpace(reconciled.PlatformResult.PlatformProductID)
			}
			if known != "" && known != evidence.PlatformProductID {
				return fmt.Errorf("%w: terminal receipt platform product differs from submitted result", ErrWorkflowGate)
			}
		}
		if err := tx.Create(&evidence).Error; err != nil {
			return err
		}
		updated := tx.Model(&PublishAttempt{}).Where("id = ? AND status IN ?", attemptID, []string{PublishStatusSubmitted, PublishStatusReconcile}).Updates(map[string]any{"status": evidence.Outcome, "error_message": evidence.FailureCode, "completed_at": now})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return fmt.Errorf("%w: publish attempt was concurrently closed", ErrWorkflowGate)
		}
		if err := tx.Model(task).Updates(map[string]any{"workflow_status": evidence.Outcome, "workflow_updated_at": now, "blocked_reason": evidence.FailureMessage}).Error; err != nil {
			return err
		}
		listingUpdates := map[string]any{"status": evidence.Outcome, "last_sync_at": evidence.ObservedAt, "sync_message": evidence.FailureMessage}
		if evidence.Outcome == PublishStatusSucceeded {
			listingUpdates["platform_product_id"] = evidence.PlatformProductID
			listingUpdates["sync_message"] = "platform_terminal_succeeded"
		}
		if err := tx.Model(&listingRow{}).Where("id = ? AND product_id = ? AND platform_id = ?", attempt.ListingID, attempt.ProductID, attempt.PlatformID).Updates(listingUpdates).Error; err != nil {
			return err
		}
		return tx.Create(&operationlog.OperationLog{Module: "sourcing1688", Action: "publish.terminal.observe", ResourceID: strconv.FormatInt(attemptID, 10), Operator: strconv.FormatInt(in.OwnerID, 10), UserID: in.OwnerID, Content: fmt.Sprintf("evidence_id=%s receipt_sha256=%s source=%s", evidence.EvidenceID, evidence.ReceiptSHA256, evidence.SourceType), Result: evidence.Outcome, TriggerType: "external_observation", ApprovalID: attempt.ApprovalID, EntityType: PublishApprovalTargetType, EntityID: attemptID}).Error
	})
	if err != nil {
		return nil, err
	}
	return &evidence, nil
}
