package integrations

import (
	"context"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"testing"
	"time"
)

func TestListOwnerFactOptionsIsOwnerIsolated(t *testing.T) {
	db := dbtest.NewDB(t)
	ddl := []string{
		`CREATE TABLE owner_platform_account_authority(owner_id integer,account_id integer,platform_code text)`,
		`CREATE TABLE platform_integration_account(id integer primary key,store_name text)`,
		`CREATE TABLE platform_order_ingest(id integer primary key,owner_id integer,account_id integer,external_order_id text,platform_code text,normalized_order_id integer,event_action text,truth_status text,processing_status text,observed_at datetime)`,
		`CREATE TABLE platform_order_ingest_item(id integer primary key,ingest_id integer,line_number integer,currency text)`,
		`CREATE TABLE sales_order_item(id integer primary key,order_id integer,sku_id integer,sku_code text,product_name text,quantity integer)`,
		`CREATE TABLE sourcing_sku_mapping(id integer primary key,owner_id integer,internal_sku_id integer)`,
		`CREATE TABLE sourcing_cost_version(id integer primary key,owner_id integer,sku_mapping_id integer,version integer,total_minor integer,target_currency text,created_at datetime)`,
		`CREATE TABLE sourcing_cost_line(id integer primary key,cost_version_id integer,truth_status text)`,
		`CREATE TABLE platform_settlement_ingest(id integer primary key,owner_id integer,account_id integer,external_settlement_id text,platform_code text,currency text,truth_status text,observed_at datetime)`,
		`CREATE TABLE platform_settlement_fact_line(id integer primary key,ingest_id integer,kind text,amount_minor integer)`,
		`CREATE TABLE cash_receipt(id integer primary key,owner_id integer,amount_minor integer,external_receipt_id text,currency text,reconciliation_status text,truth_status text,observed_at datetime)`,
		`CREATE TABLE finance_account(id integer primary key,owner_id integer,name text,account_type text,currency text,status text)`,
		`CREATE TABLE aftersales_resolution_case(id integer primary key,owner_id integer,order_id integer,kind text,status text,requested_minor integer,currency text,created_at datetime)`,
	}
	for _, q := range ddl {
		if err := db.Exec(q).Error; err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	for _, owner := range []int{7, 8} {
		base := owner * 100
		db.Exec(`INSERT INTO platform_integration_account VALUES (?,?)`, base, "store")
		db.Exec(`INSERT INTO owner_platform_account_authority VALUES (?,?,?)`, owner, base, "p")
		db.Exec(`INSERT INTO platform_order_ingest VALUES (?,?,?,?,?,?,?,?,?,?)`, base, owner, base, "ext", "p", base, "reserve", "external_observed", "applied", now)
		db.Exec(`INSERT INTO platform_order_ingest_item VALUES (?,?,?,?)`, base, base, 1, "USD")
		db.Exec(`INSERT INTO sales_order_item VALUES (?,?,?,?,?,?)`, base, base, base, "sku", "product", 1)
		db.Exec(`INSERT INTO sourcing_sku_mapping VALUES (?,?,?)`, base, owner, base)
		db.Exec(`INSERT INTO sourcing_cost_version VALUES (?,?,?,?,?,?,?)`, base, owner, base, 1, 100, "USD", now)
		db.Exec(`INSERT INTO sourcing_cost_line VALUES (?,?,?)`, base, base, "actual")
		db.Exec(`INSERT INTO platform_settlement_ingest VALUES (?,?,?,?,?,?,?,?)`, base, owner, base, "settle", "p", "USD", "external_observed", now)
		db.Exec(`INSERT INTO platform_settlement_fact_line VALUES (?,?,?,?)`, base, base, "sale", 100)
		db.Exec(`INSERT INTO cash_receipt VALUES (?,?,?,?,?,?,?,?)`, base, owner, 100, "cash", "USD", "unreconciled", "external_observed", now)
		db.Exec(`INSERT INTO finance_account VALUES (?,?,?,?,?,?)`, base, owner, "bank", "bank", "USD", "active")
		db.Exec(`INSERT INTO aftersales_resolution_case VALUES (?,?,?,?,?,?,?,?)`, base, owner, base, "refund", "requested", 100, "USD", now)
	}
	out, err := NewService(db, dbtest.NewLogger(t)).ListOwnerFactOptions(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Orders) != 1 || out.Orders[0].ID != 700 || len(out.Settlements) != 1 || out.Settlements[0].ID != 700 || len(out.CashReceipts) != 1 || out.CashReceipts[0].ID != 700 || len(out.AftersalesCases) != 1 || out.AftersalesCases[0].ID != 700 {
		t.Fatalf("cross-owner option leak: %+v", out)
	}
}
