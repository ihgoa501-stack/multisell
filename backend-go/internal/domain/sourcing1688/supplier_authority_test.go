package sourcing1688

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

// These deliberately small rows describe the cross-domain persistence contract
// exercised by sourcing conversion. They keep this package's tests independent
// from supplier's HTTP/service layer while still using the canonical tables.
type supplierAuthorityTestRow struct {
	ID                 int64 `gorm:"primaryKey"`
	OwnerID            int64
	Name               string
	Status             int16
	SourceSystem       string
	ExternalBusinessID string
	SourceSnapshotID   int64
	IdentitySHA256     string
	TruthStatus        string
	ObservedAt         time.Time
	VerifiedBy         int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (supplierAuthorityTestRow) TableName() string { return "supplier" }

type productSupplierAuthorityTestRow struct {
	ID                   int64 `gorm:"primaryKey"`
	OwnerID              int64
	ProductID            int64
	SupplierID           int64
	SupplyPrice          *float64
	MinOrderQty          int
	SourcingProductID    int64
	SourceSnapshotID     int64
	ProductOpportunityID int64
	TruthStatus          string
	SourceURI            string
	ObservedAt           time.Time
	CreatedAt            time.Time
}

func (productSupplierAuthorityTestRow) TableName() string { return "product_supplier" }

type supplierAuthorityFixture struct {
	svc        *Service
	db         *gorm.DB
	now        time.Time
	platformID int64
}

func newSupplierAuthorityFixture(t *testing.T) *supplierAuthorityFixture {
	t.Helper()
	db := dbtest.NewDB(t,
		&Sourcing1688Product{}, &Sourcing1688Snapshot{}, &DuplicateCandidate{}, &ImageProcessingRecord{},
		&demandCaseRow{}, &experimentRow{}, &gateRow{}, &objectLinkRow{}, &platformRow{},
		&productRow{}, &skuRow{}, &mediaRow{}, &costRow{}, &listingRow{}, &draftRow{},
		&Sourcing1688TaskLink{}, &sourcingOpportunityRow{}, &sourcingOpportunityDecisionRow{}, &sourcingMarketDecisionRow{},
		&supplierAuthorityTestRow{}, &productSupplierAuthorityTestRow{}, &SourcingSKUMapping{},
	)
	now := time.Now().UTC().Truncate(time.Second)
	platform := platformRow{Name: "Ozon", Code: "ozon", Status: 1}
	if err := db.Create(&platform).Error; err != nil {
		t.Fatalf("create platform: %v", err)
	}
	return &supplierAuthorityFixture{svc: NewService(db, dbtest.NewLogger(t)), db: db, now: now, platformID: platform.ID}
}

func (f *supplierAuthorityFixture) seedConvertibleSource(t *testing.T, ownerID, caseID int64, experimentID, businessID, supplierName string) (Sourcing1688Product, *ConvertInput) {
	t.Helper()
	if err := f.db.Create(&demandCaseRow{ID: caseID, OwnerID: ownerID, Region: "RU", Consumer: "test consumer", NeedScenario: "test need", SalesChannel: "Ozon", TargetLocale: "ru-RU", Status: "experiment_ready"}).Error; err != nil {
		t.Fatalf("create demand case: %v", err)
	}
	if err := f.db.Create(&experimentRow{ExperimentID: experimentID, OwnerID: ownerID, Status: "active", Stage: "product"}).Error; err != nil {
		t.Fatalf("create trace case: %v", err)
	}
	if err := f.db.Create(&gateRow{ExperimentID: experimentID, Stage: "opportunity", Result: "pass"}).Error; err != nil {
		t.Fatalf("create trace gate: %v", err)
	}
	if err := f.db.Create(&objectLinkRow{ExperimentID: experimentID, ObjectType: "demand_case", ObjectID: fmt.Sprint(caseID)}).Error; err != nil {
		t.Fatalf("create trace demand-case link: %v", err)
	}
	source := Sourcing1688Product{
		OwnerID: ownerID, SourceURL: fmt.Sprintf("https://detail.1688.com/offer/%d.html", caseID),
		SourceOfferID: fmt.Sprintf("offer-%d", caseID), SupplierBusinessID: businessID, SupplierName: supplierName,
		Status: StatusReviewed, LifecycleStatus: LifecycleReadyForProduct, DemandCaseID: &caseID, ExperimentID: &experimentID,
		ReviewedBy: &ownerID, ReviewedAt: &f.now,
	}
	if err := f.db.Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	raw := json.RawMessage(fmt.Sprintf(`{"supplier_business_id":%q}`, businessID))
	snapshot := Sourcing1688Snapshot{
		SourcingProductID: source.ID, SourceURL: source.SourceURL, CollectedAt: f.now, CollectedBy: ownerID,
		Driver: "owner-browser", ParserVersion: "test-1", CaptureMode: CaptureModeControlledFetch,
		CollectionRequestID: fmt.Sprintf("req_supplier_%d", caseID), RawPayload: raw, RawSHA256: strings.Repeat("a", 64),
		ObservedSupplier: supplierName, ObservedSupplierBusinessID: businessID,
	}
	if err := f.db.Create(&snapshot).Error; err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if err := f.db.Model(&source).Update("snapshot_id", snapshot.ID).Error; err != nil {
		t.Fatalf("bind snapshot: %v", err)
	}
	source.SnapshotID = &snapshot.ID
	seedFrozenOpportunityAuthority(t, f.db, source.ID, caseID, ownerID, experimentID, "Ozon")

	processed := ImageProcessingRecord{
		SourcingProductID: source.ID, SnapshotID: snapshot.ID, SourceURL: "https://cbu01.alicdn.com/a.png",
		SourceSHA256: strings.Repeat("b", 64), ProcessedSHA256: strings.Repeat("c", 64), OutputFormat: "jpeg",
		OutputWidth: 1200, OutputHeight: 1200, Quality: 90, ProcessorVersion: "test-1",
		Operations: json.RawMessage(`["crop","background_remove"]`), RightsEvidenceURI: "evidence://rights/1",
		RightsTruthStatus: "actual", RightsObservedAt: f.now, ChannelRuleURI: "evidence://ozon/images/1",
		EvidenceFingerprint: strings.Repeat("d", 64), ProcessedBytes: []byte("test-image"), ProcessedBy: ownerID,
	}
	if err := f.db.Create(&processed).Error; err != nil {
		t.Fatalf("create image processing evidence: %v", err)
	}
	in := completeConvertInput(f.now)
	in.CreatedBy = ownerID
	in.PlatformID = f.platformID
	in.ConversionRequestID = fmt.Sprintf("convert_supplier_%d", caseID)
	in.Media[0].ProcessingRecordID = processed.ID
	in.Media[0].ProcessedURL = fmt.Sprintf("/api/v1/sourcing-1688/processed-images/%d/content", processed.ID)
	in.Media[0].ContentSHA256 = processed.ProcessedSHA256
	in.Media[0].Operations = processed.Operations
	in.Media[0].RightsEvidenceURI = processed.RightsEvidenceURI
	in.Media[0].RightsObservedAt = processed.RightsObservedAt
	in.Media[0].ChannelRuleURI = processed.ChannelRuleURI
	in.Validation.Channel.PlatformID = f.platformID
	in.Validation.ChannelRules.PlatformID = f.platformID
	in.Validation.Images[0].SourceURI = "sha256:" + processed.ProcessedSHA256
	return source, in
}

func TestConvertCreatesCanonicalSupplierAndProductSupplierAuthority(t *testing.T) {
	f := newSupplierAuthorityFixture(t)
	source, in := f.seedConvertibleSource(t, 42, 7001, "EXP-SUPPLIER-7001", "1688-business-001", "杭州真实供应商")
	result, err := f.svc.Convert(source.ID, in)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	var supplier supplierAuthorityTestRow
	if err := f.db.Where("owner_id = ? AND source_system = ? AND external_business_id = ?", 42, "1688", "1688-business-001").First(&supplier).Error; err != nil {
		t.Fatalf("canonical supplier missing: %v", err)
	}
	if supplier.Name != "杭州真实供应商" || supplier.SourceSnapshotID != *source.SnapshotID || supplier.TruthStatus != "quoted" || supplier.VerifiedBy != 42 || supplier.IdentitySHA256 == "" || supplier.ObservedAt.IsZero() {
		t.Fatalf("supplier authority not frozen from source snapshot: %+v", supplier)
	}
	var link productSupplierAuthorityTestRow
	if err := f.db.Where("product_id = ? AND supplier_id = ?", result.ProductID, supplier.ID).First(&link).Error; err != nil {
		t.Fatalf("product_supplier authority missing: %v", err)
	}
	var task Sourcing1688TaskLink
	if err := f.db.First(&task, result.TaskLinkID).Error; err != nil {
		t.Fatal(err)
	}
	if link.OwnerID != 42 || link.SourcingProductID != source.ID || link.SourceSnapshotID != *source.SnapshotID || task.ProductOpportunityID == nil || link.ProductOpportunityID != *task.ProductOpportunityID || link.TruthStatus != "quoted" || link.SourceURI == "" || link.ObservedAt.IsZero() {
		t.Fatalf("product_supplier provenance incomplete: %+v task=%+v", link, task)
	}
	var persisted Sourcing1688Product
	if err := f.db.First(&persisted, source.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.SupplierID == nil || *persisted.SupplierID != supplier.ID {
		t.Fatalf("source supplier_id = %v, want %d", persisted.SupplierID, supplier.ID)
	}
}

func TestConvertReusesSupplierForSameOwnerAndBusinessID(t *testing.T) {
	f := newSupplierAuthorityFixture(t)
	first, firstInput := f.seedConvertibleSource(t, 42, 7101, "EXP-SUPPLIER-7101", "1688-business-reuse", "同一供应商")
	if _, err := f.svc.Convert(first.ID, firstInput); err != nil {
		t.Fatalf("first Convert: %v", err)
	}
	second, secondInput := f.seedConvertibleSource(t, 42, 7102, "EXP-SUPPLIER-7102", "1688-business-reuse", "同一供应商")
	if _, err := f.svc.Convert(second.ID, secondInput); err != nil {
		t.Fatalf("second Convert: %v", err)
	}
	var suppliers []supplierAuthorityTestRow
	if err := f.db.Where("owner_id = ? AND external_business_id = ?", 42, "1688-business-reuse").Find(&suppliers).Error; err != nil {
		t.Fatal(err)
	}
	if len(suppliers) != 1 {
		t.Fatalf("same Owner and business identity created %d suppliers, want 1", len(suppliers))
	}
	var sources []Sourcing1688Product
	if err := f.db.Where("id IN ?", []int64{first.ID, second.ID}).Order("id").Find(&sources).Error; err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 || sources[0].SupplierID == nil || sources[1].SupplierID == nil || *sources[0].SupplierID != suppliers[0].ID || *sources[1].SupplierID != suppliers[0].ID {
		t.Fatalf("sources did not reuse canonical supplier: %+v", sources)
	}
}

func TestConvertDoesNotReuseSupplierAcrossOwners(t *testing.T) {
	f := newSupplierAuthorityFixture(t)
	first, firstInput := f.seedConvertibleSource(t, 42, 7201, "EXP-SUPPLIER-7201", "shared-external-business", "供应商")
	if _, err := f.svc.Convert(first.ID, firstInput); err != nil {
		t.Fatalf("first Convert: %v", err)
	}
	second, secondInput := f.seedConvertibleSource(t, 84, 7202, "EXP-SUPPLIER-7202", "shared-external-business", "供应商")
	if _, err := f.svc.Convert(second.ID, secondInput); err != nil {
		t.Fatalf("second Convert: %v", err)
	}
	var suppliers []supplierAuthorityTestRow
	if err := f.db.Where("external_business_id = ?", "shared-external-business").Order("owner_id").Find(&suppliers).Error; err != nil {
		t.Fatal(err)
	}
	if len(suppliers) != 2 || suppliers[0].OwnerID == suppliers[1].OwnerID || suppliers[0].ID == suppliers[1].ID {
		t.Fatalf("cross-Owner supplier identities were not isolated: %+v", suppliers)
	}
}

func TestConvertFailsClosedOnConflictingSupplierSourceIdentity(t *testing.T) {
	f := newSupplierAuthorityFixture(t)
	conflict := supplierAuthorityTestRow{OwnerID: 42, Name: "身份摘要冲突的供应商", Status: 1, SourceSystem: "1688", ExternalBusinessID: "conflicting-business", SourceSnapshotID: 999, IdentitySHA256: strings.Repeat("e", 64), TruthStatus: "actual", ObservedAt: f.now, VerifiedBy: 42}
	if err := f.db.Create(&conflict).Error; err != nil {
		t.Fatal(err)
	}
	source, in := f.seedConvertibleSource(t, 42, 7301, "EXP-SUPPLIER-7301", "conflicting-business", "1688供应商")
	var before int64
	if err := f.db.Model(&productRow{}).Count(&before).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Convert(source.ID, in); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("conflicting source identity must fail closed with workflow gate, got %v", err)
	}
	var after int64
	if err := f.db.Model(&productRow{}).Count(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("failed conversion left product rows: before=%d after=%d", before, after)
	}
	var links int64
	if err := f.db.Model(&productSupplierAuthorityTestRow{}).Where("sourcing_product_id = ?", source.ID).Count(&links).Error; err != nil {
		t.Fatal(err)
	}
	if links != 0 {
		t.Fatalf("failed conversion left %d product_supplier rows", links)
	}
}

func TestConvertRejectsSnapshotSupplierBusinessIDMismatch(t *testing.T) {
	f := newSupplierAuthorityFixture(t)
	source, in := f.seedConvertibleSource(t, 42, 7401, "EXP-SUPPLIER-7401", "source-business-id", "供应商")
	if err := f.db.Model(&Sourcing1688Snapshot{}).Where("id = ?", *source.SnapshotID).Update("observed_supplier_business_id", "different-snapshot-business-id").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Convert(source.ID, in); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("snapshot/source supplier identity mismatch must be blocked, got %v", err)
	}
	var products int64
	if err := f.db.Model(&productRow{}).Count(&products).Error; err != nil {
		t.Fatal(err)
	}
	if products != 0 {
		t.Fatalf("identity mismatch created %d products", products)
	}
}
