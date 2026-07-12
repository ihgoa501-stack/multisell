package sourcing1688

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newSampleFactService(t *testing.T) (*Service, *gorm.DB, Sourcing1688Product, Sourcing1688TaskLink, Sourcing1688Snapshot) {
	t.Helper()
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &Sourcing1688TaskLink{},
		&SourcingSample{}, &SourcingSampleEvent{}, &sourcingOpportunityRow{}, &sourcingOpportunityDecisionRow{},
		&sourcingMarketDecisionRow{}, &demandCaseRow{}, &experimentRow{}, &gateRow{}, &objectLinkRow{})
	ownerID, caseID, supplierID := int64(42), int64(7), int64(501)
	experimentID := "EXP-SAMPLE-FACT"
	if err := db.Create(&demandCaseRow{ID: caseID, OwnerID: ownerID, SalesChannel: "Ozon", Status: "experiment_ready"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&experimentRow{ExperimentID: experimentID, OwnerID: ownerID, Status: "active", Stage: "supply"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&gateRow{ExperimentID: experimentID, Stage: "opportunity", Result: "pass"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&objectLinkRow{ExperimentID: experimentID, ObjectType: "demand_case", ObjectID: fmt.Sprint(caseID)}).Error; err != nil {
		t.Fatal(err)
	}
	source := Sourcing1688Product{OwnerID: ownerID, DemandCaseID: &caseID, ExperimentID: &experimentID, SupplierID: &supplierID, SourceURL: "https://detail.1688.com/offer/1.html", SupplierName: "supplier", Status: "reviewed"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	snapshot := Sourcing1688Snapshot{SourcingProductID: source.ID, SourceURL: source.SourceURL, CollectedAt: time.Now().Add(-time.Hour), CollectedBy: ownerID, Driver: "test", ParserVersion: "v1", RawPayload: []byte(`{"id":1}`), RawSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	if err := db.Create(&snapshot).Error; err != nil {
		t.Fatal(err)
	}
	seedApprovedSourcingAuthority(t, db, source.ID, ownerID, caseID, experimentID, "Ozon")
	var link Sourcing1688TaskLink
	if err := db.Where("sourcing_product_id = ?", source.ID).First(&link).Error; err != nil {
		t.Fatal(err)
	}
	return NewService(db, dbtest.NewLogger(t)), db, source, link, snapshot
}

func TestSourcingSampleFullFactChainDoesNotExecuteExternalOrder(t *testing.T) {
	svc, db, source, link, snapshot := newSampleFactService(t)
	sku := "SUP-SKU-1"
	detail, err := svc.CreateSourcingSample(42, source.ID, &CreateSourcingSampleInput{TaskLinkID: link.ID, SupplierID: *source.SupplierID, SnapshotID: snapshot.ID, SupplierSKU: &sku, Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	if detail.Sample.Status != SampleStatusRequest || len(detail.Events) != 1 || detail.Events[0].TruthStatus != "unknown" {
		t.Fatalf("unexpected request: %+v", detail)
	}

	steps := []TransitionSourcingSampleInput{{ToStatus: SampleStatusApproved, Note: "Owner approves budget"}}
	observed := time.Now().Add(-time.Minute)
	amount, currency, orderURI := 18.50, "cny", "https://evidence.local/order/123"
	steps = append(steps, TransitionSourcingSampleInput{ToStatus: SampleStatusOrdered, OrderAmount: &amount, Currency: &currency, ExternalCredentialURI: &orderURI, ObservedAt: &observed, TruthStatus: "actual", Note: "Owner recorded supplier order receipt"})
	receiptURI := "https://evidence.local/delivery/123"
	steps = append(steps, TransitionSourcingSampleInput{ToStatus: SampleStatusReceived, ExternalCredentialURI: &receiptURI, ObservedAt: &observed, TruthStatus: "actual", Note: "parcel received"})
	evalURI := "https://evidence.local/evaluation/123"
	steps = append(steps, TransitionSourcingSampleInput{ToStatus: SampleStatusEvaluated, ExternalCredentialURI: &evalURI, ObservedAt: &observed, TruthStatus: "actual", Note: "seams and dimensions checked"})
	steps = append(steps, TransitionSourcingSampleInput{ToStatus: SampleStatusAccepted, ExternalCredentialURI: &evalURI, ObservedAt: &observed, TruthStatus: "actual", Note: "meets approved requirements"})
	for i := range steps {
		detail, err = svc.TransitionSourcingSample(42, source.ID, detail.Sample.ID, &steps[i])
		if err != nil {
			t.Fatalf("step %s: %v", steps[i].ToStatus, err)
		}
	}
	if detail.Sample.Status != SampleStatusAccepted || detail.Sample.Currency == nil || *detail.Sample.Currency != "CNY" || len(detail.Events) != 6 {
		t.Fatalf("unexpected final detail: %+v", detail)
	}
	var persisted int64
	if err := db.Model(&SourcingSample{}).Count(&persisted).Error; err != nil || persisted != 1 {
		t.Fatalf("expected only a factual sample record, count=%d err=%v", persisted, err)
	}
}

func TestSourcingSampleTransitionsRevalidateFrozenOpportunity(t *testing.T) {
	svc, db, source, link, snapshot := newSampleFactService(t)
	detail, err := svc.CreateSourcingSample(42, source.ID, &CreateSourcingSampleInput{TaskLinkID: link.ID, SupplierID: *source.SupplierID, SnapshotID: snapshot.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&sourcingMarketDecisionRow{}).Where("id = ?", 1).Update("decision", "paused").Error; err != nil {
		t.Fatal(err)
	}
	_, err = svc.TransitionSourcingSample(42, source.ID, detail.Sample.ID, &TransitionSourcingSampleInput{ToStatus: SampleStatusApproved, Note: "should fail"})
	if !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("revoked market authority must fail closed, got %v", err)
	}
}

func TestSourcingSampleRejectsOwnerBoundaryAndSkippedOrUnprovenFacts(t *testing.T) {
	svc, _, source, link, snapshot := newSampleFactService(t)
	if _, err := svc.CreateSourcingSample(99, source.ID, &CreateSourcingSampleInput{TaskLinkID: link.ID, SupplierID: *source.SupplierID, SnapshotID: snapshot.ID}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("cross-owner create should fail, got %v", err)
	}
	detail, err := svc.CreateSourcingSample(42, source.ID, &CreateSourcingSampleInput{TaskLinkID: link.ID, SupplierID: *source.SupplierID, SnapshotID: snapshot.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TransitionSourcingSample(42, source.ID, detail.Sample.ID, &TransitionSourcingSampleInput{ToStatus: SampleStatusReceived}); !errors.Is(err, ErrInvalidLifecycleTransition) {
		t.Fatalf("skipped state should fail, got %v", err)
	}
	if _, err := svc.TransitionSourcingSample(42, source.ID, detail.Sample.ID, &TransitionSourcingSampleInput{ToStatus: SampleStatusApproved, Note: "Owner approves sample budget"}); err != nil {
		t.Fatal(err)
	}
	amount, currency := 10.0, "CNY"
	if _, err := svc.TransitionSourcingSample(42, source.ID, detail.Sample.ID, &TransitionSourcingSampleInput{ToStatus: SampleStatusOrdered, OrderAmount: &amount, Currency: &currency, TruthStatus: "actual"}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("order without external evidence must fail, got %v", err)
	}
}

func TestSourcingSampleMigrationDeclaresFrozenIdentityAndAppendOnlyEvents(t *testing.T) {
	body, err := os.ReadFile("../../../migrations/000115_sourcing_sample_fact_chain.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, required := range []string{
		"owner_id BIGINT NOT NULL", "sourcing_product_id BIGINT NOT NULL", "task_link_id BIGINT NOT NULL",
		"product_opportunity_id BIGINT NOT NULL", "opportunity_decision_id BIGINT NOT NULL",
		"supplier_id BIGINT NOT NULL", "snapshot_id BIGINT NOT NULL", "supplier_sku VARCHAR(160)",
		"'request','approved_to_order','ordered','received','evaluated','accepted','rejected'",
		"external_credential_uri TEXT", "observed_at TIMESTAMPTZ", "truth_status VARCHAR(16)",
		"sourcing_sample_event is append-only", "BEFORE UPDATE OR DELETE ON sourcing_sample_event",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration is missing invariant %q", required)
		}
	}
}

func TestDraftSampleGateRequiresAcceptedSampleOrImmutableOwnerWaiver(t *testing.T) {
	t.Run("explicit waiver", func(t *testing.T) {
		svc, db, source, link, _ := newSampleFactService(t)
		if err := requireSampleApprovalGate(db, &link); !errors.Is(err, ErrWorkflowGate) {
			t.Fatalf("missing sample unexpectedly passed: %v", err)
		}
		waived, err := svc.WaiveSourcingSample(42, source.ID, link.ID, &WaiveSourcingSampleInput{Reason: "该低风险标准品先不下样，承担尺寸偏差风险"})
		if err != nil || waived.SamplePolicy != "waived" || waived.SampleWaivedBy == nil || waived.SampleWaivedAt == nil {
			t.Fatalf("waiver=%+v err=%v", waived, err)
		}
		if err := requireSampleApprovalGate(db, waived); err != nil {
			t.Fatalf("valid waiver did not pass: %v", err)
		}
		if _, err := svc.WaiveSourcingSample(42, source.ID, link.ID, &WaiveSourcingSampleInput{Reason: "change reason"}); !errors.Is(err, ErrWorkflowGate) {
			t.Fatalf("waiver mutation was accepted: %v", err)
		}
	})

	t.Run("accepted sample", func(t *testing.T) {
		svc, db, source, link, snapshot := newSampleFactService(t)
		detail, err := svc.CreateSourcingSample(42, source.ID, &CreateSourcingSampleInput{TaskLinkID: link.ID, SupplierID: *source.SupplierID, SnapshotID: snapshot.ID})
		if err != nil {
			t.Fatal(err)
		}
		observed, amount, currency, uri := time.Now().Add(-time.Minute), 1.0, "CNY", "evidence://sample/1"
		steps := []TransitionSourcingSampleInput{
			{ToStatus: SampleStatusApproved, Note: "approve"},
			{ToStatus: SampleStatusOrdered, OrderAmount: &amount, Currency: &currency, ExternalCredentialURI: &uri, ObservedAt: &observed, TruthStatus: "actual"},
			{ToStatus: SampleStatusReceived, ExternalCredentialURI: &uri, ObservedAt: &observed, TruthStatus: "actual"},
			{ToStatus: SampleStatusEvaluated, ExternalCredentialURI: &uri, ObservedAt: &observed, TruthStatus: "actual", Note: "checked"},
			{ToStatus: SampleStatusAccepted, ExternalCredentialURI: &uri, ObservedAt: &observed, TruthStatus: "actual", Note: "accepted"},
		}
		for i := range steps {
			detail, err = svc.TransitionSourcingSample(42, source.ID, detail.Sample.ID, &steps[i])
			if err != nil {
				t.Fatalf("%s: %v", steps[i].ToStatus, err)
			}
		}
		if err := requireSampleApprovalGate(db, &link); err != nil {
			t.Fatalf("accepted sample did not pass: %v", err)
		}
	})
}
