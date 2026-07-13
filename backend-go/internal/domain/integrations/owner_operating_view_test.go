package integrations

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestOwnerOperatingViewJSONContractUsesSnakeCaseAndNoRawPayload(t *testing.T) {
	b, err := json.Marshal(OwnerOperatingView{Order: OwnerOrderFact{OrderID: 7, IngestID: 8, PayloadSHA256: strings.Repeat("a", 64)}, Inventory: []OwnerInventoryFact{{SKUID: 9, BeforeLockedQuantity: 1}}, Profit: &OwnerProfitFact{ProfitMinor: 12}})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, key := range []string{`"order_id":7`, `"ingest_id":8`, `"payload_sha256"`, `"sku_id":9`, `"before_locked_quantity":1`, `"profit_minor":12`, `"blockers"`} {
		if !strings.Contains(s, key) {
			t.Fatalf("missing %s in %s", key, s)
		}
	}
	for _, forbidden := range []string{"OrderID", "RawPayload", "raw_payload", "ExternalReceiptID"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("leaked/non-contract field %s in %s", forbidden, s)
		}
	}
}

func TestOwnerOperatingViewDoesNotAttributeBatchCashToOneOrder(t *testing.T) {
	db := dbtest.NewDB(t)
	ddl := []string{
		`CREATE TABLE platform_order_ingest (id integer primary key,owner_id integer,account_id integer,platform_code text,event_action text,truth_status text,payload_sha256 text,observed_at datetime,normalized_order_id integer,processing_status text)`,
		`CREATE TABLE order_inventory_ledger (id integer primary key,owner_id integer,ingest_id integer,order_id integer,order_item_id integer,inventory_id integer,sku_id integer,action text,quantity integer,before_quantity integer,after_quantity integer,before_locked_quantity integer,after_locked_quantity integer,created_at datetime)`,
		`CREATE TABLE supply_chain_tracking (id text,owner_id integer,order_id text)`,
		`CREATE TABLE supply_chain_carrier_event (id integer,owner_id integer,tracking_id text,source_system text,status text,truth_status text,payload_sha256 text,occurred_at datetime,observed_at datetime)`,
		`CREATE TABLE aftersales_resolution_case (id integer,owner_id integer,order_id integer,kind text,status text,currency text,request_source text,request_evidence_id text,requested_minor integer,request_observed_at datetime)`,
		`CREATE TABLE aftersales_resolution_receipt (id integer,owner_id integer,resolution_id integer,outcome text,source_type text,evidence_id text,external_receipt_id text,currency text,receipt_sha256 text,actual_minor integer,observed_at datetime)`,
		`CREATE TABLE platform_settlement_ingest (id integer,owner_id integer,account_id integer,platform_code text,truth_status text,currency text,payload_sha256 text,content_sha256 text,observed_at datetime)`,
		`CREATE TABLE platform_settlement_fact_line (id integer,ingest_id integer,line_number integer,order_id integer,kind text,fee_code text,currency text,amount_minor integer,occurred_at datetime)`,
		`CREATE TABLE order_final_profit_version (id integer,owner_id integer,order_id integer,version integer,revenue_minor integer,product_cost_minor integer,settlement_fee_minor integer,fulfillment_fee_minor integer,refund_minor integer,total_cost_minor integer,profit_minor integer,currency text,source_manifest_sha256 text,finalized_at datetime)`,
		`CREATE TABLE cash_reconciliation (id integer,owner_id integer,cash_receipt_id integer,platform_settlement_ingest_id integer,amount_minor integer,expected_receivable_minor integer,currency text,status text,request_sha256 text,reconciled_at datetime)`,
	}
	for _, q := range ddl {
		if err := db.Exec(q).Error; err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	hash := strings.Repeat("a", 64)
	if err := db.Exec(`INSERT INTO platform_order_ingest VALUES (1,7,3,'p','reserve','external_observed',?,?,11,'applied')`, hash, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO platform_settlement_ingest VALUES (20,7,3,'p','external_observed','USD',?,?,?)`, hash, hash, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO platform_settlement_fact_line VALUES (1,20,1,11,'sale','','USD',100,?),(2,20,2,12,'sale','','USD',200,?)`, now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO cash_reconciliation VALUES (30,7,2,20,300,300,'USD','reconciled',?,?)`, hash, now).Error; err != nil {
		t.Fatal(err)
	}
	v, err := NewService(db, dbtest.NewLogger(t)).ReadOwnerOperatingView(context.Background(), 7, 11)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Cash) != 0 {
		t.Fatalf("batch cash leaked into order: %#v", v.Cash)
	}
	found := false
	for _, b := range v.Blockers {
		if strings.Contains(b, "批次级现金不可归属单订单") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing blocker: %#v", v.Blockers)
	}
}
