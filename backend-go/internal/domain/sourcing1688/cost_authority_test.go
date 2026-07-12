package sourcing1688

import (
	"testing"
	"time"
)

func completePreciseCostInput(link Sourcing1688TaskLink, snapshotID, mappingID int64) CreateSourcingCostVersionInput {
	now := time.Now().Add(-time.Hour).UTC()
	lines := make([]CreateSourcingCostLineInput, 0, len(requiredCostTypes))
	for i, typ := range requiredCostTypes {
		amount := int64((i + 1) * 100)
		lines = append(lines, CreateSourcingCostLineInput{CostType: typ, AmountMinor: amount, Currency: "USD", NormalizedAmountMinor: amount, TruthStatus: "quoted", SourceURI: "https://evidence.local/cost/" + typ, ObservedAt: now})
	}
	return CreateSourcingCostVersionInput{TaskLinkID: link.ID, SourceSnapshotID: snapshotID, SKUMappingID: mappingID, TargetCurrency: "usd", Lines: lines}
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

func TestCreatePreciseCostVersionFreezesAuthorityAndVersions(t *testing.T) {
	svc, db, source, link, snapshot := newSampleFactService(t)
	if err := db.AutoMigrate(&SourcingSKUMapping{}, &SourcingCostVersion{}, &SourcingCostLine{}); err != nil {
		t.Fatal(err)
	}
	mapping := SourcingSKUMapping{OwnerID: 42, SourcingProductID: source.ID, TaskLinkID: link.ID, SnapshotID: snapshot.ID, ProductOpportunityID: *link.ProductOpportunityID, ProductID: 70, SupplierSKU: "SUP-1", InternalSKUID: 71, InternalSKU: "INT-1", ChannelSKU: "CH-1", PlatformID: 72, ListingID: 73, Version: 1, ContentHash: "mapping", CreatedBy: 42}
	if err := db.Create(&mapping).Error; err != nil {
		t.Fatal(err)
	}
	in := completePreciseCostInput(link, snapshot.ID, mapping.ID)
	first, err := svc.CreateSourcingCostVersion(42, source.ID, &in)
	if err != nil {
		t.Fatal(err)
	}
	if first.Version.Version != 1 || first.Version.TotalMinor != 5500 || len(first.Lines) != 10 || first.Version.ContentHash == "" {
		t.Fatalf("unexpected first version: %+v", first)
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
