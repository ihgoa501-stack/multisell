package sourcing1688

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PrivateRequestReceiving         = "receiving"
	PrivateRequestSaved             = "saved"
	PrivateRequestNotSaved          = "not_saved"
	PrivateRequestReconcileRequired = "reconcile_required"
)

// PrivateCollectionRequest is the durable receipt for one extension request.
// It intentionally stores no page payload, HTML, credentials, title or supplier
// data; those belong to the immutable snapshot only after a successful save.
type PrivateCollectionRequest struct {
	ID                    int64      `gorm:"column:id;primaryKey;autoIncrement" json:"-"`
	OwnerID               int64      `gorm:"column:owner_id;not null;uniqueIndex:ux_private_collection_request,priority:1" json:"-"`
	RequestID             string     `gorm:"column:request_id;size:80;not null;uniqueIndex:ux_private_collection_request,priority:2" json:"request_id"`
	Status                string     `gorm:"column:status;size:32;not null" json:"status"`
	RequestEnvelopeSHA256 string     `gorm:"column:request_envelope_sha256;size:64;not null;default:''" json:"-"`
	FailureCode           string     `gorm:"column:failure_code;size:80;not null;default:''" json:"failure_code,omitempty"`
	SafeMessage           string     `gorm:"column:safe_message;size:500;not null;default:''" json:"safe_message,omitempty"`
	RecordID              *int64     `gorm:"column:record_id" json:"record_id,omitempty"`
	SnapshotID            *int64     `gorm:"column:snapshot_id" json:"snapshot_id,omitempty"`
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	CompletedAt           *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
}

func (PrivateCollectionRequest) TableName() string {
	return "sourcing_1688_private_collection_request"
}

func privateRequestSafeFailure(err error) (string, string) {
	var duplicate *DuplicatePrivateCollectionError
	if errors.As(err, &duplicate) {
		return "duplicate_requires_choice", "该1688商品已在私人采集箱，本次未保存；请选择查看已有记录或明确保存为新观察"
	}
	if errors.Is(err, ErrInvalidWorkflow) || errors.Is(err, ErrWorkflowGate) {
		return PrivateFailureInvalidPayload, privateFailureMessages[PrivateFailureInvalidPayload]
	}
	return "persistence_outcome_unknown", "服务器未能确认采集是否保存，请勿重复采集并继续对账"
}

func (s *Service) beginPrivateCollectionRequest(ownerID int64, requestID, envelopeHash string) (*PrivateCollectionRequest, error) {
	requestID = strings.TrimSpace(requestID)
	if ownerID <= 0 || !strings.HasPrefix(requestID, "collect_") || len(requestID) > 80 {
		return nil, ErrInvalidWorkflow
	}
	record := PrivateCollectionRequest{OwnerID: ownerID, RequestID: requestID, Status: PrivateRequestReceiving, RequestEnvelopeSHA256: envelopeHash}
	result := s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "owner_id"}, {Name: "request_id"}}, DoNothing: true}).Create(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 1 {
		return &record, nil
	}
	if err := s.db.Where("owner_id = ? AND request_id = ?", ownerID, requestID).First(&record).Error; err != nil {
		return nil, err
	}
	if record.RequestEnvelopeSHA256 != "" && envelopeHash != "" && record.RequestEnvelopeSHA256 != envelopeHash {
		return nil, fmt.Errorf("%w: collection request payload conflict", ErrInvalidWorkflow)
	}
	if record.Status == PrivateRequestNotSaved {
		if record.FailureCode == "duplicate_requires_choice" && record.RecordID != nil {
			if record.SnapshotID != nil {
				var snapshot Sourcing1688Snapshot
				if err := s.db.Where("id = ? AND sourcing_product_id = ? AND collected_by = ?", *record.SnapshotID, *record.RecordID, ownerID).First(&snapshot).Error; err != nil {
					return nil, err
				}
				return nil, &DuplicatePrivateCollectionError{RecordID: *record.RecordID, SnapshotID: *record.SnapshotID, Existing: duplicateSummary(&snapshot)}
			}
			var product Sourcing1688Product
			if err := s.db.Where("id = ? AND owner_id = ?", *record.RecordID, ownerID).First(&product).Error; err != nil {
				return nil, err
			}
			return nil, &DuplicatePrivateCollectionError{RecordID: *record.RecordID, Existing: duplicateSummaryFromProduct(&product)}
		}
		return nil, fmt.Errorf("%w: collection request already completed as not_saved", ErrInvalidWorkflow)
	}
	return &record, nil
}

func markPrivateRequest(tx *gorm.DB, ownerID int64, requestID, status, failureCode, safeMessage string, recordID, snapshotID *int64) error {
	now := time.Now().UTC()
	updates := map[string]any{"status": status, "failure_code": failureCode, "safe_message": safeMessage, "record_id": recordID, "snapshot_id": snapshotID, "updated_at": now}
	if status != PrivateRequestReceiving {
		updates["completed_at"] = now
	}
	result := tx.Model(&PrivateCollectionRequest{}).Where("owner_id = ? AND request_id = ?", ownerID, strings.TrimSpace(requestID)).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("private collection request receipt missing")
	}
	return nil
}

func (s *Service) markPrivateRequestFailure(in *PrivateCollectInput, err error) {
	if in == nil || in.OwnerID <= 0 || !strings.HasPrefix(strings.TrimSpace(in.RequestID), "collect_") {
		return
	}
	code, message := privateRequestSafeFailure(err)
	status := PrivateRequestNotSaved
	if code == "persistence_outcome_unknown" {
		status = PrivateRequestReconcileRequired
	}
	// Failure-state persistence is best effort so it never hides the original
	// collection error returned to the extension.
	var recordID, snapshotID *int64
	var duplicate *DuplicatePrivateCollectionError
	if errors.As(err, &duplicate) {
		recordID = &duplicate.RecordID
		if duplicate.SnapshotID > 0 {
			snapshotID = &duplicate.SnapshotID
		}
	}
	_ = markPrivateRequest(s.db, in.OwnerID, in.RequestID, status, code, message, recordID, snapshotID)
}

func (s *Service) GetPrivateCollectionRequest(ownerID int64, requestID string) (*PrivateCollectHTTPResult, error) {
	requestID = strings.TrimSpace(requestID)
	if ownerID <= 0 || !strings.HasPrefix(requestID, "collect_") || len(requestID) > 80 {
		return nil, ErrInvalidWorkflow
	}
	var receipt PrivateCollectionRequest
	if err := s.db.Where("owner_id = ? AND request_id = ?", ownerID, requestID).First(&receipt).Error; err != nil {
		return nil, err
	}
	result := &PrivateCollectHTTPResult{Status: receipt.Status, RequestID: receipt.RequestID, FailureCode: receipt.FailureCode, SafeMessage: receipt.SafeMessage}
	if receipt.RecordID != nil {
		result.RecordID = *receipt.RecordID
	}
	if receipt.SnapshotID != nil {
		result.SnapshotID = *receipt.SnapshotID
	}
	if receipt.Status == PrivateRequestSaved {
		result.IdempotentReplay = true
	}
	return result, nil
}
