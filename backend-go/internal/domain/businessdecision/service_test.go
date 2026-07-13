package businessdecision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type purchaseAuthorityFact struct {
	ID, OwnerID, SupplierID, SKUMappingID, InternalSKUID, CostVersionID, InventoryID int64
	Quantity                                                                         int
	UnitAmountMinor, TotalAmountMinor                                                int64
	Currency, Status, RequestSHA256                                                  string
	CreatedAt                                                                        time.Time
}

func (purchaseAuthorityFact) TableName() string { return "purchase_authority" }

type orderIngest struct {
	ID                int64 `gorm:"primaryKey"`
	OwnerID           int64
	AccountID         int64
	PlatformCode      string
	ExternalEventID   string
	ExternalOrderID   string
	EventAction       string
	TruthStatus       string
	PayloadSHA256     string
	ObservedAt        time.Time
	NormalizedOrderID *int64
	ProcessingStatus  string
}

func (orderIngest) TableName() string { return "platform_order_ingest" }
func setup(t *testing.T) (*Service, context.Context) {
	db := dbtest.NewDB(t, &orderIngest{}, &Case{}, &FactSnapshot{}, &AIRecommendation{}, &OwnerDecision{})
	f := orderIngest{OwnerID: 7, AccountID: 3, PlatformCode: "ozon", ExternalEventID: "evt", ExternalOrderID: "ord", EventAction: "reserve", TruthStatus: "external_observed", PayloadSHA256: string(make([]byte, 64)), ObservedAt: time.Now().UTC(), ProcessingStatus: "applied"}
	if e := db.Create(&f).Error; e != nil {
		t.Fatal(e)
	}
	return NewService(db), context.Background()
}
func TestFrozenFactAIAndOwnerDecisionAreSeparated(t *testing.T) {
	s, ctx := setup(t)
	c, e := s.CreateCase(ctx, 7, CreateCaseInput{Question: "是否补货", Target: "减少缺货", ObjectType: "platform_order_ingest", ObjectID: 1, Unknowns: []string{"需求持续性未知"}, IdempotencyKey: "case-1"})
	if e != nil {
		t.Fatal(e)
	}
	if c.TruthStatus != "external_observed" || len(c.ManifestSHA256) != 64 {
		t.Fatalf("bad case %#v", c)
	}
	r, e := s.Recommend(ctx, 7, c.ID, RecommendInput{Recommendation: "小批补货", Rationale: "已有订单", TruthStatus: "inferred", Unknowns: []string{"未来需求"}, IdempotencyKey: "ai-1"})
	if e != nil {
		t.Fatal(e)
	}
	payload := json.RawMessage(`{"price":100}`)
	canonical, _ := canonicalJSON(payload)
	h := sha256.Sum256(canonical)
	payloadSHA := hex.EncodeToString(h[:])
	d, e := s.Decide(ctx, 7, c.ID, DecideInput{RecommendationID: &r.ID, Decision: DecisionSelected, CapabilityID: "command.price_update.v1", CommandType: "price_update", TargetType: "sku", TargetID: "1", InputSHA256: payloadSHA, InputPayload: payload, Reason: "Owner批准受控后续研究，不自动采购", ManifestSHA256: c.ManifestSHA256, IdempotencyKey: "owner-1"})
	if e != nil {
		t.Fatal(e)
	}
	if d.Decision != DecisionSelected {
		t.Fatal(d)
	}
	detail, e := s.Get(ctx, 7, c.ID)
	if e != nil {
		t.Fatal(e)
	}
	if len(detail.Recommendations) != 1 || len(detail.Decisions) != 1 || len(detail.Case.Unknowns) != 1 {
		t.Fatalf("bad detail %#v", detail)
	}
	if len(detail.Recommendations[0].Unknowns) != 1 || detail.Recommendations[0].Unknowns[0] != "未来需求" {
		t.Fatalf("recommendation unknowns were not restored: %#v", detail.Recommendations[0].Unknowns)
	}
	listed, e := s.List(ctx, 7)
	if e != nil || len(listed) != 1 || listed[0].LatestDecision == nil || listed[0].LatestDecision.ID != d.ID {
		t.Fatalf("Owner list cannot restore latest decision: %#v %v", listed, e)
	}
	other, e := s.List(ctx, 8)
	if e != nil || len(other) != 0 {
		t.Fatalf("cross-Owner cases leaked: %#v %v", other, e)
	}
}
func TestIdempotencyConflictOwnerIsolationAndFrozenManifest(t *testing.T) {
	s, ctx := setup(t)
	in := CreateCaseInput{Question: "Q", Target: "T", ObjectType: "platform_order_ingest", ObjectID: 1, IdempotencyKey: "same"}
	c, e := s.CreateCase(ctx, 7, in)
	if e != nil {
		t.Fatal(e)
	}
	in.Question = "changed"
	if _, e = s.CreateCase(ctx, 7, in); !errors.Is(e, ErrConflict) {
		t.Fatalf("want conflict %v", e)
	}
	if _, e = s.CreateCase(ctx, 8, CreateCaseInput{Question: "Q", Target: "T", ObjectType: "platform_order_ingest", ObjectID: 1, IdempotencyKey: "other"}); e == nil {
		t.Fatal("cross owner fact leaked")
	}
	if _, e = s.Decide(ctx, 7, c.ID, DecideInput{Decision: DecisionPaused, Reason: "wait", ManifestSHA256: string(make([]byte, 64)), IdempotencyKey: "d"}); !errors.Is(e, ErrConflict) {
		t.Fatalf("want stale conflict %v", e)
	}
}
func TestAIAdviceCannotClaimActualAndAllOwnerStatuses(t *testing.T) {
	s, ctx := setup(t)
	c, _ := s.CreateCase(ctx, 7, CreateCaseInput{Question: "Q", Target: "T", ObjectType: "platform_order_ingest", ObjectID: 1, IdempotencyKey: "c"})
	if _, e := s.Recommend(ctx, 7, c.ID, RecommendInput{Recommendation: "x", Rationale: "x", TruthStatus: "actual", IdempotencyKey: "r"}); !errors.Is(e, ErrInvalid) {
		t.Fatal(e)
	}
	if _, e := s.Decide(ctx, 7, c.ID, DecideInput{Decision: DecisionSelected, Reason: "owner", ManifestSHA256: c.ManifestSHA256, IdempotencyKey: "missing-action"}); !errors.Is(e, ErrInvalid) {
		t.Fatalf("selected without exact action must fail: %v", e)
	}
	if _, e := s.Decide(ctx, 7, c.ID, DecideInput{Decision: DecisionPaused, CapabilityID: "command.price_update.v1", Reason: "owner", ManifestSHA256: c.ManifestSHA256, IdempotencyKey: "paused-with-action"}); !errors.Is(e, ErrInvalid) {
		t.Fatalf("non-selected decision must not carry action authority: %v", e)
	}
	for i, v := range []string{DecisionSelected, DecisionRejected, DecisionPaused, DecisionMoreEvidence} {
		in := DecideInput{Decision: v, Reason: "owner", ManifestSHA256: c.ManifestSHA256, IdempotencyKey: "d" + string(rune('0'+i))}
		if v == DecisionSelected {
			in.InputPayload = json.RawMessage(`{}`)
			canonical, _ := canonicalJSON(in.InputPayload)
			h := sha256.Sum256(canonical)
			in.CapabilityID, in.CommandType, in.TargetType, in.TargetID, in.InputSHA256 = "command.price_update.v1", "price_update", "sku", "1", hex.EncodeToString(h[:])
		}
		if _, e := s.Decide(ctx, 7, c.ID, in); e != nil {
			t.Fatalf("%s: %v", v, e)
		}
	}
}

func TestPurchaseAuthorityCanBecomeExactOwnerDecisionCase(t *testing.T) {
	db := dbtest.NewDB(t, &purchaseAuthorityFact{}, &Case{}, &FactSnapshot{}, &AIRecommendation{}, &OwnerDecision{})
	p := purchaseAuthorityFact{OwnerID: 7, SupplierID: 2, SKUMappingID: 3, InternalSKUID: 4, CostVersionID: 5, InventoryID: 6, Quantity: 8, UnitAmountMinor: 1250, TotalAmountMinor: 10000, Currency: "CNY", Status: "requested", RequestSHA256: strings.Repeat("b", 64), CreatedAt: time.Now().UTC()}
	if e := db.Create(&p).Error; e != nil {
		t.Fatal(e)
	}
	s := NewService(db)
	c, e := s.CreateCase(context.Background(), 7, CreateCaseInput{Question: "是否执行本次采购", Target: "按冻结成本补货", ObjectType: "purchase_authority", ObjectID: p.ID, IdempotencyKey: "purchase-case"})
	if e != nil {
		t.Fatal(e)
	}
	if c.TruthStatus != "actual" {
		t.Fatalf("truth=%s", c.TruthStatus)
	}
	d, e := s.Decide(context.Background(), 7, c.ID, DecideInput{Decision: DecisionSelected, CapabilityID: "purchase.authority.execute", CommandType: "purchase.submit", TargetType: "purchase_authority", TargetID: strconv.FormatInt(p.ID, 10), InputSHA256: p.RequestSHA256, Reason: "Owner按冻结对象批准", ManifestSHA256: c.ManifestSHA256, IdempotencyKey: "purchase-decision"})
	if e != nil || d.TargetID != strconv.FormatInt(p.ID, 10) {
		t.Fatalf("decision=%+v err=%v", d, e)
	}
	if _, e = s.CreateCase(context.Background(), 8, CreateCaseInput{Question: "Q", Target: "T", ObjectType: "purchase_authority", ObjectID: p.ID, IdempotencyKey: "cross-owner"}); e == nil {
		t.Fatal("cross-owner purchase fact leaked")
	}
}
