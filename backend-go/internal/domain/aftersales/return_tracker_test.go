package aftersales

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// sqliteSkuReturnStatsDDL is the SQLite-compatible DDL for sku_return_stats.
const sqliteSkuReturnStatsDDL = `
	CREATE TABLE IF NOT EXISTS sku_return_stats (
		sku_id INTEGER PRIMARY KEY,
		total_orders INTEGER DEFAULT 0,
		total_returns INTEGER DEFAULT 0,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)
`

func TestReturnRateTracker_TrackReturn(t *testing.T) {
	db := dbtest.NewDB(t)
	if err := db.Exec(sqliteSkuReturnStatsDDL).Error; err != nil {
		t.Fatalf("failed to create sku_return_stats table: %v", err)
	}

	log := dbtest.NewLogger(t)
	tracker := NewReturnRateTracker(db, log)

	// Track a return for SKU 1001 with qty=2.
	if err := tracker.TrackReturn(1001, 2); err != nil {
		t.Fatalf("TrackReturn failed: %v", err)
	}
	// Rate = 2/1 * 100 = 200.
	rate, err := tracker.GetReturnRate(1001)
	if err != nil {
		t.Fatalf("GetReturnRate failed: %v", err)
	}
	if rate != 200.0 {
		t.Errorf("expected rate 200.0, got %v", rate)
	}

	// Track another return for the same SKU with qty=1.
	if err := tracker.TrackReturn(1001, 1); err != nil {
		t.Fatalf("TrackReturn failed: %v", err)
	}
	// Rate = (2+1)/(1+1) * 100 = 150.
	rate, err = tracker.GetReturnRate(1001)
	if err != nil {
		t.Fatalf("GetReturnRate failed: %v", err)
	}
	if rate != 150.0 {
		t.Errorf("expected rate 150.0, got %v", rate)
	}
}

func TestReturnRateTracker_GetReturnRate_NoData(t *testing.T) {
	db := dbtest.NewDB(t)
	if err := db.Exec(sqliteSkuReturnStatsDDL).Error; err != nil {
		t.Fatalf("failed to create sku_return_stats table: %v", err)
	}

	log := dbtest.NewLogger(t)
	tracker := NewReturnRateTracker(db, log)

	rate, err := tracker.GetReturnRate(9999)
	if err != nil {
		t.Fatalf("GetReturnRate failed: %v", err)
	}
	if rate != 0 {
		t.Errorf("expected rate 0 for unknown SKU, got %v", rate)
	}
}

func TestReturnRateTracker_MultipleSKUs(t *testing.T) {
	db := dbtest.NewDB(t)
	if err := db.Exec(sqliteSkuReturnStatsDDL).Error; err != nil {
		t.Fatalf("failed to create sku_return_stats table: %v", err)
	}

	log := dbtest.NewLogger(t)
	tracker := NewReturnRateTracker(db, log)

	tracker.TrackReturn(1001, 1)
	tracker.TrackReturn(1001, 1)
	tracker.TrackReturn(2002, 1)
	tracker.TrackReturn(3003, 3)
	tracker.TrackReturn(3003, 2)

	// SKU 1001: 2 orders, 2 returns → rate = 100
	rate, _ := tracker.GetReturnRate(1001)
	if rate != 100.0 {
		t.Errorf("expected rate 100.0 for SKU 1001, got %v", rate)
	}

	// SKU 2002: 1 order, 1 return → rate = 100
	rate, _ = tracker.GetReturnRate(2002)
	if rate != 100.0 {
		t.Errorf("expected rate 100.0 for SKU 2002, got %v", rate)
	}

	// SKU 3003: 2 orders, 5 returns → rate = 250
	rate, _ = tracker.GetReturnRate(3003)
	if rate != 250.0 {
		t.Errorf("expected rate 250.0 for SKU 3003, got %v", rate)
	}
}
