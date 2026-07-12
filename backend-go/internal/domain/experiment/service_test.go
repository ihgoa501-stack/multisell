package experiment

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	financeDomain "github.com/lingmirror/backend-go/internal/domain/finance"
	orderDomain "github.com/lingmirror/backend-go/internal/domain/order"
	profitDomain "github.com/lingmirror/backend-go/internal/domain/profit"
	settlementDomain "github.com/lingmirror/backend-go/internal/domain/settlement"
	"gorm.io/gorm"
)

func testService(t *testing.T) *Service {
	db := dbtest.NewDB(t, &ExperimentCase{}, &GateDecision{}, &EvidenceRecord{}, &ObjectLink{})
	return NewService(db, dbtest.NewLogger(t))
}

func closureService(t *testing.T) *Service {
	db := dbtest.NewDB(t, &ExperimentCase{}, &GateDecision{}, &EvidenceRecord{}, &ObjectLink{}, &orderDomain.Order{}, &settlementDomain.Settlement{}, &settlementDomain.SettlementItem{}, &profitDomain.OrderProfitRecord{}, &financeDomain.FinanceAccount{}, &financeDomain.FinanceTransaction{})
	return NewService(db, dbtest.NewLogger(t))
}

func addVerifiedEvidence(t *testing.T, s *Service, c *ExperimentCase, kind, title string) *EvidenceRecord {
	t.Helper()
	observed := time.Now()
	e := &EvidenceRecord{ExperimentID: c.ExperimentID, Stage: c.Stage, EvidenceKind: kind, TruthStatus: TruthUnknown, Title: title, SourceURI: "https://example.test/" + title, ObservedAt: &observed}
	if err := s.AddEvidence(context.Background(), c.OwnerID, e); err != nil {
		t.Fatal(err)
	}
	verified, err := s.VerifyEvidence(context.Background(), c.ExperimentID, e.ID, c.OwnerID)
	if err != nil {
		t.Fatal(err)
	}
	return verified
}

func TestFinalProfitAndCashCannotBeDeclaredWithoutActualPassingGates(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	c := &ExperimentCase{Name: "truth", Stage: StageOpportunity, OwnerID: 1}
	if err := s.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Model(c).Update("stage", StageProfit).Error; err != nil {
		t.Fatal(err)
	}
	c.Stage = StageProfit
	c.FinalProfitStatus = ProfitFinal
	if err := s.Update(ctx, c.ExperimentID, 1, c); err == nil {
		t.Fatal("final profit accepted without a passing profit gate")
	}
	quoted := &EvidenceRecord{ExperimentID: c.ExperimentID, Stage: StageProfit, TruthStatus: TruthQuoted, Title: "quoted fee"}
	if err := s.AddEvidence(ctx, 1, quoted); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EvaluateGate(ctx, c.ExperimentID, 1, GateInput{Stage: StageProfit, GateCode: "profit_final", Result: ResultPass, EvidenceIDs: []int64{quoted.ID}}); err == nil {
		t.Fatal("quoted evidence passed final profit gate")
	}
}

func TestExpiredEvidenceCannotPassHighRiskGate(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	c := &ExperimentCase{Name: "expired", Stage: StageOpportunity, OwnerID: 1}
	if err := s.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Model(c).Update("stage", StageSupply).Error; err != nil {
		t.Fatal(err)
	}
	c.Stage = StageSupply
	past := time.Now().Add(-time.Minute)
	e := &EvidenceRecord{ExperimentID: c.ExperimentID, Stage: StageSupply, TruthStatus: TruthQuoted, Title: "old certificate", ExpiresAt: &past}
	if err := s.AddEvidence(ctx, 1, e); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EvaluateGate(ctx, c.ExperimentID, 1, GateInput{Stage: StageSupply, GateCode: "supply_ready", Result: ResultPass, EvidenceIDs: []int64{e.ID}}); err == nil {
		t.Fatal("expired evidence passed high-risk gate")
	}
}

func TestCreateAndGetDetailKeepsIndependentProfitAndCashState(t *testing.T) {
	s := testService(t)
	c := &ExperimentCase{Name: "first real test", Stage: StageOpportunity, FinalProfitStatus: "pending", CashRecoveryStatus: "pending", OwnerID: 1}
	if err := s.Create(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.ExperimentID == "" {
		t.Fatal("experiment_id must be generated")
	}
	d, err := s.GetDetail(context.Background(), c.ExperimentID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if d.Case.FinalProfitStatus != "pending" || d.Case.CashRecoveryStatus != "pending" {
		t.Fatalf("profit and cash states were not independent: %+v", d.Case)
	}
}

func TestGetDetailFailsClosedWhenAnyChildQueryFails(t *testing.T) {
	for _, failAt := range []int{2, 3, 4} {
		t.Run(strconv.Itoa(failAt), func(t *testing.T) {
			s := testService(t)
			c := &ExperimentCase{Name: "query failure", Stage: StageOpportunity, OwnerID: 1}
			if err := s.Create(context.Background(), c); err != nil {
				t.Fatal(err)
			}
			calls := 0
			name := "fail_get_detail_query_" + strconv.Itoa(failAt)
			if err := s.db.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
				calls++
				if calls == failAt {
					tx.AddError(errors.New("injected child query failure"))
				}
			}); err != nil {
				t.Fatal(err)
			}
			if detail, err := s.GetDetail(context.Background(), c.ExperimentID, 1); err == nil || detail != nil {
				t.Fatalf("GetDetail query %d returned detail=%#v err=%v", failAt, detail, err)
			}
		})
	}
}

func TestCreateAlwaysResetsClientSuppliedTerminalState(t *testing.T) {
	s := testService(t)
	now := time.Now()
	c := &ExperimentCase{Name: "injected", Stage: StageOpportunity, OwnerID: 1, Status: StatusCompleted, FinalProfitStatus: ProfitFinal, FinalProfitAmount: 999, ProfitCurrency: "USD", CashRecoveryStatus: CashRecovered, CashRecoveredAmount: 999, CashCurrency: "USD", CashRecoveredAt: &now, FinalDecision: "continue"}
	if err := s.Create(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusActive || c.FinalProfitStatus != ProfitPending || c.CashRecoveryStatus != CashPending || c.FinalDecision != "" || c.FinalProfitAmount != 0 || c.CashRecoveredAmount != 0 || c.CashRecoveredAt != nil {
		t.Fatalf("terminal injection survived create: %+v", c)
	}
}

func TestHighRiskGateCannotPassWithUntrustedEvidence(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	c := &ExperimentCase{Name: "x", Stage: StageOpportunity, OwnerID: 1}
	if err := s.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Model(c).Update("stage", StageSupply).Error; err != nil {
		t.Fatal(err)
	}
	c.Stage = StageSupply
	for _, truth := range []string{TruthMock, TruthInferred, TruthUnknown} {
		e := &EvidenceRecord{ExperimentID: c.ExperimentID, Stage: StageSupply, TruthStatus: truth, Title: truth}
		if err := s.AddEvidence(ctx, 1, e); err != nil {
			t.Fatal(err)
		}
		_, err := s.EvaluateGate(ctx, c.ExperimentID, 1, GateInput{Stage: StageSupply, GateCode: "supply_ready", Result: ResultPass, EvidenceIDs: []int64{e.ID}})
		if err == nil {
			t.Fatalf("%s evidence must not pass high-risk gate", truth)
		}
	}
}

func TestFutureStageGateIsBlockedAndOwnerSummaryReportsBlockers(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	c := &ExperimentCase{Name: "x", Stage: StageOpportunity, OwnerID: 1}
	if err := s.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	e := &EvidenceRecord{ExperimentID: c.ExperimentID, Stage: StageCash, TruthStatus: TruthQuoted, Title: "bank receipt"}
	if err := s.AddEvidence(ctx, 1, e); err == nil {
		t.Fatal("future-stage evidence unexpectedly accepted")
	}
	summary, err := s.OwnerSummary(ctx, c.ExperimentID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PassedGates != 0 || len(summary.Blockers) == 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestOwnerIsolationAndExplicitActualVerification(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	c := &ExperimentCase{Name: "private", Stage: StageOpportunity, OwnerID: 1}
	if err := s.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetDetail(ctx, c.ExperimentID, 2); err == nil {
		t.Fatal("other owner read experiment")
	}
	observed := time.Now()
	e := &EvidenceRecord{ExperimentID: c.ExperimentID, Stage: StageOpportunity, TruthStatus: TruthUnknown, Title: "source", SourceURI: "https://example.test/evidence", ObservedAt: &observed}
	if err := s.AddEvidence(ctx, 1, e); err != nil {
		t.Fatal(err)
	}
	if _, err := s.VerifyEvidence(ctx, c.ExperimentID, e.ID, 2); err == nil {
		t.Fatal("other owner verified evidence")
	}
	verified, err := s.VerifyEvidence(ctx, c.ExperimentID, e.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if verified.TruthStatus != TruthActual || verified.VerifiedBy != 1 || verified.VerifiedAt == nil {
		t.Fatalf("not verified: %+v", verified)
	}
}

func TestRejectsInvalidEnumsAndLinksObjects(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	if err := s.Create(ctx, &ExperimentCase{Name: "bad", Stage: "fantasy"}); err == nil {
		t.Fatal("invalid stage accepted")
	}
	c := &ExperimentCase{Name: "ok", Stage: StageOpportunity, OwnerID: 1}
	if err := s.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := s.AddEvidence(ctx, 1, &EvidenceRecord{ExperimentID: c.ExperimentID, Stage: StageProduct, TruthStatus: "probably", Title: "x"}); err == nil {
		t.Fatal("invalid truth accepted")
	}
	l := &ObjectLink{ExperimentID: c.ExperimentID, ObjectType: "sku", ObjectID: "42"}
	if err := s.AddObjectLink(ctx, 1, l); err != nil {
		t.Fatal(err)
	}
	d, err := s.GetDetail(ctx, c.ExperimentID, 1)
	if err != nil || len(d.ObjectLinks) != 1 {
		t.Fatalf("link missing: %+v %v", d, err)
	}
}

func TestExperimentCannotSkipStagesOrAdvanceWithoutPassingCurrentGate(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	c := &ExperimentCase{Name: "ordered", Stage: StageOpportunity, OwnerID: 1}
	if err := s.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	c.Stage = StageSupply
	if err := s.Update(ctx, c.ExperimentID, 1, c); err == nil {
		t.Fatal("experiment skipped directly to supply")
	}
	c.Stage = StageProduct
	if err := s.Update(ctx, c.ExperimentID, 1, c); err == nil {
		t.Fatal("experiment advanced without passing opportunity gate")
	}
	c.Stage = StageOpportunity
	support := addVerifiedEvidence(t, s, c, "support", "market-support")
	counter := addVerifiedEvidence(t, s, c, "counter", "market-counter")
	if _, err := s.EvaluateGate(ctx, c.ExperimentID, 1, GateInput{Stage: StageOpportunity, GateCode: "demand_evidence", Result: ResultPass, EvidenceIDs: []int64{support.ID, counter.ID}}); err != nil {
		t.Fatal(err)
	}
	c.Stage = StageProduct
	if err := s.Update(ctx, c.ExperimentID, 1, c); err != nil {
		t.Fatal(err)
	}
}

func TestProfitAndCashGatesRequireAndCaptureLinkedBusinessTruth(t *testing.T) {
	s := closureService(t)
	ctx := context.Background()
	c := &ExperimentCase{Name: "closure", Stage: StageOpportunity, OwnerID: 1}
	if err := s.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	order := orderDomain.Order{OrderNo: "EXP-ORDER-1"}
	if err := s.db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	imported, reconciled, paid := time.Now(), time.Now(), time.Now()
	st := settlementDomain.Settlement{PlatformID: 1, SettlementNo: "EXP-ST-1", Currency: "CNY", Status: "reconciled", SourceType: "api_sync", ImportedAt: &imported}
	if err := s.db.Create(&st).Error; err != nil {
		t.Fatal(err)
	}
	item := settlementDomain.SettlementItem{SettlementID: st.ID, TransactionType: "order_sale", OrderNo: order.OrderNo, OrderID: &order.ID, Amount: 100, ReconciliationStatus: "matched", ReconciledAt: &reconciled, ReconciledBy: "owner"}
	if err := s.db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	p := profitDomain.OrderProfitRecord{OrderID: order.ID, Revenue: 100, TotalCost: 70, Profit: 30, ProfitStatus: "final"}
	if err := s.db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	for typ, id := range map[string]int64{"order": order.ID, "settlement": st.ID, "profit_record": p.ID} {
		if err := s.AddObjectLink(ctx, 1, &ObjectLink{ExperimentID: c.ExperimentID, ObjectType: typ, ObjectID: strconv.FormatInt(id, 10)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.db.Model(c).Update("stage", StageProfit).Error; err != nil {
		t.Fatal(err)
	}
	c.Stage = StageProfit
	profitEvidence := addVerifiedEvidence(t, s, c, "support", "profit-proof")
	if _, err := s.EvaluateGate(ctx, c.ExperimentID, 1, GateInput{Stage: StageProfit, GateCode: "profit_final", Result: ResultPass, EvidenceIDs: []int64{profitEvidence.ID}}); err != nil {
		t.Fatal(err)
	}
	var stored ExperimentCase
	if err := s.db.Where("experiment_id = ?", c.ExperimentID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.FinalProfitAmount != 30 || stored.FinalRevenue != 100 || stored.ProfitCurrency != "CNY" {
		t.Fatalf("profit truth not captured: %+v", stored)
	}
	account := financeDomain.FinanceAccount{Name: "bank", AccountType: "bank", Currency: "CNY"}
	if err := s.db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	txn := financeDomain.FinanceTransaction{AccountID: account.ID, TransactionType: "revenue", Amount: 100, Currency: "CNY", OrderID: &order.ID, SettlementID: &st.ID, TransactionDate: &paid}
	if err := s.db.Create(&txn).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.AddObjectLink(ctx, 1, &ObjectLink{ExperimentID: c.ExperimentID, ObjectType: "cash_transaction", ObjectID: strconv.FormatInt(txn.ID, 10)}); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Model(c).Update("stage", StageCash).Error; err != nil {
		t.Fatal(err)
	}
	c.Stage = StageCash
	cashEvidence := addVerifiedEvidence(t, s, c, "support", "cash-proof")
	if _, err := s.EvaluateGate(ctx, c.ExperimentID, 1, GateInput{Stage: StageCash, GateCode: "cash_recovered", Result: ResultPass, EvidenceIDs: []int64{cashEvidence.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Where("experiment_id = ?", c.ExperimentID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.CashRecoveredAmount != 100 || stored.CashRecoveredAt == nil {
		t.Fatalf("cash truth not captured: %+v", stored)
	}
}

func TestManualUnreconciledSettlementCannotCloseProfit(t *testing.T) {
	s := closureService(t)
	ctx := context.Background()
	c := &ExperimentCase{Name: "blocked", Stage: StageOpportunity, OwnerID: 1}
	if err := s.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	order := orderDomain.Order{OrderNo: "EXP-BLOCK-1"}
	if err := s.db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	st := settlementDomain.Settlement{PlatformID: 1, SettlementNo: "EXP-BLOCK-ST", Status: "pending", SourceType: "manual"}
	if err := s.db.Create(&st).Error; err != nil {
		t.Fatal(err)
	}
	p := profitDomain.OrderProfitRecord{OrderID: order.ID, Revenue: 100, Profit: 100, ProfitStatus: "final"}
	if err := s.db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	for typ, id := range map[string]int64{"order": order.ID, "settlement": st.ID, "profit_record": p.ID} {
		if err := s.AddObjectLink(ctx, 1, &ObjectLink{ExperimentID: c.ExperimentID, ObjectType: typ, ObjectID: strconv.FormatInt(id, 10)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.db.Model(c).Update("stage", StageProfit).Error; err != nil {
		t.Fatal(err)
	}
	c.Stage = StageProfit
	e := addVerifiedEvidence(t, s, c, "support", "fake-final")
	if _, err := s.EvaluateGate(ctx, c.ExperimentID, 1, GateInput{Stage: StageProfit, GateCode: "profit_final", Result: ResultPass, EvidenceIDs: []int64{e.ID}}); err == nil {
		t.Fatal("manual unreconciled settlement closed profit")
	}
}
