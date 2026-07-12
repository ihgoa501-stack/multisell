package sourcing1688

import (
	"errors"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func seedApprovedSourcingAuthority(t *testing.T, db *gorm.DB, sourceID, ownerID, demandCaseID int64, experimentID, channel string) {
	t.Helper()
	market := sourcingMarketDecisionRow{DemandCaseID: demandCaseID, OwnerID: ownerID, Decision: "selected"}
	if err := db.Create(&market).Error; err != nil {
		t.Fatal(err)
	}
	opportunity := sourcingOpportunityRow{OwnerID: ownerID, DemandCaseID: demandCaseID, MarketDecisionID: market.ID, Version: 1, Title: "approved test opportunity", TargetChannel: channel, Status: "approved", ContentHash: "test-opportunity-content"}
	if err := db.Create(&opportunity).Error; err != nil {
		t.Fatal(err)
	}
	decision := sourcingOpportunityDecisionRow{OpportunityID: opportunity.ID, OwnerID: ownerID, Version: opportunity.Version, Decision: "approved", ContentHash: opportunity.ContentHash}
	if err := db.Create(&decision).Error; err != nil {
		t.Fatal(err)
	}
	link := Sourcing1688TaskLink{SourcingProductID: sourceID, DemandCaseID: demandCaseID, ExperimentID: experimentID, OwnerID: ownerID, ProductOpportunityID: &opportunity.ID, OpportunityDecisionID: &decision.ID, AuthorityKind: "product_opportunity", Status: "linked", IsPrimary: true}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
}

func TestReviewRequiresFrozenApprovedProductOpportunity(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688TaskLink{}, &sourcingOpportunityRow{}, &sourcingOpportunityDecisionRow{}, &sourcingMarketDecisionRow{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	caseID := int64(7)
	experimentID := "EXP-TRACE-ONLY"
	snapshotID := int64(9)
	db.Create(&demandCaseRow{ID: caseID, OwnerID: 42, SalesChannel: "Ozon"})
	source := Sourcing1688Product{OwnerID: 42, DemandCaseID: &caseID, ExperimentID: &experimentID, SnapshotID: &snapshotID, Status: StatusPendingReview}
	db.Create(&source)

	if _, err := svc.Review(source.ID, &ReviewInput{ReviewedBy: 42, Notes: "reviewed"}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("experiment trace without opportunity authority must be blocked, got %v", err)
	}
	seedApprovedSourcingAuthority(t, db, source.ID, 42, caseID, experimentID, "Ozon")
	if _, err := svc.Review(source.ID, &ReviewInput{ReviewedBy: 42, Notes: "reviewed"}); err != nil {
		t.Fatalf("frozen approved opportunity should authorize review: %v", err)
	}
}
