package integrations

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

type factTestPlatform struct {
	ID   int64  `gorm:"column:id;primaryKey"`
	Code string `gorm:"column:code"`
}

func (factTestPlatform) TableName() string { return "platform" }

type factTestProduct struct {
	ID   int64  `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name"`
}

func (factTestProduct) TableName() string { return "product" }

type factTestSKU struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	ProductID int64  `gorm:"column:product_id"`
	Code      string `gorm:"column:code;unique"`
}

func (factTestSKU) TableName() string { return "sku" }

type factTestOrder struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement"`
	OrderNo    string `gorm:"column:order_no;unique"`
	PlatformID int64  `gorm:"column:platform_id"`
	Status     string `gorm:"column:status"`
}

func (factTestOrder) TableName() string { return "sales_order" }

type factTestOrderItem struct {
	ID                        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	OrderID, SkuID, ProductID int64
	ProductName, SkuCode      string
	UnitPrice                 float64
	Quantity                  int
	Subtotal                  float64
}

func (factTestOrderItem) TableName() string { return "sales_order_item" }

type factTestInventory struct {
	ID                       int64 `gorm:"column:id;primaryKey"`
	SkuID                    int64 `gorm:"column:sku_id;unique"`
	Quantity, LockedQuantity int
	UpdatedAt                time.Time
}

func (factTestInventory) TableName() string { return "inventory" }

type factTestAccountAuthority struct {
	OwnerID, AccountID int64
	PlatformCode       string
}

func (factTestAccountAuthority) TableName() string { return "owner_platform_account_authority" }

type factTestSKUAuthority struct{ OwnerID, InternalSkuID int64 }

func (factTestSKUAuthority) TableName() string { return "sourcing_sku_mapping" }

func newOrderFactTestService(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &factTestPlatform{}, &PlatformIntegrationAccount{}, &factTestProduct{}, &factTestSKU{}, &factTestOrder{}, &factTestOrderItem{}, &factTestInventory{}, &factTestAccountAuthority{}, &factTestSKUAuthority{}, &PlatformOrderIngest{}, &PlatformOrderIngestItem{}, &OrderInventoryLedger{})
	db.Create(&factTestPlatform{ID: 7, Code: "ozon"})
	db.Create(&PlatformIntegrationAccount{ID: 9, PlatformID: 7, StoreName: "owner store", AccountID: "external-account"})
	db.Create(&factTestAccountAuthority{OwnerID: 1, AccountID: 9, PlatformCode: "ozon"})
	db.Create(&factTestProduct{ID: 11, Name: "Owner product"})
	db.Create(&factTestSKU{ID: 12, ProductID: 11, Code: "SKU-1"})
	db.Create(&factTestSKUAuthority{OwnerID: 1, InternalSkuID: 12})
	db.Create(&factTestInventory{ID: 13, SkuID: 12, Quantity: 10})
	return NewService(db, zap.NewNop())
}

func factInput(eventID, orderID, action string) IngestExternalOrderInput {
	payload, _ := json.Marshal(map[string]any{"event_id": eventID, "order_id": orderID, "action": action})
	return IngestExternalOrderInput{OwnerID: 1, AccountID: 9, PlatformCode: "ozon", ExternalEventID: eventID, ExternalOrderID: orderID, Action: action, TruthStatus: "external_observed", ObservedAt: time.Now().UTC(), RawPayload: payload, Status: "pending", Lines: []ExternalOrderLine{{SKUCode: "SKU-1", Quantity: 3, UnitPriceMinor: 1950, Currency: "USD"}}}
}

func TestIngestExternalOrderReserveReplayCommitFactChain(t *testing.T) {
	svc := newOrderFactTestService(t)
	reserved, err := svc.IngestExternalOrder(context.Background(), factInput("evt-reserve", "order-1", OrderActionReserve))
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if reserved.OrderID == 0 || reserved.Replay {
		t.Fatalf("unexpected reserve result: %+v", reserved)
	}

	replayed, err := svc.IngestExternalOrder(context.Background(), factInput("evt-reserve", "order-1", OrderActionReserve))
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !replayed.Replay || replayed.OrderID != reserved.OrderID || replayed.IngestID != reserved.IngestID {
		t.Fatalf("replay did not return frozen result: %+v", replayed)
	}

	committed, err := svc.IngestExternalOrder(context.Background(), factInput("evt-commit", "order-1", OrderActionCommit))
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if committed.OrderID != reserved.OrderID {
		t.Fatalf("commit order mismatch: %+v", committed)
	}

	var inv factTestInventory
	svc.db.First(&inv, 13)
	if inv.Quantity != 7 || inv.LockedQuantity != 0 {
		t.Fatalf("inventory after commit = %+v", inv)
	}
	var ledgerCount, lineCount int64
	svc.db.Model(&OrderInventoryLedger{}).Count(&ledgerCount)
	svc.db.Model(&factTestOrderItem{}).Count(&lineCount)
	if ledgerCount != 2 || lineCount != 1 {
		t.Fatalf("ledger=%d order_lines=%d", ledgerCount, lineCount)
	}
}

func TestIngestExternalOrderReleaseIsReversible(t *testing.T) {
	svc := newOrderFactTestService(t)
	if _, err := svc.IngestExternalOrder(context.Background(), factInput("evt-reserve", "order-2", OrderActionReserve)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.IngestExternalOrder(context.Background(), factInput("evt-release", "order-2", OrderActionRelease)); err != nil {
		t.Fatal(err)
	}
	var inv factTestInventory
	svc.db.First(&inv, 13)
	if inv.Quantity != 10 || inv.LockedQuantity != 0 {
		t.Fatalf("release did not restore availability: %+v", inv)
	}
}

func TestIngestExternalOrderFailsClosedAndRollsBack(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*IngestExternalOrderInput)
	}{
		{"unknown account", func(in *IngestExternalOrderInput) { in.AccountID = 999 }},
		{"unknown sku", func(in *IngestExternalOrderInput) { in.Lines[0].SKUCode = "MISSING" }},
		{"unknown owner", func(in *IngestExternalOrderInput) { in.OwnerID = 0 }},
		{"unowned account", func(in *IngestExternalOrderInput) { in.OwnerID = 2 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newOrderFactTestService(t)
			in := factInput("evt-fail", "order-fail", OrderActionReserve)
			tt.mutate(&in)
			if _, err := svc.IngestExternalOrder(context.Background(), in); err == nil {
				t.Fatal("expected fail-closed error")
			}
			var ingests, orders, ledgers int64
			svc.db.Model(&PlatformOrderIngest{}).Count(&ingests)
			svc.db.Model(&factTestOrder{}).Count(&orders)
			svc.db.Model(&OrderInventoryLedger{}).Count(&ledgers)
			if ingests != 0 || orders != 0 || ledgers != 0 {
				t.Fatalf("partial facts persisted: ingest=%d order=%d ledger=%d", ingests, orders, ledgers)
			}
		})
	}
}

func TestIngestExternalOrderRejectsEventIdentityContentMismatch(t *testing.T) {
	svc := newOrderFactTestService(t)
	in := factInput("evt-stable", "order-3", OrderActionReserve)
	if _, err := svc.IngestExternalOrder(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	in.RawPayload = json.RawMessage(`{"changed":true}`)
	if _, err := svc.IngestExternalOrder(context.Background(), in); err == nil {
		t.Fatal("expected immutable event identity rejection")
	}
}
