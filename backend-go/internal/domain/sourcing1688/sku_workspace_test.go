package sourcing1688

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func newSKUWorkspaceFixture(t *testing.T) (*Service, Sourcing1688Product, Sourcing1688TaskLink) {
	t.Helper()
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &Sourcing1688TaskLink{}, &SourcingSKUMapping{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := "SKU工作台商品"
	product := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/321.html", SourceOfferID: "321", Title: &title}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&demandCaseRow{ID: 7, OwnerID: 42, SalesChannel: "Ozon", TargetLocale: "ru-RU"}).Error; err != nil {
		t.Fatal(err)
	}
	opportunityID := int64(70)
	link := Sourcing1688TaskLink{SourcingProductID: product.ID, DemandCaseID: 7, ExperimentID: "EXP-SKU", OwnerID: 42, ProductOpportunityID: &opportunityID}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	return svc, product, link
}

func skuWorkspaceRaw(t *testing.T, driver string, dimensions any, variants any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"driver": driver, "sku_dimensions": dimensions, "spec_variants": variants, "field_statuses": map[string]string{"sku": "observed"}})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSKUWorkspaceParsesDimensionsDuplicatesMissingAndMappings(t *testing.T) {
	svc, product, link := newSKUWorkspaceFixture(t)
	now := time.Date(2026, 7, 13, 3, 0, 0, 0, time.UTC)
	raw := skuWorkspaceRaw(t, "plugin", []map[string]any{{"name": "颜色", "values": []string{"红色", "蓝色"}}, {"name": "尺码", "values": []string{"S", "M"}}}, []map[string]any{
		{"supplier_sku": "RED-S", "spec": "红色 / S", "values": map[string]string{"颜色": "红色", "尺码": "S"}, "price": 10.5, "stock": 8},
		{"supplier_sku": "RED-S-COPY", "spec": "红色 / S", "values": map[string]string{"颜色": "红色", "尺码": "S"}, "price": 10.5, "stock": 8},
		{"supplier_sku": "BLUE-M", "spec": "蓝色 / M", "values": map[string]string{"颜色": "蓝色", "尺码": "M"}},
	})
	snapshot := Sourcing1688Snapshot{SourcingProductID: product.ID, SourceURL: product.SourceURL, CollectedAt: now, CollectedBy: 42, Driver: "chrome_extension", ParserVersion: "detail-v1", CaptureMode: CaptureModeExtensionClick, RawPayload: raw, RawSHA256: "hash"}
	if err := svc.db.Create(&snapshot).Error; err != nil {
		t.Fatal(err)
	}
	mapping := SourcingSKUMapping{OwnerID: 42, SourcingProductID: product.ID, TaskLinkID: link.ID, SnapshotID: snapshot.ID, ProductOpportunityID: 70, ProductID: 80, SupplierSKU: "RED-S", InternalSKUID: 81, InternalSKU: "INT-RED-S", ChannelSKU: "OZ-RED-S", PlatformID: 3, ListingID: 90, Version: 1, ContentHash: "hash", CreatedBy: 42}
	if err := svc.db.Create(&mapping).Error; err != nil {
		t.Fatal(err)
	}

	workspace, err := svc.GetSKUWorkspace(42, product.ID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.Status != "needs_attention" || len(workspace.Dimensions) != 2 || workspace.Dimensions[0].Source != "declared" {
		t.Fatalf("workspace=%#v", workspace)
	}
	if len(workspace.Combinations) != 3 || workspace.Combinations[0].Mapping == nil || workspace.Combinations[0].Mapping.ChannelSKU != "OZ-RED-S" {
		t.Fatalf("combinations=%#v", workspace.Combinations)
	}
	if len(workspace.DuplicateCombinations) != 1 || !workspace.Combinations[0].Duplicate || !workspace.Combinations[1].Duplicate {
		t.Fatalf("duplicates=%#v", workspace.DuplicateCombinations)
	}
	if len(workspace.MissingPrice) != 1 || len(workspace.MissingStock) != 1 || workspace.Combinations[2].QuotedPrice != nil || workspace.Combinations[2].StockStatus != "unknown" {
		t.Fatalf("missing price/stock=%#v", workspace)
	}
	if workspace.MissingCombinations.Status != "calculated" || len(workspace.MissingCombinations.Combinations) != 2 {
		t.Fatalf("missing combinations=%#v", workspace.MissingCombinations)
	}
	if workspace.Target.SalesChannel != "Ozon" || workspace.Target.TargetLocale != "ru-RU" || len(workspace.Target.PlatformIDs) != 1 || workspace.Target.PlatformIDs[0] != 3 {
		t.Fatalf("target=%#v", workspace.Target)
	}
}

func TestSKUWorkspaceDerivedDimensionsNeverClaimsMissingCombinations(t *testing.T) {
	svc, product, link := newSKUWorkspaceFixture(t)
	raw := skuWorkspaceRaw(t, "plugin", nil, []map[string]any{{"spec": "黄色 / 90cm", "price": 11.9}, {"spec": "蓝色 / 100cm", "price": 11.9}})
	if err := svc.db.Create(&Sourcing1688Snapshot{SourcingProductID: product.ID, SourceURL: product.SourceURL, CollectedAt: time.Now().UTC(), CollectedBy: 42, Driver: "extension", ParserVersion: "detail", RawPayload: raw, RawSHA256: "hash"}).Error; err != nil {
		t.Fatal(err)
	}
	workspace, err := svc.GetSKUWorkspace(42, product.ID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspace.Dimensions) != 2 || workspace.Dimensions[0].Name != "维度1" || workspace.MissingCombinations.Status != "unknown" {
		t.Fatalf("workspace=%#v", workspace)
	}
	if workspace.Combinations[0].SupplierSKU != "" {
		t.Fatalf("spec was invented as supplier SKU: %#v", workspace.Combinations[0])
	}
}

func TestSKUWorkspaceIsOwnerAndTaskIsolated(t *testing.T) {
	svc, product, link := newSKUWorkspaceFixture(t)
	if _, err := svc.GetSKUWorkspace(99, product.ID, link.ID); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("cross owner err=%v", err)
	}
	otherLink := Sourcing1688TaskLink{SourcingProductID: product.ID, DemandCaseID: 7, ExperimentID: "EXP-OTHER", OwnerID: 99}
	if err := svc.db.Create(&otherLink).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetSKUWorkspace(42, product.ID, otherLink.ID); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("cross task err=%v", err)
	}
}

func TestSKUWorkspaceWithoutDetailReturnsExplicitBlocker(t *testing.T) {
	svc, product, link := newSKUWorkspaceFixture(t)
	raw := skuWorkspaceRaw(t, "chrome_extension_list_visible", nil, nil)
	if err := svc.db.Create(&Sourcing1688Snapshot{SourcingProductID: product.ID, SourceURL: product.SourceURL, CollectedAt: time.Now().UTC(), CollectedBy: 42, Driver: "extension", ParserVersion: "1688-list-visible-v1", RawPayload: raw, RawSHA256: "hash"}).Error; err != nil {
		t.Fatal(err)
	}
	workspace, err := svc.GetSKUWorkspace(42, product.ID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.Status != "no_detail_observation" || workspace.SnapshotID != 0 || len(workspace.Blockers) == 0 || workspace.MissingCombinations.Status != "unknown" {
		t.Fatalf("workspace=%#v", workspace)
	}
}
