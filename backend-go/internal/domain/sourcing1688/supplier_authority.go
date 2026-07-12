package sourcing1688

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const sourcingSupplierSystem = "1688"

type authoritativeSupplierRow struct {
	ID                 int64      `gorm:"column:id;primaryKey"`
	OwnerID            int64      `gorm:"column:owner_id"`
	Name               string     `gorm:"column:name"`
	Status             int16      `gorm:"column:status"`
	SourceSystem       string     `gorm:"column:source_system"`
	ExternalBusinessID string     `gorm:"column:external_business_id"`
	SourceSnapshotID   *int64     `gorm:"column:source_snapshot_id"`
	IdentitySHA256     string     `gorm:"column:identity_sha256"`
	TruthStatus        string     `gorm:"column:truth_status"`
	ObservedAt         *time.Time `gorm:"column:observed_at"`
	VerifiedBy         *int64     `gorm:"column:verified_by"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

func (authoritativeSupplierRow) TableName() string { return "supplier" }

type authoritativeProductSupplierRow struct {
	ID                   int64      `gorm:"column:id;primaryKey"`
	OwnerID              int64      `gorm:"column:owner_id"`
	ProductID            int64      `gorm:"column:product_id"`
	SupplierID           int64      `gorm:"column:supplier_id"`
	SourcingProductID    int64      `gorm:"column:sourcing_product_id"`
	SourceSnapshotID     int64      `gorm:"column:source_snapshot_id"`
	ProductOpportunityID int64      `gorm:"column:product_opportunity_id"`
	SupplyPrice          *float64   `gorm:"column:supply_price"`
	MinOrderQty          int        `gorm:"column:min_order_qty"`
	TruthStatus          string     `gorm:"column:truth_status"`
	SourceURI            string     `gorm:"column:source_uri"`
	ObservedAt           *time.Time `gorm:"column:observed_at"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
}

func (authoritativeProductSupplierRow) TableName() string { return "product_supplier" }

func supplierIdentityHash(ownerID int64, externalBusinessID string) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s\x00%s", ownerID, sourcingSupplierSystem, strings.TrimSpace(externalBusinessID))))
	return hex.EncodeToString(digest[:])
}

// ensureAuthoritativeSupplier resolves one Owner-scoped supplier identity from
// the immutable source snapshot. A display-name change is evidence evolution;
// a business-id or source mismatch is an identity conflict and fails closed.
func ensureAuthoritativeSupplier(tx *gorm.DB, source *Sourcing1688Product, snapshot *Sourcing1688Snapshot, ownerID int64) (*authoritativeSupplierRow, error) {
	if source == nil || snapshot == nil || ownerID <= 0 || source.OwnerID != ownerID || snapshot.SourcingProductID != source.ID || snapshot.CollectedBy != ownerID {
		return nil, fmt.Errorf("%w: supplier source identity does not belong to authenticated Owner", ErrWorkflowGate)
	}
	businessID := strings.TrimSpace(source.SupplierBusinessID)
	observedBusinessID := strings.TrimSpace(snapshot.ObservedSupplierBusinessID)
	name := strings.TrimSpace(snapshot.ObservedSupplier)
	if businessID == "" || observedBusinessID == "" || businessID != observedBusinessID {
		return nil, fmt.Errorf("%w: immutable snapshot supplier identity does not match the sourcing record", ErrWorkflowGate)
	}
	if name == "" {
		name = businessID
	}
	identityHash := supplierIdentityHash(ownerID, businessID)
	var supplier authoritativeSupplierRow
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("owner_id = ? AND source_system = ? AND external_business_id = ?", ownerID, sourcingSupplierSystem, businessID).
		First(&supplier).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		observedAt, verifiedBy, snapshotID := snapshot.CollectedAt.UTC(), ownerID, snapshot.ID
		supplier = authoritativeSupplierRow{OwnerID: ownerID, Name: name, Status: 1, SourceSystem: sourcingSupplierSystem, ExternalBusinessID: businessID, SourceSnapshotID: &snapshotID, IdentitySHA256: identityHash, TruthStatus: "quoted", ObservedAt: &observedAt, VerifiedBy: &verifiedBy}
		if err := tx.Create(&supplier).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		if supplier.IdentitySHA256 != identityHash || supplier.OwnerID != ownerID || supplier.SourceSystem != sourcingSupplierSystem || supplier.ExternalBusinessID != businessID {
			return nil, fmt.Errorf("%w: supplier external identity conflicts with the frozen Owner record", ErrWorkflowGate)
		}
		if supplier.Status != 1 {
			return nil, fmt.Errorf("%w: supplier is inactive", ErrWorkflowGate)
		}
	}
	return &supplier, nil
}

func ensureSourceSupplierAuthority(tx *gorm.DB, source *Sourcing1688Product, ownerID int64) (*authoritativeSupplierRow, *Sourcing1688Snapshot, error) {
	if source == nil || source.SnapshotID == nil {
		return nil, nil, fmt.Errorf("%w: immutable source snapshot required for supplier identity", ErrWorkflowGate)
	}
	var snapshot Sourcing1688Snapshot
	if err := tx.First(&snapshot, *source.SnapshotID).Error; err != nil {
		return nil, nil, err
	}
	supplier, err := ensureAuthoritativeSupplier(tx, source, &snapshot, ownerID)
	if err != nil {
		return nil, nil, err
	}
	if source.SupplierID == nil || *source.SupplierID != supplier.ID {
		if err := tx.Model(source).Update("supplier_id", supplier.ID).Error; err != nil {
			return nil, nil, err
		}
		source.SupplierID = &supplier.ID
	}
	return supplier, &snapshot, nil
}

func bindProductSupplier(tx *gorm.DB, source *Sourcing1688Product, snapshot *Sourcing1688Snapshot, task *Sourcing1688TaskLink, supplier *authoritativeSupplierRow, productID int64) error {
	if source == nil || snapshot == nil || task == nil || supplier == nil || task.ProductOpportunityID == nil || productID <= 0 {
		return fmt.Errorf("%w: complete supplier binding authority is required", ErrWorkflowGate)
	}
	observedAt := snapshot.CollectedAt.UTC()
	relation := authoritativeProductSupplierRow{OwnerID: source.OwnerID, ProductID: productID, SupplierID: supplier.ID, SourcingProductID: source.ID, SourceSnapshotID: snapshot.ID, ProductOpportunityID: *task.ProductOpportunityID, SupplyPrice: source.Price, MinOrderQty: source.MOQ, TruthStatus: "quoted", SourceURI: snapshot.SourceURL, ObservedAt: &observedAt}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&relation).Error; err != nil {
		return err
	}
	return tx.Model(source).Updates(map[string]any{"supplier_id": supplier.ID}).Error
}
