package profit

import (
	"context"
	"errors"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newFinalProfitTestService(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &finalOrderAuthority{}, &finalOrderItem{}, &finalCostVersion{}, &finalCostLine{}, &finalSKUMapping{}, &finalResolution{}, &finalReceipt{}, &finalSettlementIngest{}, &finalSettlementLine{}, &OrderProductCostAllocation{}, &OrderFinalProfitVersion{})
	return NewService(db, zap.NewNop(), nil, 7.2)
}

func seedFinalProfit(t *testing.T, s *Service, quotedCost, omitFulfillment, unresolvedRefund bool) {
	t.Helper()
	db := s.db
	must := func(v any) {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	must(&finalOrderAuthority{ID: 1, OwnerID: 7, NormalizedOrderID: 10, EventAction: "reserve", TruthStatus: "external_observed", ProcessingStatus: "applied"})
	must(&finalOrderItem{ID: 20, OrderID: 10, SkuID: 30, Quantity: 2})
	must(&finalSKUMapping{ID: 40, OwnerID: 7, InternalSKUID: 30})
	must(&finalCostVersion{ID: 50, OwnerID: 7, SKUMappingID: 40, TotalMinor: 1200, TargetCurrency: "USD"})
	truth := "actual"
	if quotedCost {
		truth = "quoted"
	}
	must(&finalCostLine{CostVersionID: 50, TruthStatus: truth})
	must(&finalSettlementIngest{ID: 60, OwnerID: 7, TruthStatus: "external_observed", Currency: "USD", ContentSHA256: "settlement-hash"})
	must(&finalSettlementLine{ID: 61, IngestID: 60, OrderID: 10, AmountMinor: 10000, Kind: "sale", Currency: "USD"})
	must(&finalSettlementLine{ID: 62, IngestID: 60, OrderID: 10, AmountMinor: 800, Kind: "commission", FeeCode: "platform_fee", Currency: "USD"})
	if !omitFulfillment {
		must(&finalSettlementLine{ID: 63, IngestID: 60, OrderID: 10, AmountMinor: 900, Kind: "fee", FeeCode: "fulfillment_fee", Currency: "USD"})
	}
	if unresolvedRefund {
		must(&finalResolution{ID: 70, OwnerID: 7, OrderID: 10, Status: "approved", Currency: "USD"})
	}
}

func TestFinalProfitAuthorityHappyPathIsExactImmutableAndIdempotent(t *testing.T) {
	s := newFinalProfitTestService(t)
	seedFinalProfit(t, s, false, false, false)
	a, err := s.AllocateOrderProductCost(context.Background(), 7, 10, AllocateOrderProductCostInput{OrderItemID: 20, SourcingCostVersionID: 50})
	if err != nil || a.AmountMinor != 2400 || a.Currency != "USD" {
		t.Fatalf("allocation=%+v err=%v", a, err)
	}
	got, err := s.FinalizeOrderProfit(context.Background(), 7, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.RevenueMinor != 10000 || got.ProductCostMinor != 2400 || got.SettlementFeeMinor != 800 || got.FulfillmentFeeMinor != 900 || got.RefundMinor != 0 || got.TotalCostMinor != 4100 || got.ProfitMinor != 5900 || got.Currency != "USD" || got.Version != 1 {
		t.Fatalf("unexpected final profit: %+v", got)
	}
	replay, err := s.FinalizeOrderProfit(context.Background(), 7, 10)
	if err != nil || replay.ID != got.ID || replay.Version != 1 {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
}

func TestFinalProfitAuthorityFailsClosedForMissingOrWeakEvidence(t *testing.T) {
	tests := []struct {
		name                                string
		quoted, omitFulfillment, unresolved bool
	}{
		{"quoted product cost", true, false, false}, {"missing fulfillment", false, true, false}, {"unresolved aftersales", false, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newFinalProfitTestService(t)
			seedFinalProfit(t, s, tc.quoted, tc.omitFulfillment, tc.unresolved)
			_, allocErr := s.AllocateOrderProductCost(context.Background(), 7, 10, AllocateOrderProductCostInput{OrderItemID: 20, SourcingCostVersionID: 50})
			if tc.quoted {
				if !errors.Is(allocErr, ErrFinalProfitUnknown) {
					t.Fatalf("expected unknown, got %v", allocErr)
				}
				return
			}
			if allocErr != nil {
				t.Fatal(allocErr)
			}
			_, err := s.FinalizeOrderProfit(context.Background(), 7, 10)
			if !errors.Is(err, ErrFinalProfitUnknown) {
				t.Fatalf("expected fail closed unknown, got %v", err)
			}
		})
	}
}

func TestFinalProfitAuthorityRejectsCrossOwnerAndRefundMismatch(t *testing.T) {
	s := newFinalProfitTestService(t)
	seedFinalProfit(t, s, false, false, false)
	if _, err := s.AllocateOrderProductCost(context.Background(), 8, 10, AllocateOrderProductCostInput{OrderItemID: 20, SourcingCostVersionID: 50}); !errors.Is(err, ErrFinalProfitUnknown) {
		t.Fatalf("cross owner error=%v", err)
	}
	if _, err := s.AllocateOrderProductCost(context.Background(), 7, 10, AllocateOrderProductCostInput{OrderItemID: 20, SourcingCostVersionID: 50}); err != nil {
		t.Fatal(err)
	}
	s.db.Create(&finalResolution{ID: 70, OwnerID: 7, OrderID: 10, Status: "succeeded", Currency: "USD"})
	s.db.Create(&finalReceipt{ResolutionID: 70, OwnerID: 7, ActualMinor: 500, Outcome: "succeeded", Currency: "USD", SourceType: "platform_receipt"})
	if _, err := s.FinalizeOrderProfit(context.Background(), 7, 10); !errors.Is(err, ErrFinalProfitUnknown) {
		t.Fatalf("refund mismatch error=%v", err)
	}
}
