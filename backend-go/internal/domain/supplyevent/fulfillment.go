package supplyevent

import "time"

// QuoteRequested is published when A8 recommends a product and A10 needs to
// quote shipping costs.
// Topic: "supplychain.quote_requested"
type QuoteRequested struct {
	ProductID   int64     `json:"product_id"`
	SourceURL   string    `json:"source_url"`
	Destination string    `json:"destination"`
	WeightKg    float64   `json:"weight_kg"`
	Timestamp   time.Time `json:"timestamp"`
}

// QuoteReady is published when A10 completes a shipping quote.
// Topic: "supplychain.quote_ready"
type QuoteReady struct {
	ProductID        int64     `json:"product_id"`
	ChannelName      string    `json:"channel_name"`
	TotalShippingFee float64   `json:"total_shipping_fee"`
	Currency         string    `json:"currency"`
	Timestamp        time.Time `json:"timestamp"`
}

// OrderRequested is published when a supply chain order needs to be created.
// Topic: "supplychain.order.requested"
type OrderRequested struct {
	ProductID   int64     `json:"product_id"`
	ChannelName string    `json:"channel_name"`
	TotalAmount float64   `json:"total_amount"`
	Timestamp   time.Time `json:"timestamp"`
}

// FlywheelEvent is published when a supply_chain_flow is marked completed.
// It carries actual fulfillment data (cost, delivery time, loss info) back
// to A10 (carrier_performance) and A8 (category_performance) for the data
// flywheel loop — closing the feedback gap between quoted vs. actual shipping.
//
// Consumers subscribe via "supplychain.flywheel".
type FlywheelEvent struct {
	FlowID       string    `json:"flow_id"`
	ChannelName  string    `json:"channel_name"`
	ProviderName string    `json:"provider_name"`
	CategoryName string    `json:"category_name,omitempty"`
	ActualCost   float64   `json:"actual_cost"`
	Currency     string    `json:"currency"`
	DeliveryDays int       `json:"delivery_days"`
	IsLost       bool      `json:"is_lost"`
	SourceType   string    `json:"source_type"`
	SourceID     string    `json:"source_id"`
	Timestamp    time.Time `json:"timestamp"`
}
