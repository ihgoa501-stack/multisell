package logistics

import (
	"context"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

// ---------------------------------------------------------------------------
// Carrier performance tests
// ---------------------------------------------------------------------------

func TestRecordCarrierPerformance_InitialRecord(t *testing.T) {
	svc := NewService(nil)

	svc.recordCarrierPerformance("CNAIR", "CNExpress", 236.25, 9, false)

	cp := svc.GetCarrierPerformanceByChannel("CNAIR", "CNExpress")
	if cp == nil {
		t.Fatal("expected carrier performance record, got nil")
	}
	if cp.TotalOrders != 1 {
		t.Errorf("expected TotalOrders 1, got %d", cp.TotalOrders)
	}
	if cp.TotalCost != 236.25 {
		t.Errorf("expected TotalCost 236.25, got %f", cp.TotalCost)
	}
	if cp.AvgCost != 236.25 {
		t.Errorf("expected AvgCost 236.25, got %f", cp.AvgCost)
	}
	if cp.AvgDeliveryDays != 9 {
		t.Errorf("expected AvgDeliveryDays 9, got %f", cp.AvgDeliveryDays)
	}
	if cp.MinDeliveryDays != 9 || cp.MaxDeliveryDays != 9 {
		t.Errorf("expected Min/MaxDeliveryDays both 9, got min=%d max=%d", cp.MinDeliveryDays, cp.MaxDeliveryDays)
	}
	if cp.LostPackages != 0 {
		t.Errorf("expected LostPackages 0, got %d", cp.LostPackages)
	}
	if cp.LossRate != 0 {
		t.Errorf("expected LossRate 0, got %f", cp.LossRate)
	}
}

func TestRecordCarrierPerformance_AggregateMultiple(t *testing.T) {
	svc := NewService(nil)

	// Two shipments via the same channel.
	svc.recordCarrierPerformance("CNAIR", "CNExpress", 200.00, 7, false)
	svc.recordCarrierPerformance("CNAIR", "CNExpress", 300.00, 10, true) // lost

	cp := svc.GetCarrierPerformanceByChannel("CNAIR", "CNExpress")
	if cp == nil {
		t.Fatal("expected carrier performance record")
	}
	if cp.TotalOrders != 2 {
		t.Errorf("expected TotalOrders 2, got %d", cp.TotalOrders)
	}
	if cp.TotalCost != 500.00 {
		t.Errorf("expected TotalCost 500, got %f", cp.TotalCost)
	}
	if cp.AvgCost != 250.00 {
		t.Errorf("expected AvgCost 250, got %f", cp.AvgCost)
	}
	if cp.LostPackages != 1 {
		t.Errorf("expected LostPackages 1, got %d", cp.LostPackages)
	}
	if cp.LossRate != 50.0 {
		t.Errorf("expected LossRate 50, got %f", cp.LossRate)
	}
	// Min/Max should track extremes.
	if cp.MinDeliveryDays != 7 {
		t.Errorf("expected MinDeliveryDays 7, got %d", cp.MinDeliveryDays)
	}
	if cp.MaxDeliveryDays != 10 {
		t.Errorf("expected MaxDeliveryDays 10, got %d", cp.MaxDeliveryDays)
	}
}

func TestRecordCarrierPerformance_MultipleChannels(t *testing.T) {
	svc := NewService(nil)

	svc.recordCarrierPerformance("CNAIR", "CNExpress", 100, 5, false)
	svc.recordCarrierPerformance("CNSEA", "CNExpress", 50, 25, false)

	all := svc.GetCarrierPerformance()
	if len(all) != 2 {
		t.Fatalf("expected 2 carrier performance records, got %d", len(all))
	}

	// Find by channel.
	var air, sea *CarrierPerformance
	for i := range all {
		switch all[i].ChannelName {
		case "CNAIR":
			air = &all[i]
		case "CNSEA":
			sea = &all[i]
		}
	}
	if air == nil || sea == nil {
		t.Fatal("expected both CNAIR and CNSEA records")
	}
	if air.TotalOrders != 1 || sea.TotalOrders != 1 {
		t.Errorf("expected both to have 1 order, got air=%d sea=%d", air.TotalOrders, sea.TotalOrders)
	}
}

func TestRecordCarrierPerformance_MissingChannel(t *testing.T) {
	svc := NewService(nil)
	// Empty channel should create a record but not be very useful.
	svc.recordCarrierPerformance("", "", 100, 5, false)
	cp := svc.GetCarrierPerformanceByChannel("", "")
	if cp == nil {
		t.Fatal("expected a record even with empty channel")
	}
	if cp.TotalOrders != 1 {
		t.Errorf("expected TotalOrders 1, got %d", cp.TotalOrders)
	}
}

// ---------------------------------------------------------------------------
// Category performance tests
// ---------------------------------------------------------------------------

func TestRecordCategoryPerformance_InitialRecord(t *testing.T) {
	svc := NewService(nil)

	svc.recordCategoryPerformance("Electronics", "CNAIR", 236.25, 9)

	results := svc.GetCategoryPerformanceByCategory("Electronics")
	if len(results) != 1 {
		t.Fatalf("expected 1 category performance record, got %d", len(results))
	}
	cp := results[0]
	if cp.TotalOrders != 1 {
		t.Errorf("expected TotalOrders 1, got %d", cp.TotalOrders)
	}
	if cp.AvgCost != 236.25 {
		t.Errorf("expected AvgCost 236.25, got %f", cp.AvgCost)
	}
	if cp.AvgDeliveryDays != 9 {
		t.Errorf("expected AvgDeliveryDays 9, got %f", cp.AvgDeliveryDays)
	}
}

func TestRecordCategoryPerformance_Aggregate(t *testing.T) {
	svc := NewService(nil)

	svc.recordCategoryPerformance("Clothing", "CNAIR", 80.00, 5)
	svc.recordCategoryPerformance("Clothing", "CNAIR", 120.00, 7)

	results := svc.GetCategoryPerformanceByCategory("Clothing")
	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}
	cp := results[0]
	if cp.TotalOrders != 2 {
		t.Errorf("expected TotalOrders 2, got %d", cp.TotalOrders)
	}
	if cp.AvgCost != 100.00 {
		t.Errorf("expected AvgCost 100, got %f", cp.AvgCost)
	}
}

func TestRecordCategoryPerformance_MultipleCategories(t *testing.T) {
	svc := NewService(nil)

	svc.recordCategoryPerformance("Electronics", "CNAIR", 200, 7)
	svc.recordCategoryPerformance("Clothing", "CNSEA", 50, 20)

	all := svc.GetCategoryPerformance()
	if len(all) != 2 {
		t.Fatalf("expected 2 category performance records, got %d", len(all))
	}
}

func TestRecordCategoryPerformance_EmptyCategory(t *testing.T) {
	svc := NewService(nil)

	// Empty category should be skipped silently.
	svc.recordCategoryPerformance("", "CNAIR", 100, 5)

	results := svc.GetCategoryPerformance()
	if len(results) != 0 {
		t.Errorf("expected 0 records for empty category, got %d", len(results))
	}
}

func TestCategoryPerformance_SameCategoryDifferentChannels(t *testing.T) {
	svc := NewService(nil)

	svc.recordCategoryPerformance("Electronics", "CNAIR", 200, 7)
	svc.recordCategoryPerformance("Electronics", "CNSEA", 100, 25)

	results := svc.GetCategoryPerformanceByCategory("Electronics")
	if len(results) != 2 {
		t.Fatalf("expected 2 records (one per channel), got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// Event handler tests
// ---------------------------------------------------------------------------

func TestHandleFlywheelEvent_RecordsCarrierStats(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	svc := NewService(nil)
	bus.Subscribe("supplychain.flywheel", svc.HandleFlywheelEvent())

	payload := map[string]interface{}{
		"channel_name":  "CNAIR",
		"provider_name": "CNExpress",
		"actual_cost":   236.25,
		"currency":      "CNY",
		"delivery_days": 9,
		"is_lost":       false,
	}

	_, err := bus.Publish(ctx, "supplychain.flywheel", "supplychain", payload)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Allow event to be processed.
	time.Sleep(100 * time.Millisecond)

	cp := svc.GetCarrierPerformanceByChannel("CNAIR", "CNExpress")
	if cp == nil {
		t.Fatal("expected carrier performance record after event")
	}
	if cp.TotalOrders != 1 {
		t.Errorf("expected TotalOrders 1, got %d", cp.TotalOrders)
	}
	if cp.AvgCost != 236.25 {
		t.Errorf("expected AvgCost 236.25, got %f", cp.AvgCost)
	}
}

func TestHandleFlywheelEvent_MissingChannel(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	svc := NewService(nil)
	bus.Subscribe("supplychain.flywheel", svc.HandleFlywheelEvent())

	// Event without channel_name should be silently skipped.
	payload := map[string]interface{}{
		"actual_cost":   100,
		"delivery_days": 5,
	}

	_, err := bus.Publish(ctx, "supplychain.flywheel", "supplychain", payload)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	all := svc.GetCarrierPerformance()
	if len(all) != 0 {
		t.Errorf("expected 0 records for event with missing channel, got %d", len(all))
	}
}

func TestHandleFlywheelEvent_LostPackage(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	svc := NewService(nil)
	bus.Subscribe("supplychain.flywheel", svc.HandleFlywheelEvent())

	payload := map[string]interface{}{
		"channel_name":  "CNAIR",
		"provider_name": "CNExpress",
		"actual_cost":   300.00,
		"delivery_days": 0,
		"is_lost":       true,
	}

	_, err := bus.Publish(ctx, "supplychain.flywheel", "supplychain", payload)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	cp := svc.GetCarrierPerformanceByChannel("CNAIR", "CNExpress")
	if cp == nil {
		t.Fatal("expected carrier performance record")
	}
	if cp.LostPackages != 1 {
		t.Errorf("expected LostPackages 1, got %d", cp.LostPackages)
	}
}

func TestHandleCategoryFlywheelEvent(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	svc := NewService(nil)
	bus.Subscribe("supplychain.flywheel", svc.HandleCategoryFlywheelEvent())

	payload := map[string]interface{}{
		"category_name": "Electronics",
		"channel_name":  "CNAIR",
		"actual_cost":   250.00,
		"delivery_days": 7,
	}

	_, err := bus.Publish(ctx, "supplychain.flywheel", "supplychain", payload)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	results := svc.GetCategoryPerformanceByCategory("Electronics")
	if len(results) != 1 {
		t.Fatalf("expected 1 category performance record, got %d", len(results))
	}
	cp := results[0]
	if cp.TotalOrders != 1 {
		t.Errorf("expected TotalOrders 1, got %d", cp.TotalOrders)
	}
	if cp.AvgCost != 250.00 {
		t.Errorf("expected AvgCost 250, got %f", cp.AvgCost)
	}
}

func TestHandleCategoryFlywheelEvent_MissingCategory(t *testing.T) {
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger, eventbus.WithWorkers(1), eventbus.WithBufferSize(10))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	svc := NewService(nil)
	bus.Subscribe("supplychain.flywheel", svc.HandleCategoryFlywheelEvent())

	payload := map[string]interface{}{
		"channel_name": "CNAIR",
		"actual_cost":  100,
	}

	_, err := bus.Publish(ctx, "supplychain.flywheel", "supplychain", payload)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	all := svc.GetCategoryPerformance()
	if len(all) != 0 {
		t.Errorf("expected 0 records for missing category, got %d", len(all))
	}
}

func TestBothFlywheelHandlers_Independent(t *testing.T) {
	// Verify both handlers can be subscribed and process the same event
	// independently.
	logger := dbtest.NewLogger(t)
	bus := eventbus.New(logger, eventbus.WithWorkers(2), eventbus.WithBufferSize(10))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus.Start(ctx)

	svc := NewService(nil)
	bus.Subscribe("supplychain.flywheel", svc.HandleFlywheelEvent())
	bus.Subscribe("supplychain.flywheel", svc.HandleCategoryFlywheelEvent())

	payload := map[string]interface{}{
		"channel_name":  "CNAIR",
		"provider_name": "CNExpress",
		"category_name": "Electronics",
		"actual_cost":   236.25,
		"currency":      "CNY",
		"delivery_days": 9,
		"is_lost":       false,
	}

	_, err := bus.Publish(ctx, "supplychain.flywheel", "supplychain", payload)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Verify A10 data was recorded.
	cp := svc.GetCarrierPerformanceByChannel("CNAIR", "CNExpress")
	if cp == nil {
		t.Fatal("expected carrier performance after flywheel event")
	}
	if cp.TotalOrders != 1 {
		t.Errorf("carrier: expected TotalOrders 1, got %d", cp.TotalOrders)
	}

	// Verify A8 data was recorded.
	cats := svc.GetCategoryPerformanceByCategory("Electronics")
	if len(cats) == 0 {
		t.Fatal("expected category performance after flywheel event")
	}
	if cats[0].TotalOrders != 1 {
		t.Errorf("category: expected TotalOrders 1, got %d", cats[0].TotalOrders)
	}
}

// ---------------------------------------------------------------------------
// Concurrency test
// ---------------------------------------------------------------------------

func TestRecordCarrierPerformance_ConcurrentSafe(t *testing.T) {
	svc := NewService(nil)

	done := make(chan struct{})
	const goroutines = 10
	const recordsPerGoroutine = 100

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < recordsPerGoroutine; j++ {
				channel := "CNAIR"
				if id%2 == 0 {
					channel = "CNSEA"
				}
				svc.recordCarrierPerformance(channel, "CNExpress", 100.0, 7, false)
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	all := svc.GetCarrierPerformance()
	for _, cp := range all {
		if cp.TotalOrders != goroutines/2*recordsPerGoroutine {
			t.Errorf("channel %s: expected %d orders, got %d",
				cp.ChannelName, goroutines/2*recordsPerGoroutine, cp.TotalOrders)
		}
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestComputeAvg(t *testing.T) {
	tests := []struct {
		name        string
		currentAvg  float64
		newValue    float64
		count       int
		want        float64
	}{
		{"first value, count 1", 0, 100, 1, 100},
		{"second value", 100, 200, 2, 150},
		{"third value", 150, 300, 3, 200},
		{"large count convergence", 250, 250, 100, 250},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeAvg(tt.currentAvg, tt.newValue, tt.count)
			if got != tt.want {
				t.Errorf("computeAvg(%f, %f, %d) = %f, want %f", tt.currentAvg, tt.newValue, tt.count, got, tt.want)
			}
		})
	}
}

func TestParseDays(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want int
	}{
		{"int", int(5), 5},
		{"int64", int64(10), 10},
		{"float64", float64(15), 15},
		{"float64 fractional", float64(7.5), 7},
		{"nil", nil, 0},
		{"string", "3", 0}, // not a numeric type
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDays(tt.v)
			if got != tt.want {
				t.Errorf("parseDays(%v) = %d, want %d", tt.v, got, tt.want)
			}
		})
	}
}

func TestCarrierKey(t *testing.T) {
	if key := carrierKey("CNAIR", "CNExpress"); key != "CNAIR|CNExpress" {
		t.Errorf("expected 'CNAIR|CNExpress', got '%s'", key)
	}
}

func TestCategoryKey(t *testing.T) {
	if key := categoryKey("Electronics", "CNAIR"); key != "Electronics|CNAIR" {
		t.Errorf("expected 'Electronics|CNAIR', got '%s'", key)
	}
}
