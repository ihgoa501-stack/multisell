package experiment

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func TestReadOwnerBusinessClosureIsOwnerScopedAndRedacted(t *testing.T) {
	db := closureDB(t)
	svc := NewService(db, dbtest.NewLogger(t))
	now := time.Now().UTC()
	db.Exec(`INSERT INTO experiment_case (experiment_id, owner_id, name, stage, status) VALUES ('exp-owner', 42, 'case', 'profit', 'active')`)
	db.Exec(`INSERT INTO experiment_object_link (experiment_id, object_type, object_id) VALUES ('exp-owner','order','10'),('exp-owner','settlement','20'),('exp-owner','profit_record','30')`)
	db.Exec(`INSERT INTO sales_order (id, order_no, status, recipient_name, recipient_phone, shipping_address, remark, paid_at, delivered_at) VALUES (10,'ORDER-SECRET-1234','delivered','Alice','13800000000','secret address','secret remark',?,?)`, now, now)
	db.Exec(`INSERT INTO settlement (id, settlement_no, platform_id, currency, status, source_type, imported_at, raw_data) VALUES (20,'SET-1234',1,'CNY','reconciled','platform_import',?,'{"secret":true}')`, now)
	db.Exec(`INSERT INTO settlement_item (settlement_id, order_id, order_no, transaction_type, reconciliation_status, reconciled_at, reconciled_by) VALUES (20,10,'ORDER-SECRET-1234','order_sale','matched',?,'owner')`, now)
	db.Exec(`INSERT INTO order_profit_record (id, order_id, revenue, total_cost, profit, profit_status, missing_costs, calculated_at) VALUES (30,10,100,80,20,'final','',?)`, now)

	if _, err := svc.ReadOwnerBusinessClosure(context.Background(), 7, "exp-owner"); err == nil {
		t.Fatal("cross-owner read succeeded")
	}
	view, err := svc.ReadOwnerBusinessClosure(context.Background(), 42, "exp-owner")
	if err != nil {
		t.Fatal(err)
	}
	if view.AuthorityScope != "trace_only" || view.CausalStatus != "not_established" || view.FeedbackLoopStatus != "not_authorized" {
		t.Fatalf("closure projection must expose trace-only truth boundary: %#v", view)
	}
	if view.Order.TruthStatus != TruthUnknown || view.Order.SourceStatus != "internal_record" {
		t.Fatalf("order truth upgraded: %#v", view.Order)
	}
	if !view.Settlement.Trusted || !view.Settlement.FullyReconciled || !view.Profit.Final {
		t.Fatalf("closure not recognized: %#v", view)
	}
	raw, _ := json.Marshal(view)
	for _, secret := range []string{"Alice", "13800000000", "secret address", "secret remark", `\"secret\"`} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("sensitive value leaked: %s in %s", secret, raw)
		}
	}
	if len(view.Unknowns) < 2 {
		t.Fatalf("aftersales/cash unknowns missing: %#v", view.Unknowns)
	}
}

func TestReadOwnerBusinessClosureRejectsAmbiguousOrderAndBlocksMixedSettlements(t *testing.T) {
	db := closureDB(t)
	svc := NewService(db, dbtest.NewLogger(t))
	db.Exec(`INSERT INTO experiment_case (experiment_id, owner_id, name, stage, status) VALUES ('ambiguous',42,'case','profit','active')`)
	db.Exec(`INSERT INTO experiment_object_link (experiment_id, object_type, object_id) VALUES ('ambiguous','order','10'),('ambiguous','order','11')`)
	if _, err := svc.ReadOwnerBusinessClosure(context.Background(), 42, "ambiguous"); err == nil {
		t.Fatal("ambiguous order link accepted")
	}

	now := time.Now().UTC()
	db.Exec(`DELETE FROM experiment_object_link WHERE experiment_id='ambiguous'`)
	db.Exec(`INSERT INTO experiment_object_link (experiment_id, object_type, object_id) VALUES ('ambiguous','order','10'),('ambiguous','settlement','20'),('ambiguous','profit_record','30')`)
	db.Exec(`INSERT INTO sales_order (id, order_no, status) VALUES (10,'ORDER-10','paid')`)
	db.Exec(`INSERT INTO settlement (id, settlement_no, platform_id, currency, status, source_type, imported_at) VALUES (20,'S20',1,'CNY','reconciled','api_sync',?),(21,'S21',1,'CNY','reconciled','api_sync',?)`, now, now)
	db.Exec(`INSERT INTO settlement_item (settlement_id, order_id, order_no, transaction_type, reconciliation_status, reconciled_at, reconciled_by) VALUES (20,10,'ORDER-10','order_sale','matched',?,'o'),(21,10,'ORDER-10','platform_fee','matched',?,'o')`, now, now)
	db.Exec(`INSERT INTO order_profit_record (id, order_id, revenue, total_cost, profit, profit_status, missing_costs, calculated_at) VALUES (30,10,100,90,10,'final','',?)`, now)
	view, err := svc.ReadOwnerBusinessClosure(context.Background(), 42, "ambiguous")
	if err != nil {
		t.Fatal(err)
	}
	if view.Profit.Final {
		t.Fatalf("mixed settlement profit presented final: %#v", view.Profit)
	}
	if !containsText(view.Blockers, "多个结算") {
		t.Fatalf("mixed settlement blocker missing: %#v", view.Blockers)
	}
}

func TestReadOwnerBusinessClosureKeepsEvidenceUnknownAndRejectsCashWithoutSettlement(t *testing.T) {
	db := closureDB(t)
	svc := NewService(db, dbtest.NewLogger(t))
	now := time.Now().UTC()
	db.Exec(`INSERT INTO experiment_case (experiment_id, owner_id, name, stage, status) VALUES ('cash-no-settlement',42,'case','cash','active')`)
	db.Exec(`INSERT INTO experiment_object_link (experiment_id, object_type, object_id) VALUES ('cash-no-settlement','order','10'),('cash-no-settlement','cash_transaction','40')`)
	db.Exec(`INSERT INTO sales_order (id, order_no, status, pay_amount, paid_at) VALUES (10,'ORDER-10','confirmed',100,?)`, now)
	db.Exec(`INSERT INTO finance_account (id, account_type, currency, status) VALUES (50,'bank','CNY','active')`)
	db.Exec(`INSERT INTO finance_transaction (id, account_id, transaction_type, amount, currency, order_id, transaction_date) VALUES (40,50,'revenue',100,'CNY',10,?)`, now)
	view, err := svc.ReadOwnerBusinessClosure(context.Background(), 42, "cash-no-settlement")
	if err != nil {
		t.Fatal(err)
	}
	if view.Cash.RecordedCount != 0 || !containsText(view.Unknowns, "没有实验关联结算") {
		t.Fatalf("cash accepted without settlement: %#v", view.Cash)
	}
	for _, ref := range view.EvidenceRefs {
		if ref.TruthStatus != TruthUnknown {
			t.Fatalf("evidence truth upgraded: %#v", ref)
		}
	}
}

func containsText(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func closureDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t,
		&ExperimentCase{}, &ObjectLink{}, &closureOrderRow{}, &closureSettlementRow{},
		&closureSettlementItemRow{}, &closureProfitRow{}, &closureAftersaleRow{},
		&closureCashRow{}, &closureAccountRow{},
	)
}
