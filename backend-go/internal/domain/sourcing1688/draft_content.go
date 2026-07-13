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
	ContentSHA256          string `json:"content_sha256"`
	CostVersionID          int64  `json:"cost_version_id,omitempty"`
	CostVersionContentHash string `json:"cost_version_content_hash,omitempty"`
}

type canonicalDraftContent struct {
	Draft struct {
		ID, SourcingProductID, SnapshotID, ProductID, ListingID, DemandCaseID int64
		ExperimentID                                                          string
		CostVersionID                                                         *int64
		CostVersionContentHash                                                string
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
	content.Draft.CostVersionID = draft.CostVersionID
	content.Draft.CostVersionContentHash = draft.CostVersionContentHash
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

func marshalDraftApprovalNewValue(hash string, draft ...*draftRow) (string, error) {
	value := draftApprovalNewValue{ContentSHA256: hash}
	if len(draft) > 0 && draft[0] != nil && draft[0].CostVersionID != nil {
		value.CostVersionID, value.CostVersionContentHash = *draft[0].CostVersionID, draft[0].CostVersionContentHash
	}
	payload, err := json.Marshal(value)
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
	if err := validateFrozenDraftCost(tx, draft, req); err != nil {
		return err
	}
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
	if err := validateFrozenDraftCost(tx, draft, req); err != nil {
		return err
	}
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

func validateFrozenDraftCost(tx *gorm.DB, draft *draftRow, req *approval.ApprovalRequest) error {
	if draft.CostVersionID == nil {
		return nil
	} // compatibility for drafts created before migration 000149
	var version SourcingCostVersion
	if err := tx.Where("id = ? AND sourcing_product_id = ? AND task_link_id = ?", *draft.CostVersionID, draft.SourcingProductID, draftTaskLinkID(draft)).First(&version).Error; err != nil {
		return fmt.Errorf("%w: frozen precise cost version is missing", ErrWorkflowGate)
	}
	var value draftApprovalNewValue
	if req == nil || json.Unmarshal([]byte(req.NewValue), &value) != nil || value.CostVersionID != version.ID || value.CostVersionContentHash != version.ContentHash || draft.CostVersionContentHash != version.ContentHash {
		return fmt.Errorf("%w: precise cost version changed after approval submission", ErrWorkflowGate)
	}
	return nil
}

func draftTaskLinkID(draft *draftRow) int64 {
	if draft != nil && draft.TaskLinkID != nil {
		return *draft.TaskLinkID
	}
	return 0
}
