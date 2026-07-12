package demandcase

import (
	"context"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func comparisonService(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &DemandCase{}, &DemandEvidence{}, &DemandVerdict{}, &ResearchBatch{}, &ResearchSnapshot{}, &MarketOwnerDecision{})
	return NewService(db, zap.NewNop())
}

func TestCompareKeepsDimensionsCounterevidenceAndUnknownsInOneOwnerView(t *testing.T) {
	s, ctx := comparisonService(t), context.Background()
	cases := []*DemandCase{
		{OwnerID: 7, Region: "DE", Consumer: "养猫家庭", NeedScenario: "出行饮水", SalesChannel: "独立站", TargetLocale: "de-DE", StopCondition: "费用未知时停止"},
		{OwnerID: 7, Region: "FR", Consumer: "养猫家庭", NeedScenario: "出行饮水", SalesChannel: "marketplace", TargetLocale: "fr-FR", StopCondition: "权限未知时停止"},
	}
	now := time.Now().UTC()
	for _, c := range cases {
		if err := s.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
		if err := s.db.Create(&DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionDemand, Kind: EvidenceCounter, TruthStatus: TruthQuoted, Title: "已有低价替代品", SourceURI: "https://example.test/counter", ObservedAt: &now, RunID: "counter", SnapshotID: 1}).Error; err != nil {
			t.Fatal(err)
		}
		if err := s.db.Create(&DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionPayment, Kind: EvidenceConflict, TruthStatus: TruthUnknown, Title: "收款权限未知", RunID: "reality", SnapshotID: 2}).Error; err != nil {
			t.Fatal(err)
		}
	}
	view, err := s.Compare(ctx, 7, []int64{cases[0].ID, cases[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Dimensions) != 8 || len(view.Candidates) != 2 {
		t.Fatalf("view=%+v", view)
	}
	if view.Candidates[0].StrongestCounterevidence != "已有低价替代品" || len(view.Candidates[0].Unknowns) == 0 {
		t.Fatalf("candidate=%+v", view.Candidates[0])
	}
}

func TestCompareRejectsDuplicatesAndCrossOwnerCases(t *testing.T) {
	s, ctx := comparisonService(t), context.Background()
	a := &DemandCase{OwnerID: 7, Region: "DE", Consumer: "A", NeedScenario: "A", SalesChannel: "A", TargetLocale: "de-DE"}
	b := &DemandCase{OwnerID: 8, Region: "FR", Consumer: "B", NeedScenario: "B", SalesChannel: "B", TargetLocale: "fr-FR"}
	_ = s.Create(ctx, a)
	_ = s.Create(ctx, b)
	if _, err := s.Compare(ctx, 7, []int64{a.ID, a.ID}); err == nil {
		t.Fatal("duplicate comparison ids accepted")
	}
	if _, err := s.Compare(ctx, 7, []int64{a.ID, b.ID}); err == nil {
		t.Fatal("cross-owner comparison accepted")
	}
}
