package sourcing1688

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func newComplianceService(t *testing.T) (*Service, Sourcing1688Product, Sourcing1688TaskLink) {
	t.Helper()
	svc, db, source, link, snapshot := newSampleFactService(t)
	if err := db.AutoMigrate(&SourcingComplianceEvidence{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE product (id integer primary key)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE sku (id integer primary key, product_id integer, code text)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO product(id) VALUES (701)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO sku(id, product_id, code) VALUES (801,701,'SKU-801')").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&demandCaseRow{}).Where("id = ?", link.DemandCaseID).Updates(map[string]any{"region": "RU", "sales_channel": "Ozon"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&source).Updates(map[string]any{"snapshot_id": snapshot.ID, "product_id": 701}).Error; err != nil {
		t.Fatal(err)
	}
	source.SnapshotID, source.ProductID = &snapshot.ID, ptrInt64(701)
	return svc, source, link
}

func validComplianceInput() *CreateComplianceEvidenceInput {
	now := time.Now().Add(-time.Hour).UTC()
	expiry := now.Add(24 * time.Hour)
	skuID := int64(801)
	return &CreateComplianceEvidenceInput{OwnerID: 42, ProductID: 701, InternalSKUID: &skuID, CountryCode: "RU", ChannelCode: "Ozon", RequirementCode: "eac_declaration", RequirementText: "EAC declaration required", EvidenceSource: "evidence://authority/eac-1", TruthStatus: "actual", Scope: "product 701 / sku 801 / RU / Ozon", IssuedAt: &now, ObservedAt: now, ExpiresAt: &expiry}
}

func TestComplianceEvidenceFreezesAuthorityAndPassesOnlyCurrentApprovedActual(t *testing.T) {
	svc, source, link := newComplianceService(t)
	row, err := svc.CreateComplianceEvidence(source.ID, link.ID, validComplianceInput())
	if err != nil {
		t.Fatal(err)
	}
	if row.OwnerID != 42 || row.ProductOpportunityID == 0 || row.SourceSnapshotID == 0 || row.ReviewStatus != ComplianceReviewPending {
		t.Fatalf("incomplete frozen evidence: %+v", row)
	}
	if err := svc.RequireCurrentCompliance(source.ID, link.ID, 42, []string{"eac_declaration"}, time.Now()); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("pending evidence passed: %v", err)
	}
	row, err = svc.ReviewComplianceEvidence(source.ID, link.ID, row.ID, &ReviewComplianceEvidenceInput{OwnerID: 42, Decision: "approved", Notes: "Owner checked issuing authority and scope"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RequireCurrentCompliance(source.ID, link.ID, 42, []string{"eac_declaration"}, time.Now()); err != nil {
		t.Fatalf("approved current actual failed: %v", err)
	}
	row, err = svc.RevokeComplianceEvidence(source.ID, link.ID, row.ID, &RevokeComplianceEvidenceInput{OwnerID: 42, Reason: "issuing authority revoked declaration"})
	if err != nil {
		t.Fatal(err)
	}
	if row.RevokedAt == nil || !errors.Is(svc.RequireCurrentCompliance(source.ID, link.ID, 42, []string{"eac_declaration"}, time.Now()), ErrWorkflowGate) {
		t.Fatal("revocation did not fail closed")
	}
}

func TestComplianceEvidenceRejectsQuotedExpiredCrossOwnerAndWrongSKU(t *testing.T) {
	svc, source, link := newComplianceService(t)
	quoted := validComplianceInput()
	quoted.TruthStatus = "quoted"
	row, err := svc.CreateComplianceEvidence(source.ID, link.ID, quoted)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReviewComplianceEvidence(source.ID, link.ID, row.ID, &ReviewComplianceEvidenceInput{OwnerID: 42, Decision: "approved", Notes: "not enough"}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("quoted evidence passed: %v", err)
	}
	expired := validComplianceInput()
	past := time.Now().Add(-time.Minute)
	expired.ExpiresAt = &past
	row, err = svc.CreateComplianceEvidence(source.ID, link.ID, expired)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReviewComplianceEvidence(source.ID, link.ID, row.ID, &ReviewComplianceEvidenceInput{OwnerID: 42, Decision: "approved", Notes: "expired"}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("expired evidence passed: %v", err)
	}
	cross := validComplianceInput()
	cross.OwnerID = 99
	if _, err := svc.CreateComplianceEvidence(source.ID, link.ID, cross); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("cross owner accepted: %v", err)
	}
	wrong := validComplianceInput()
	wrongSKU := int64(999)
	wrong.InternalSKUID = &wrongSKU
	if _, err := svc.CreateComplianceEvidence(source.ID, link.ID, wrong); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("wrong SKU accepted: %v", err)
	}
}

func TestComplianceMigrationProtectsScopeTruthReviewAndRevocation(t *testing.T) {
	body, err := os.ReadFile("../../../migrations/000121_sourcing_compliance_evidence.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, required := range []string{"owner_id BIGINT NOT NULL", "product_opportunity_id BIGINT NOT NULL", "source_snapshot_id BIGINT NOT NULL", "internal_sku_id BIGINT", "country_code VARCHAR(16)", "channel_code VARCHAR(64)", "truth_status IN ('actual','quoted','estimated','unknown','mock','inferred')", "expires_at TIMESTAMPTZ", "revoked_at TIMESTAMPTZ", "review_status VARCHAR(16)", "authority chain mismatch", "evidence are immutable", "review is immutable once decided", "revocation is immutable"} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
}
