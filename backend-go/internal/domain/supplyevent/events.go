// Package supplyevent defines shared event contracts for the supply chain domain.
//
// These events flow through the internal EventBus (internal/platform/eventbus/)
// and decouple supply-chain subdomains (purchase, inventory, aftersales) from
// each other — modules publish events without importing each other's services.
//
// Topic convention: "supplychain.{event_name}" — e.g. "supplychain.order.received".
//
// All events are pure Go structs with JSON tags for serialization.
// No handler, service, or model pattern — this package is event shapes only.
package supplyevent

import "time"

// OrderReceived is published when a purchase order is fully or partially received.
//
// Inventory subscribes via "supplychain.order.received" to auto-increment stock.
type OrderReceived struct {
	OrderNo    string         `json:"order_no"`
	SupplierID int64          `json:"supplier_id"`
	Items      []ReceivedItem `json:"items"`
	ReceivedAt time.Time      `json:"received_at"`
}

// ReceivedItem is one line item in an OrderReceived event.
type ReceivedItem struct {
	SkuID int64 `json:"sku_id"`
	Qty   int   `json:"qty"` // quantity received in this event
}

// StockAdjusted is published when an inventory adjustment is made outside
// the standard purchase → inventory → aftersales flow (e.g. manual count).
//
// Consumer topic: "supplychain.stock.adjusted".
type StockAdjusted struct {
	SkuID     int64  `json:"sku_id"`
	Warehouse string `json:"warehouse"`
	Delta     int    `json:"delta"`  // positive = increase, negative = decrease
	Reason    string `json:"reason"` // e.g. "manual", "write-off", "return"
}

// AfterSaleProcessed is published when an aftersales operation completes
// (return, exchange, or refund) and inventory needs to be adjusted.
//
// Consumer topic: "supplychain.aftersale.completed".
// Before publishing, the publisher should validate the AftersaleOrder is in
// a completed/settled state to prevent premature inventory deduction.
type AfterSaleProcessed struct {
	AftersaleID int64  `json:"aftersale_id"`
	OrderID     int64  `json:"order_id"`
	SkuID       int64  `json:"sku_id"`
	Quantity    int    `json:"quantity"`  // positive = items returned
	Type        string `json:"type"`      // "return", "exchange", "refund"
	ProcessedAt time.Time `json:"processed_at"`
}

// AftersaleReturned is published when a customer initiates a return, before
// the aftersales order enters inspection. The orchestrator consumes this to
// create a reverse supply-chain flow (return -> inspect -> refund/resend).
//
// Consumer topic: "supplychain.aftersale.returned".
type AftersaleReturned struct {
	AftersaleID int64     `json:"aftersale_id"`
	OrderID     int64     `json:"order_id"`
	SkuID       int64     `json:"sku_id"`
	Quantity    int       `json:"quantity"`
	Reason      string    `json:"reason"`
	ReturnedAt  time.Time `json:"returned_at"`
}

// StockCritical is published when A5 stock_alert detects a red (critical) stock level.
// StockCritical is published when A5 stock_alert detects a red (critical) stock level.
// The supply chain orchestrator consumes this to trigger A8 sourcing scanning.
//
// Consumer topic: "supplychain.stock.critical".
type StockCritical struct {
	SkuID        int64     `json:"sku_id"`
	CurrentStock int       `json:"current_stock"`
	SafetyStock  int       `json:"safety_stock"`
	SellableDays float64   `json:"sellable_days"`
	ReportedAt   time.Time `json:"reported_at"`
}
