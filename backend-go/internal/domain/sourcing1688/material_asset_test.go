package sourcing1688

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func materialFixture(t *testing.T) (*Service, Sourcing1688Product, Sourcing1688TaskLink, Sourcing1688Snapshot, SourcingSKUMapping) {
	t.Helper()
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &Sourcing1688TaskLink{}, &SourcingSKUMapping{}, &SourcingMaterialAsset{}, &SourcingMaterialRightsEvidence{}, &SourcingMaterialRendition{}, &ImageProcessingRecord{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := "素材商品"
	product := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/1.html", SourceOfferID: "1", Title: &title}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	snapshot := Sourcing1688Snapshot{SourcingProductID: product.ID, SourceURL: product.SourceURL, CollectedAt: time.Now().UTC(), CollectedBy: 42, Driver: "extension", ParserVersion: "v1", RawPayload: []byte(`{}`), RawSHA256: strings.Repeat("a", 64)}
	if err := db.Create(&snapshot).Error; err != nil {
		t.Fatal(err)
	}
	link := Sourcing1688TaskLink{SourcingProductID: product.ID, DemandCaseID: 7, ExperimentID: "EXP-M", OwnerID: 42}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	mapping := SourcingSKUMapping{OwnerID: 42, SourcingProductID: product.ID, TaskLinkID: link.ID, SnapshotID: snapshot.ID, ProductOpportunityID: 8, ProductID: 9, SupplierSKU: "SUP", InternalSKUID: 10, InternalSKU: "INT", ChannelSKU: "CH", PlatformID: 11, ListingID: 12, Version: 1, ContentHash: "h", CreatedBy: 42}
	if err := db.Create(&mapping).Error; err != nil {
		t.Fatal(err)
	}
	return svc, product, link, snapshot, mapping
}

func imageAssetInput(snapshotID int64, role string, ordinal int, mapping *int64) CreateMaterialAssetInput {
	return CreateMaterialAssetInput{SnapshotID: snapshotID, Role: role, Ordinal: ordinal, CanonicalSKUMappingID: mapping, SourceURL: "https://cbu01.alicdn.com/a.jpg", SourceSHA256: strings.Repeat("b", 64), MediaType: "image", MIMEType: "image/jpeg", ByteSize: 100, Width: ptrInt(800), Height: ptrInt(800)}
}
func ptrInt(v int) *int { return &v }

func TestMaterialAssetOwnerUniqueMainOrderingAndSKUBinding(t *testing.T) {
	svc, product, link, snapshot, mapping := materialFixture(t)
	asset, err := svc.CreateMaterialAsset(42, product.ID, link.ID, imageAssetInput(snapshot.ID, "main", 0, nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.CreateMaterialAsset(42, product.ID, link.ID, imageAssetInput(snapshot.ID, "main", 1, nil)); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("second main err=%v", err)
	}
	sku, err := svc.CreateMaterialAsset(42, product.ID, link.ID, imageAssetInput(snapshot.ID, "sku", 0, &mapping.ID))
	if err != nil {
		t.Fatal(err)
	}
	if sku.CanonicalSKUMappingID == nil || *sku.CanonicalSKUMappingID != mapping.ID {
		t.Fatalf("sku binding=%#v", sku)
	}
	if _, err = svc.ReorderMaterialAsset(42, product.ID, link.ID, sku.ID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.ListMaterialAssets(99, product.ID, link.ID); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("owner isolation err=%v", err)
	}
	if asset.SourceSHA256 != strings.Repeat("b", 64) {
		t.Fatal("source hash changed")
	}
}

func TestMaterialRightsApprovalExpiryAndRevocation(t *testing.T) {
	svc, product, link, snapshot, _ := materialFixture(t)
	asset, err := svc.CreateMaterialAsset(42, product.ID, link.ID, imageAssetInput(snapshot.ID, "gallery", 0, nil))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	future := now.Add(time.Hour)
	rights, err := svc.AddMaterialRights(42, product.ID, link.ID, asset.ID, CreateMaterialRightsInput{LicenseScope: "跨境商品展示", Countries: []string{"RU"}, Channels: []string{"Ozon"}, Licensor: "供应商", SourceURI: "evidence://rights", ObservedAt: now, ValidUntil: &future})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := svc.ReviewMaterialRights(42, product.ID, link.ID, asset.ID, rights.ID, ReviewMaterialRightsInput{Decision: "approved", ReviewNote: "Owner核验"})
	if err != nil || approved.Status != "approved" {
		t.Fatalf("approved=%#v err=%v", approved, err)
	}
	revoked, err := svc.RevokeMaterialRights(42, product.ID, link.ID, asset.ID, rights.ID, "许可方撤回")
	if err != nil || revoked.Status != "revoked" {
		t.Fatalf("revoked=%#v err=%v", revoked, err)
	}
	past := now.Add(-time.Minute)
	expired := SourcingMaterialRightsEvidence{AssetID: asset.ID, OwnerID: 42, Version: 2, Status: "approved", LicenseScope: "x", Countries: []byte(`[]`), Channels: []byte(`[]`), Licensor: "x", SourceURI: "x", ObservedAt: now.Add(-time.Hour), ValidUntil: &past, SubmittedBy: 42}
	if err := svc.db.Create(&expired).Error; err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListMaterialAssets(42, product.ID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if list.Assets[0].LatestRights == nil || list.Assets[0].LatestRights.EffectiveStatus != "expired" {
		t.Fatalf("latest rights=%#v", list.Assets[0].LatestRights)
	}
}

func TestMaterialVideoBlockedAndUsedAssetCannotArchive(t *testing.T) {
	svc, product, link, snapshot, _ := materialFixture(t)
	duration := int64(1000)
	video, err := svc.CreateMaterialAsset(42, product.ID, link.ID, CreateMaterialAssetInput{SnapshotID: snapshot.ID, Role: "video", Ordinal: 0, SourceURL: "https://example.com/v.mp4", SourceSHA256: strings.Repeat("c", 64), MediaType: "video", MIMEType: "video/mp4", ByteSize: 200, DurationMS: &duration})
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListMaterialAssets(42, product.ID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if list.Assets[0].ProcessingStatus != "blocked" || list.Assets[0].Blocker == "" {
		t.Fatalf("video=%#v", list.Assets[0])
	}
	if _, err = svc.AttachMaterialRendition(42, product.ID, link.ID, video.ID, MaterialRenditionInput{ImageProcessingRecordID: 1}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("video rendition err=%v", err)
	}
	image, err := svc.CreateMaterialAsset(42, product.ID, link.ID, imageAssetInput(snapshot.ID, "gallery", 0, nil))
	if err != nil {
		t.Fatal(err)
	}
	used := time.Now().UTC()
	if err := svc.db.Model(image).Update("used_at", used).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = svc.ArchiveMaterialAsset(42, product.ID, link.ID, image.ID); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("archive used err=%v", err)
	}
}

func TestMaterialMigrationProtectsImmutableHashAndRightsContent(t *testing.T) {
	data, err := os.ReadFile("../../../migrations/000148_sourcing_material_assets.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{"source_sha256", "sourcing_material_source_immutable", "source identity and hash are immutable", "sourcing_material_rights_content_immutable", "UNIQUE INDEX ux_sourcing_material_main"} {
		if !strings.Contains(text, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
}

func TestMarkMaterialUsedRequiresLatestScopedRights(t *testing.T) {
	svc, product, link, snapshot, _ := materialFixture(t)
	if err := svc.db.Create(&demandCaseRow{ID: link.DemandCaseID, OwnerID: 42, Region: "RU", SalesChannel: "Ozon"}).Error; err != nil {
		t.Fatal(err)
	}
	asset, err := svc.CreateMaterialAsset(42, product.ID, link.ID, imageAssetInput(snapshot.ID, "main", 0, nil))
	if err != nil {
		t.Fatal(err)
	}
	now, future := time.Now().UTC(), time.Now().UTC().Add(time.Hour)
	rights, err := svc.AddMaterialRights(42, product.ID, link.ID, asset.ID, CreateMaterialRightsInput{LicenseScope: "跨境展示", Countries: []string{"RU"}, Channels: []string{"Ozon"}, Licensor: "供应商", SourceURI: "evidence://rights", ObservedAt: now, ValidUntil: &future})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReviewMaterialRights(42, product.ID, link.ID, asset.ID, rights.ID, ReviewMaterialRightsInput{Decision: "approved", ReviewNote: "核验"}); err != nil {
		t.Fatal(err)
	}
	record := ImageProcessingRecord{SourcingProductID: product.ID, SnapshotID: snapshot.ID, SourceURL: asset.SourceURL, SourceSHA256: asset.SourceSHA256, ProcessedSHA256: strings.Repeat("d", 64), OutputFormat: "jpeg", OutputWidth: 800, OutputHeight: 800, Quality: 90, ProcessorVersion: "test", Operations: []byte(`[]`), RightsEvidenceURI: "evidence://rights", RightsTruthStatus: "actual", RightsObservedAt: now, ChannelRuleURI: "evidence://rule", EvidenceFingerprint: strings.Repeat("e", 64), ProcessedBytes: []byte("image"), ProcessedBy: 42}
	if err := svc.db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AttachMaterialRendition(42, product.ID, link.ID, asset.ID, MaterialRenditionInput{ImageProcessingRecordID: record.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMaterialRights(42, product.ID, link.ID, asset.ID, CreateMaterialRightsInput{LicenseScope: "跨境展示", Countries: []string{"DE"}, Channels: []string{"Amazon"}, Licensor: "供应商", SourceURI: "evidence://replacement", ObservedAt: now, ValidUntil: &future}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkMaterialUsed(42, product.ID, link.ID, asset.ID); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("newer pending rights must block old approved rights: %v", err)
	}
}
