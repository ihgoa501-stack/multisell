package sourcing1688

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SourcingSKUMapping is the immutable authority for the supplier -> internal
// -> channel SKU identity chain. Embedded listing JSON is deliberately not the
// authority because it cannot enforce Owner scope or relational identity.
type SourcingSKUMapping struct {
	ID                   int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID              int64     `gorm:"column:owner_id;not null" json:"owner_id"`
	SourcingProductID    int64     `gorm:"column:sourcing_product_id;not null" json:"sourcing_product_id"`
	TaskLinkID           int64     `gorm:"column:task_link_id;not null" json:"task_link_id"`
	SnapshotID           int64     `gorm:"column:snapshot_id;not null" json:"snapshot_id"`
	ProductOpportunityID int64     `gorm:"column:product_opportunity_id;not null" json:"product_opportunity_id"`
	ProductID            int64     `gorm:"column:product_id;not null" json:"product_id"`
	SupplierSKU          string    `gorm:"column:supplier_sku;size:240;not null" json:"supplier_sku"`
	InternalSKUID        int64     `gorm:"column:internal_sku_id;not null" json:"internal_sku_id"`
	InternalSKU          string    `gorm:"column:internal_sku;size:240;not null" json:"internal_sku"`
	ChannelSKU           string    `gorm:"column:channel_sku;size:240;not null" json:"channel_sku"`
	PlatformID           int64     `gorm:"column:platform_id;not null" json:"platform_id"`
	ListingID            int64     `gorm:"column:listing_id;not null" json:"listing_id"`
	Version              int64     `gorm:"column:version;not null" json:"version"`
	ContentHash          string    `gorm:"column:content_hash;size:64;not null" json:"content_hash"`
	CreatedBy            int64     `gorm:"column:created_by;not null" json:"created_by"`
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SourcingSKUMapping) TableName() string { return "sourcing_sku_mapping" }

// ListCanonicalSKUMappings returns only mappings frozen for the exact
// Owner/source/task authority. It is the read seam used by the Owner UI; IDs
// must never be guessed or copied from another sourcing task.
func (s *Service) ListCanonicalSKUMappings(ownerID, sourceID, taskLinkID int64) ([]SourcingSKUMapping, error) {
	if ownerID <= 0 || sourceID <= 0 || taskLinkID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	if _, err := requireTaskSourcingAuthority(s.db, sourceID, ownerID, taskLinkID); err != nil {
		return nil, err
	}
	var rows []SourcingSKUMapping
	err := s.db.Where("owner_id = ? AND sourcing_product_id = ? AND task_link_id = ?", ownerID, sourceID, taskLinkID).
		Order("version DESC, supplier_sku ASC").Find(&rows).Error
	return rows, err
}

type CanonicalSKUMappingItem struct {
	SupplierSKU   string
	InternalSKUID int64
	InternalSKU   string
	ChannelSKU    string
}

// CanonicalSKUMappingInput binds a complete mapping set to one frozen sourcing
// observation, approved opportunity, internal product and channel listing.
type CanonicalSKUMappingInput struct {
	OwnerID              int64
	SourcingProductID    int64
	TaskLinkID           int64
	SnapshotID           int64
	ProductOpportunityID int64
	ProductID            int64
	PlatformID           int64
	ListingID            int64
	Version              int64
	CreatedBy            int64
	Items                []CanonicalSKUMappingItem
}

type skuMappingProductRow struct{ ID, OwnerID int64 }

func (skuMappingProductRow) TableName() string { return "sourcing_1688_product" }

type skuMappingSnapshotRow struct{ ID, SourcingProductID int64 }

func (skuMappingSnapshotRow) TableName() string { return "sourcing_1688_snapshot" }

type skuMappingTaskRow struct {
	ID, OwnerID, SourcingProductID int64
	ProductOpportunityID           *int64
}

func (skuMappingTaskRow) TableName() string { return "sourcing_1688_task_link" }

type skuMappingOpportunityRow struct{ ID, OwnerID int64 }

func (skuMappingOpportunityRow) TableName() string { return "product_opportunity" }

type skuMappingInternalRow struct {
	ID, ProductID int64
	Code          string
}

func (skuMappingInternalRow) TableName() string { return "sku" }

type skuMappingListingRow struct{ ID, ProductID, PlatformID int64 }

func (skuMappingListingRow) TableName() string { return "product_listing" }

// PersistCanonicalSKUMappings validates every relational identity again inside
// the caller transaction, then inserts an immutable version. An exact replay
// returns existing rows; the same version with different content fails closed.
func PersistCanonicalSKUMappings(tx *gorm.DB, in CanonicalSKUMappingInput) ([]SourcingSKUMapping, error) {
	if tx == nil || in.OwnerID <= 0 || in.CreatedBy != in.OwnerID || in.SourcingProductID <= 0 || in.TaskLinkID <= 0 || in.SnapshotID <= 0 || in.ProductOpportunityID <= 0 || in.ProductID <= 0 || in.PlatformID <= 0 || in.ListingID <= 0 || in.Version <= 0 || len(in.Items) == 0 {
		return nil, fmt.Errorf("%w: complete Owner-scoped SKU mapping context required", ErrInvalidWorkflow)
	}
	items := append([]CanonicalSKUMappingItem(nil), in.Items...)
	for i := range items {
		items[i].SupplierSKU = strings.TrimSpace(items[i].SupplierSKU)
		items[i].InternalSKU = strings.TrimSpace(items[i].InternalSKU)
		items[i].ChannelSKU = strings.TrimSpace(items[i].ChannelSKU)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SupplierSKU != items[j].SupplierSKU {
			return items[i].SupplierSKU < items[j].SupplierSKU
		}
		if items[i].InternalSKU != items[j].InternalSKU {
			return items[i].InternalSKU < items[j].InternalSKU
		}
		return items[i].ChannelSKU < items[j].ChannelSKU
	})
	seenSupplier, seenInternal, seenChannel := map[string]bool{}, map[int64]bool{}, map[string]bool{}
	for _, item := range items {
		if item.SupplierSKU == "" || item.InternalSKUID <= 0 || item.InternalSKU == "" || item.ChannelSKU == "" || seenSupplier[item.SupplierSKU] || seenInternal[item.InternalSKUID] || seenChannel[item.ChannelSKU] {
			return nil, fmt.Errorf("%w: SKU mappings must be non-empty and one-to-one", ErrInvalidWorkflow)
		}
		seenSupplier[item.SupplierSKU], seenInternal[item.InternalSKUID], seenChannel[item.ChannelSKU] = true, true, true
	}

	if err := validateCanonicalSKUMappingAuthority(tx, in, items); err != nil {
		return nil, err
	}
	rows := make([]SourcingSKUMapping, 0, len(items))
	for _, item := range items {
		row := SourcingSKUMapping{OwnerID: in.OwnerID, SourcingProductID: in.SourcingProductID, TaskLinkID: in.TaskLinkID, SnapshotID: in.SnapshotID, ProductOpportunityID: in.ProductOpportunityID, ProductID: in.ProductID, SupplierSKU: item.SupplierSKU, InternalSKUID: item.InternalSKUID, InternalSKU: item.InternalSKU, ChannelSKU: item.ChannelSKU, PlatformID: in.PlatformID, ListingID: in.ListingID, Version: in.Version, CreatedBy: in.CreatedBy}
		row.ContentHash = canonicalSKUMappingHash(row)
		rows = append(rows, row)
	}
	var existing []SourcingSKUMapping
	if err := tx.Where("owner_id = ? AND task_link_id = ? AND version = ?", in.OwnerID, in.TaskLinkID, in.Version).Order("supplier_sku ASC").Find(&existing).Error; err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		if len(existing) != len(rows) {
			return nil, fmt.Errorf("%w: SKU mapping version is already frozen with different content", ErrWorkflowGate)
		}
		for i := range rows {
			if existing[i].ContentHash != rows[i].ContentHash {
				return nil, fmt.Errorf("%w: SKU mapping version is already frozen with different content", ErrWorkflowGate)
			}
		}
		return existing, nil
	}
	if err := tx.Create(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func validateCanonicalSKUMappingAuthority(tx *gorm.DB, in CanonicalSKUMappingInput, items []CanonicalSKUMappingItem) error {
	var source skuMappingProductRow
	if err := tx.First(&source, in.SourcingProductID).Error; err != nil || source.OwnerID != in.OwnerID {
		return fmt.Errorf("%w: sourcing product Owner mismatch", ErrWorkflowGate)
	}
	var snapshot skuMappingSnapshotRow
	if err := tx.First(&snapshot, in.SnapshotID).Error; err != nil || snapshot.SourcingProductID != in.SourcingProductID {
		return fmt.Errorf("%w: source snapshot mismatch", ErrWorkflowGate)
	}
	var task skuMappingTaskRow
	if err := tx.First(&task, in.TaskLinkID).Error; err != nil || task.OwnerID != in.OwnerID || task.SourcingProductID != in.SourcingProductID || task.ProductOpportunityID == nil || *task.ProductOpportunityID != in.ProductOpportunityID {
		return fmt.Errorf("%w: sourcing task authority mismatch", ErrWorkflowGate)
	}
	var opportunity skuMappingOpportunityRow
	if err := tx.First(&opportunity, in.ProductOpportunityID).Error; err != nil || opportunity.OwnerID != in.OwnerID {
		return fmt.Errorf("%w: product opportunity Owner mismatch", ErrWorkflowGate)
	}
	var listing skuMappingListingRow
	if err := tx.First(&listing, in.ListingID).Error; err != nil || listing.ProductID != in.ProductID || listing.PlatformID != in.PlatformID {
		return fmt.Errorf("%w: channel listing identity mismatch", ErrWorkflowGate)
	}
	for _, item := range items {
		var internal skuMappingInternalRow
		if err := tx.First(&internal, item.InternalSKUID).Error; err != nil || internal.ProductID != in.ProductID || strings.TrimSpace(internal.Code) != item.InternalSKU {
			return fmt.Errorf("%w: internal SKU identity mismatch", ErrWorkflowGate)
		}
	}
	return nil
}

func canonicalSKUMappingHash(row SourcingSKUMapping) string {
	payload, _ := json.Marshal(struct {
		OwnerID, SourcingProductID, TaskLinkID, SnapshotID, ProductOpportunityID, ProductID int64
		SupplierSKU                                                                         string
		InternalSKUID                                                                       int64
		InternalSKU, ChannelSKU                                                             string
		PlatformID, ListingID, Version                                                      int64
	}{row.OwnerID, row.SourcingProductID, row.TaskLinkID, row.SnapshotID, row.ProductOpportunityID, row.ProductID, row.SupplierSKU, row.InternalSKUID, row.InternalSKU, row.ChannelSKU, row.PlatformID, row.ListingID, row.Version})
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
