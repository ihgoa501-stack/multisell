package sourcing1688

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/lingmirror/backend-go/internal/domain/approval"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type draftApprovalNewValue struct {
	ContentSHA256 string `json:"content_sha256"`
}

type canonicalDraftContent struct {
	Draft struct {
		ID, SourcingProductID, SnapshotID, ProductID, ListingID, DemandCaseID int64
		ExperimentID                                                          string
	}
	Product productRow
	Listing listingRow
	SKUs    []skuRow
	Media   []mediaRow
	Costs   []costRow
}

// calculateDraftContentSHA256 produces the canonical approval fingerprint for
// all mutable product/listing/SKU/media/cost fields represented by a draft.
// Stable ID ordering makes the same persisted content hash identically.
func calculateDraftContentSHA256(tx *gorm.DB, draft *draftRow) (string, error) {
	return calculateDraftContentSHA256WithLock(tx, draft, false)
}

func calculateDraftContentSHA256Locked(tx *gorm.DB, draft *draftRow) (string, error) {
	return calculateDraftContentSHA256WithLock(tx, draft, true)
}

func calculateDraftContentSHA256WithLock(tx *gorm.DB, draft *draftRow, lock bool) (string, error) {
	if draft == nil || draft.ID <= 0 {
		return "", fmt.Errorf("%w: draft is required", ErrWorkflowGate)
	}
	var content canonicalDraftContent
	content.Draft.ID = draft.ID
	content.Draft.SourcingProductID = draft.SourcingProductID
	content.Draft.SnapshotID = draft.SnapshotID
	content.Draft.ProductID = draft.ProductID
	content.Draft.ListingID = draft.ListingID
	content.Draft.DemandCaseID = draft.DemandCaseID
	content.Draft.ExperimentID = draft.ExperimentID
	query := func() *gorm.DB {
		if lock {
			return tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		return tx
	}
	if err := query().First(&content.Product, draft.ProductID).Error; err != nil {
		return "", err
	}
	if err := query().First(&content.Listing, draft.ListingID).Error; err != nil {
		return "", err
	}
	if content.Product.ID != draft.ProductID || content.Listing.ProductID != draft.ProductID {
		return "", fmt.Errorf("%w: draft content linkage is invalid", ErrWorkflowGate)
	}
	if err := query().Where("product_id = ?", draft.ProductID).Order("id").Find(&content.SKUs).Error; err != nil {
		return "", err
	}
	if err := query().Where("product_id = ?", draft.ProductID).Order("id").Find(&content.Media).Error; err != nil {
		return "", err
	}
	if err := query().Where("product_id = ? AND experiment_id = ?", draft.ProductID, draft.ExperimentID).Order("id").Find(&content.Costs).Error; err != nil {
		return "", err
	}
	payload, err := json.Marshal(content)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func marshalDraftApprovalNewValue(hash string) (string, error) {
	payload, err := json.Marshal(draftApprovalNewValue{ContentSHA256: hash})
	return string(payload), err
}

func approvalContentHash(req *approval.ApprovalRequest) string {
	if req == nil {
		return ""
	}
	var value draftApprovalNewValue
	if json.Unmarshal([]byte(req.NewValue), &value) != nil {
		return ""
	}
	return value.ContentSHA256
}

func validateDraftApprovalContent(tx *gorm.DB, draft *draftRow, req *approval.ApprovalRequest) error {
	current, err := calculateDraftContentSHA256(tx, draft)
	if err != nil {
		return err
	}
	approvedHash := approvalContentHash(req)
	if len(current) != 64 || draft.ApprovalContentSHA256 != current || approvedHash != current {
		return fmt.Errorf("%w: draft content changed after approval submission", ErrWorkflowGate)
	}
	return nil
}

func validateDraftApprovalContentLocked(tx *gorm.DB, draft *draftRow, req *approval.ApprovalRequest) error {
	current, err := calculateDraftContentSHA256Locked(tx, draft)
	if err != nil {
		return err
	}
	approvedHash := approvalContentHash(req)
	if len(current) != 64 || draft.ApprovalContentSHA256 != current || approvedHash != current {
		return fmt.Errorf("%w: draft content changed after approval submission", ErrWorkflowGate)
	}
	return nil
}
