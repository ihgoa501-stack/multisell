package sourcing1688

import (
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
)

func completePreciseCostInput(link Sourcing1688TaskLink, snapshotID, mappingID int64) CreateSourcingCostVersionInput {
	now := time.Now().Add(-time.Hour).UTC()
	lines := make([]CreateSourcingCostLineInput, 0, len(requiredCostTypes))
	for i, typ := range requiredCostTypes {
		amount := int64((i + 1) * 100)
		lines = append(lines, CreateSourcingCostLineInput{CostType: typ, AmountMinor: amount, Currency: "USD", NormalizedAmountMinor: amount, TruthStatus: "quoted", SourceURI: "https://evidence.local/cost/" + typ, ObservedAt: now})
	}
	return CreateSourcingCostVersionInput{TaskLinkID: link.ID, SourceSnapshotID: snapshotID, SKUMappingID: mappingID, TargetCurrency: "usd", RevenueMinor: 1500, PricingBasis: "owner_confirmed_listing_price", QuantityTierMin: 1, PurchaseLineOwnerConfirmed: true, Lines: lines}
}

func TestPreciseCostValidationRejectsIncompleteAndInexactConversion(t *testing.T) {
	in := completePreciseCostInput(Sourcing1688TaskLink{ID: 1}, 1, 1)
	in.Lines = in.Lines[:9]
	if _, _, err := validatePreciseCostInput(&in); err == nil {
		t.Fatal("incomplete cost set must fail")
	}
	in = completePreciseCostInput(Sourcing1688TaskLink{ID: 1}, 1, 1)
	rate, rateURI, observed := "0.14", "https://evidence.local/fx", time.Now().Add(-time.Hour)
	in.Lines[0].Currency, in.Lines[0].AmountMinor, in.Lines[0].NormalizedAmountMinor = "CNY", 101, 15
	in.Lines[0].ExchangeRateDecimal, in.Lines[0].ExchangeRateSourceURI, in.Lines[0].ExchangeRateObservedAt = &rate, &rateURI, &observed
	if _, _, err := validatePreciseCostInput(&in); err == nil {
		t.Fatal("rounded or inexact conversion must fail closed")
	}
}

func TestPreciseCostUsesHalfUpMinorRoundingAndRequiresOwnerPurchaseConfirmation(t *testing.T) {
	for _, tc := range []struct{ numerator, denominator, want int64 }{{1, 2, 1}, {3, 2, 2}, {149, 100, 1}, {150, 100, 2}} {
		got, ok := roundPositiveRatHalfUp(new(big.Rat).SetFrac64(tc.numerator, tc.denominator))
		if !ok || got != tc.want {
			t.Fatalf("round %d/%d = %d,%v want %d", tc.numerator, tc.denominator, got, ok, tc.want)
		}
	}
	in := completePreciseCostInput(Sourcing1688TaskLink{ID: 1}, 1, 1)
	in.PurchaseLineOwnerConfirmed = false
	if _, _, err := validatePreciseCostInput(&in); err == nil {
		t.Fatal("quoted purchase candidate without Owner confirmation must fail")
	}
}

func TestCreatePreciseCostVersionFreezesAuthorityAndVersions(t *testing.T) {
	svc, db, source, link, snapshot := newSampleFactService(t)
	if err := db.AutoMigrate(&SourcingSKUMapping{}, &SourcingCostVersion{}, &SourcingCostLine{}); err != nil {
		t.Fatal(err)
	}
	mapping := SourcingSKUMapping{OwnerID: 42, SourcingProductID: source.ID, TaskLinkID: link.ID, SnapshotID: snapshot.ID, ProductOpportunityID: *link.ProductOpportunityID, ProductID: 70, SupplierSKU: "SUP-1", InternalSKUID: 71, InternalSKU: "INT-1", ChannelSKU: "CH-1", PlatformID: 72, ListingID: 73, Version: 1, ContentHash: "mapping", CreatedBy: 42}
	if err := db.Create(&mapping).Error; err != nil {
		t.Fatal(err)
	}
	mappings, err := svc.ListCanonicalSKUMappings(42, source.ID, link.ID)
	if err != nil || len(mappings) != 1 || mappings[0].ID != mapping.ID {
		t.Fatalf("exact task mapping list mismatch: %+v %v", mappings, err)
	}
	if _, err := svc.ListCanonicalSKUMappings(99, source.ID, link.ID); err == nil {
		t.Fatal("cross-owner canonical SKU mapping read must fail closed")
	}
	in := completePreciseCostInput(link, snapshot.ID, mapping.ID)
	first, err := svc.CreateSourcingCostVersion(42, source.ID, &in)
	if err != nil {
		t.Fatal(err)
	}
	if first.Version.Version != 1 || first.Version.TotalMinor != 5500 || len(first.Lines) != 10 || first.Version.ContentHash == "" {
		t.Fatalf("unexpected first version: %+v", first)
	}
	if first.Version.ContributionProfitMinor != -4000 {
		t.Fatalf("negative contribution profit math = %d", first.Version.ContributionProfitMinor)
	}
	replay, err := svc.CreateSourcingCostVersion(42, source.ID, &in)
	if err != nil || replay.Version.ID != first.Version.ID || replay.Version.Version != 1 {
		t.Fatalf("idempotent replay = %+v, %v", replay, err)
	}
	in.Lines[0].AmountMinor, in.Lines[0].NormalizedAmountMinor = 101, 101
	second, err := svc.CreateSourcingCostVersion(42, source.ID, &in)
	if err != nil {
		t.Fatal(err)
	}
	if second.Version.Version != 2 || second.Version.TotalMinor != 5501 || second.Version.ContentHash == first.Version.ContentHash {
		t.Fatalf("version did not preserve distinct facts: %+v", second)
	}
	items, err := svc.ListSourcingCostVersions(42, source.ID)
	if err != nil || len(items) != 2 || items[0].Version.Version != 2 {
		t.Fatalf("unexpected list: %+v %v", items, err)
	}
	if _, err := svc.ListSourcingCostVersions(99, source.ID); err == nil {
		t.Fatal("cross-owner read must fail closed")
	}
}

func TestFrozenDraftCostRejectsApprovalAndCostHashTampering(t *testing.T) {
	db := dbtest.NewDB(t, &SourcingCostVersion{})
	version := SourcingCostVersion{OwnerID: 42, SourcingProductID: 7, TaskLinkID: 8, ProductOpportunityID: 9, OpportunityDecisionID: 10, SourceSnapshotID: 11, SKUMappingID: 12, Version: 1, TargetCurrency: "USD", TotalMinor: 100, RevenueMinor: 120, ContributionProfitMinor: 20, PricingBasis: "owner_confirmed_listing_price", QuantityTierMin: 1, PurchaseLineOwnerConfirmed: true, ContentHash: strings.Repeat("a", 64), CreatedBy: 42}
	if err := db.Create(&version).Error; err != nil {
		t.Fatal(err)
	}
	taskID := int64(8)
	draft := draftRow{ID: 1, SourcingProductID: 7, TaskLinkID: &taskID, CostVersionID: &version.ID, CostVersionContentHash: version.ContentHash}
	value, _ := json.Marshal(draftApprovalNewValue{ContentSHA256: strings.Repeat("c", 64), CostVersionID: version.ID, CostVersionContentHash: version.ContentHash})
	req := approval.ApprovalRequest{NewValue: string(value)}
	if err := validateFrozenDraftCost(db, &draft, &req); err != nil {
		t.Fatalf("valid frozen cost = %v", err)
	}
	req.NewValue = `{"content_sha256":"` + strings.Repeat("c", 64) + `","cost_version_id":999,"cost_version_content_hash":"` + version.ContentHash + `"}`
	if err := validateFrozenDraftCost(db, &draft, &req); err == nil {
		t.Fatal("tampered approval cost id must fail")
	}
	req.NewValue = string(value)
	draft.CostVersionContentHash = strings.Repeat("b", 64)
	if err := validateFrozenDraftCost(db, &draft, &req); err == nil {
		t.Fatal("tampered draft cost hash must fail")
	}
}

func TestCostDraftAuthorityMigrationFreezesExactVersion(t *testing.T) {
	body, err := os.ReadFile("../../../migrations/000149_sourcing_cost_draft_authority.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, required := range []string{"revenue_minor", "contribution_profit_minor = revenue_minor - total_minor", "purchase_line_owner_confirmed", "cost_version_id", "draft approval requires an exact precise cost version", "approved or pending draft cost authority is immutable"} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration missing %q", required)
		}
	}
}
