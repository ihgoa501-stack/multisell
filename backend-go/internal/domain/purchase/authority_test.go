package purchase

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type authoritySupplierSeed struct {
	ID, OwnerID int64
	Status      int16
	TruthStatus string
}

func (authoritySupplierSeed) TableName() string { return "supplier" }

type authorityMappingSeed struct{ ID, OwnerID, InternalSKUID int64 }

func (authorityMappingSeed) TableName() string { return "sourcing_sku_mapping" }

type authorityCostSeed struct {
	ID, OwnerID, SKUMappingID int64
	TargetCurrency            string
}

func (authorityCostSeed) TableName() string { return "sourcing_cost_version" }

type authorityCostLineSeed struct {
	ID, CostVersionID     int64
	CostType              string
	NormalizedAmountMinor int64
	TruthStatus           string
}

func (authorityCostLineSeed) TableName() string { return "sourcing_cost_line" }

type authorityInventorySeed struct {
	ID, SkuID int64
	Quantity  int
}

func (authorityInventorySeed) TableName() string { return "inventory" }

type authorityDecisionSeed struct {
	ID, OwnerID                                                            int64
	Decision, CapabilityID, CommandType, TargetType, TargetID, InputSHA256 string
}

func (authorityDecisionSeed) TableName() string { return "business_owner_decision" }

func newAuthorityTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := dbtest.NewDB(t, &Authority{}, &ExternalFact{}, &InventoryReceiptLedger{}, &authoritySupplierSeed{}, &authorityMappingSeed{}, &authorityCostSeed{}, &authorityCostLineSeed{}, &authorityInventorySeed{}, &authorityDecisionSeed{})
	return NewService(db, zap.NewNop(), nil), db
}
func seedAuthority(t *testing.T, db *gorm.DB, owner int64) {
	t.Helper()
	for _, v := range []any{&authoritySupplierSeed{ID: 1, OwnerID: owner, Status: 1, TruthStatus: "quoted"}, &authorityMappingSeed{ID: 2, OwnerID: owner, InternalSKUID: 3}, &authorityCostSeed{ID: 4, OwnerID: owner, SKUMappingID: 2, TargetCurrency: "CNY"}, &authorityCostLineSeed{ID: 5, CostVersionID: 4, CostType: "purchase", NormalizedAmountMinor: 1250, TruthStatus: "quoted"}, &authorityInventorySeed{ID: 6, SkuID: 3, Quantity: 10}} {
		if e := db.Create(v).Error; e != nil {
			t.Fatal(e)
		}
	}
}

func TestPurchaseAuthorityRequiresOwnerScopedSupplierMappingCostAndInventory(t *testing.T) {
	svc, db := newAuthorityTestService(t)
	seedAuthority(t, db, 7)
	p, e := svc.CreateAuthority(context.Background(), 7, CreateAuthorityInput{SupplierID: 1, SKUMappingID: 2, CostVersionID: 4, InventoryID: 6, Quantity: 4, IdempotencyKey: "request-1"})
	if e != nil {
		t.Fatal(e)
	}
	if p.Status != AuthorityRequested || p.UnitAmountMinor != 1250 || p.TotalAmountMinor != 5000 || p.Currency != "CNY" || len(p.RequestSHA256) != 64 {
		t.Fatalf("bad purchase: %+v", p)
	}
	replay, e := svc.CreateAuthority(context.Background(), 7, CreateAuthorityInput{SupplierID: 1, SKUMappingID: 2, CostVersionID: 4, InventoryID: 6, Quantity: 4, IdempotencyKey: "request-1"})
	if e != nil || replay.ID != p.ID {
		t.Fatalf("replay: %+v %v", replay, e)
	}
	if _, e = svc.CreateAuthority(context.Background(), 8, CreateAuthorityInput{SupplierID: 1, SKUMappingID: 2, CostVersionID: 4, InventoryID: 6, Quantity: 4, IdempotencyKey: "other-owner"}); e == nil {
		t.Fatal("cross-owner authority accepted")
	}
}

func TestPurchaseAuthorityExactOwnerDecisionAndExternalReceiptDriveInventory(t *testing.T) {
	svc, db := newAuthorityTestService(t)
	seedAuthority(t, db, 7)
	ctx := context.Background()
	p, e := svc.CreateAuthority(ctx, 7, CreateAuthorityInput{SupplierID: 1, SKUMappingID: 2, CostVersionID: 4, InventoryID: 6, Quantity: 4, IdempotencyKey: "request-2"})
	if e != nil {
		t.Fatal(e)
	}
	bad := authorityDecisionSeed{ID: 10, OwnerID: 7, Decision: "selected", CapabilityID: "purchase.authority.execute", CommandType: "purchase.submit", TargetType: "purchase_authority", TargetID: strconv.FormatInt(p.ID, 10), InputSHA256: "bad"}
	if e = db.Create(&bad).Error; e != nil {
		t.Fatal(e)
	}
	if _, e = svc.ApproveAuthority(ctx, 7, p.ID, ApproveAuthorityInput{OwnerDecisionID: 10}); e == nil {
		t.Fatal("mismatched decision accepted")
	}
	good := authorityDecisionSeed{ID: 11, OwnerID: 7, Decision: "selected", CapabilityID: "purchase.authority.execute", CommandType: "purchase.submit", TargetType: "purchase_authority", TargetID: strconv.FormatInt(p.ID, 10), InputSHA256: p.RequestSHA256}
	if e = db.Create(&good).Error; e != nil {
		t.Fatal(e)
	}
	approved, e := svc.ApproveAuthority(ctx, 7, p.ID, ApproveAuthorityInput{OwnerDecisionID: 11})
	if e != nil || approved.Status != AuthorityOwnerApproved {
		t.Fatalf("approve: %+v %v", approved, e)
	}
	now := time.Now().UTC()
	fact := func(event, ext string, qty int) ExternalFactInput {
		return ExternalFactInput{ExternalEventID: event, ExternalOrderID: "SUP-ORDER-1", ReceivedQuantity: qty, RawPayload: json.RawMessage(`{"receipt":"` + ext + `"}`), ObservedAt: now}
	}
	if _, e = svc.RecordExternalFact(ctx, 7, p.ID, "received", fact("too-early", "early", 1)); e == nil {
		t.Fatal("internal approval was treated as receipt")
	}
	d, e := svc.RecordExternalFact(ctx, 7, p.ID, "submitted", fact("submit-1", "submitted", 0))
	if e != nil || d.Authority.Status != AuthorityExternalSubmitted {
		t.Fatalf("submit: %+v %v", d, e)
	}
	if _, e = svc.RecordExternalFact(ctx, 7, p.ID, "ordered", fact("order-1", "ordered", 0)); e != nil {
		t.Fatal(e)
	}
	d, e = svc.RecordExternalFact(ctx, 7, p.ID, "received", fact("receive-1", "partial", 2))
	if e != nil || d.Authority.Status != AuthorityPartiallyReceived || d.Authority.ReceivedQuantity != 2 || len(d.Ledger) != 1 {
		t.Fatalf("partial: %+v %v", d, e)
	}
	var inv authorityInventorySeed
	db.First(&inv, 6)
	if inv.Quantity != 12 {
		t.Fatalf("inventory=%d", inv.Quantity)
	}
	replay, e := svc.RecordExternalFact(ctx, 7, p.ID, "received", fact("receive-1", "partial", 2))
	if e != nil || len(replay.Ledger) != 1 {
		t.Fatalf("receipt replay duplicated: %+v %v", replay, e)
	}
	db.First(&inv, 6)
	if inv.Quantity != 12 {
		t.Fatalf("replay inventory=%d", inv.Quantity)
	}
	d, e = svc.RecordExternalFact(ctx, 7, p.ID, "received", fact("receive-2", "full", 2))
	if e != nil || d.Authority.Status != AuthorityFullyReceived || d.Authority.ReceivedQuantity != 4 || len(d.Ledger) != 2 {
		t.Fatalf("full: %+v %v", d, e)
	}
	db.First(&inv, 6)
	if inv.Quantity != 14 {
		t.Fatalf("final inventory=%d", inv.Quantity)
	}
}

func TestPurchaseAuthorityFailedReceiptIsTerminalAndExternalIdentityCannotChange(t *testing.T) {
	svc, db := newAuthorityTestService(t)
	seedAuthority(t, db, 7)
	ctx := context.Background()
	p, _ := svc.CreateAuthority(ctx, 7, CreateAuthorityInput{SupplierID: 1, SKUMappingID: 2, CostVersionID: 4, InventoryID: 6, Quantity: 1, IdempotencyKey: "request-3"})
	d := authorityDecisionSeed{ID: 12, OwnerID: 7, Decision: "selected", CapabilityID: "purchase.authority.execute", CommandType: "purchase.submit", TargetType: "purchase_authority", TargetID: strconv.FormatInt(p.ID, 10), InputSHA256: p.RequestSHA256}
	db.Create(&d)
	svc.ApproveAuthority(ctx, 7, p.ID, ApproveAuthorityInput{OwnerDecisionID: 12})
	now := time.Now().UTC()
	base := ExternalFactInput{ExternalEventID: "s", ExternalOrderID: "X", RawPayload: json.RawMessage(`{"s":1}`), ObservedAt: now}
	svc.RecordExternalFact(ctx, 7, p.ID, "submitted", base)
	changed := ExternalFactInput{ExternalEventID: "o", ExternalOrderID: "Y", RawPayload: json.RawMessage(`{"o":1}`), ObservedAt: now}
	if _, e := svc.RecordExternalFact(ctx, 7, p.ID, "ordered", changed); e == nil {
		t.Fatal("external order identity changed")
	}
	failed := ExternalFactInput{ExternalEventID: "f", ExternalOrderID: "X", RawPayload: json.RawMessage(`{"f":1}`), ObservedAt: now}
	got, e := svc.RecordExternalFact(ctx, 7, p.ID, "failed", failed)
	if e != nil || got.Authority.Status != AuthorityFailed {
		t.Fatalf("failed: %+v %v", got, e)
	}
}
