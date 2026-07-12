package sourcing1688

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSKUMappingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&skuMappingProductRow{}, &skuMappingSnapshotRow{}, &skuMappingTaskRow{}, &skuMappingOpportunityRow{}, &skuMappingInternalRow{}, &skuMappingListingRow{}, &SourcingSKUMapping{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedSKUMappingFixture(t *testing.T, db *gorm.DB) CanonicalSKUMappingInput {
	t.Helper()
	opportunityID := int64(41)
	rows := []any{
		&skuMappingProductRow{ID: 11, OwnerID: 7},
		&skuMappingSnapshotRow{ID: 21, SourcingProductID: 11},
		&skuMappingTaskRow{ID: 31, OwnerID: 7, SourcingProductID: 11, ProductOpportunityID: &opportunityID},
		&skuMappingOpportunityRow{ID: 41, OwnerID: 7},
		&skuMappingInternalRow{ID: 61, ProductID: 51, Code: "INT-RED"},
		&skuMappingInternalRow{ID: 62, ProductID: 51, Code: "INT-BLUE"},
		&skuMappingListingRow{ID: 71, ProductID: 51, PlatformID: 81},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	return CanonicalSKUMappingInput{OwnerID: 7, CreatedBy: 7, SourcingProductID: 11, TaskLinkID: 31, SnapshotID: 21, ProductOpportunityID: 41, ProductID: 51, PlatformID: 81, ListingID: 71, Version: 1, Items: []CanonicalSKUMappingItem{
		{SupplierSKU: "SUP-RED", InternalSKUID: 61, InternalSKU: "INT-RED", ChannelSKU: "CH-RED"},
		{SupplierSKU: "SUP-BLUE", InternalSKUID: 62, InternalSKU: "INT-BLUE", ChannelSKU: "CH-BLUE"},
	}}
}

func TestPersistCanonicalSKUMappingsFreezesThreeStageIdentityAndReplays(t *testing.T) {
	db := newSKUMappingTestDB(t)
	in := seedSKUMappingFixture(t, db)
	rows, err := PersistCanonicalSKUMappings(db, in)
	if err != nil {
		t.Fatalf("persist: %v", err)
	}
	if len(rows) != 2 || rows[0].ContentHash == "" || len(rows[0].ContentHash) != 64 {
		t.Fatalf("unexpected rows: %#v", rows)
	}
	if rows[0].SupplierSKU != "SUP-BLUE" || rows[0].InternalSKU != "INT-BLUE" || rows[0].ChannelSKU != "CH-BLUE" {
		t.Fatalf("mapping was not canonicalized: %#v", rows[0])
	}
	replayed, err := PersistCanonicalSKUMappings(db, in)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(replayed) != 2 || replayed[0].ID != rows[0].ID {
		t.Fatalf("replay inserted another version: %#v", replayed)
	}
	var count int64
	if err := db.Model(&SourcingSKUMapping{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestPersistCanonicalSKUMappingsRejectsFrozenVersionMutation(t *testing.T) {
	db := newSKUMappingTestDB(t)
	in := seedSKUMappingFixture(t, db)
	if _, err := PersistCanonicalSKUMappings(db, in); err != nil {
		t.Fatal(err)
	}
	in.Items[0].ChannelSKU = "CH-TAMPERED"
	_, err := PersistCanonicalSKUMappings(db, in)
	if !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("expected frozen-version gate, got %v", err)
	}
}

func TestPersistCanonicalSKUMappingsRejectsCrossOwnerAndIdentityMismatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CanonicalSKUMappingInput)
	}{
		{"cross owner", func(in *CanonicalSKUMappingInput) { in.OwnerID, in.CreatedBy = 8, 8 }},
		{"wrong snapshot", func(in *CanonicalSKUMappingInput) { in.SnapshotID = 999 }},
		{"wrong opportunity", func(in *CanonicalSKUMappingInput) { in.ProductOpportunityID = 42 }},
		{"wrong internal code", func(in *CanonicalSKUMappingInput) { in.Items[0].InternalSKU = "OTHER" }},
		{"wrong listing product", func(in *CanonicalSKUMappingInput) { in.ProductID = 52 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := newSKUMappingTestDB(t)
			in := seedSKUMappingFixture(t, db)
			tc.mutate(&in)
			if _, err := PersistCanonicalSKUMappings(db, in); !errors.Is(err, ErrWorkflowGate) {
				t.Fatalf("expected fail-closed authority gate, got %v", err)
			}
		})
	}
}

func TestPersistCanonicalSKUMappingsRequiresOneToOneMapping(t *testing.T) {
	db := newSKUMappingTestDB(t)
	in := seedSKUMappingFixture(t, db)
	in.Items[1].ChannelSKU = in.Items[0].ChannelSKU
	if _, err := PersistCanonicalSKUMappings(db, in); !errors.Is(err, ErrInvalidWorkflow) {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
}

func TestSKUMappingMigrationDeclaresRelationalAndAppendOnlyGuards(t *testing.T) {
	path := filepath.Join("..", "..", "..", "migrations", "000114_sourcing_sku_mapping_authority.up.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, required := range []string{"product_opportunity_id BIGINT NOT NULL", "internal_sku_id BIGINT NOT NULL", "listing_id BIGINT NOT NULL", "UNIQUE (owner_id, task_link_id, version, supplier_sku)", "validate_sourcing_sku_mapping_binding", "reject_sourcing_sku_mapping_mutation", "BEFORE UPDATE OR DELETE"} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
}
