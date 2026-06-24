package settlement

import (
	"encoding/json"
	"time"
)

// Settlement maps to "settlement".
type Settlement struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID   *int64          `gorm:"column:platform_id" json:"platform_id,omitempty"`
	SettlementNo string          `gorm:"column:settlement_no;uniqueIndex" json:"settlement_no"`
	PeriodStart  *time.Time      `gorm:"column:period_start" json:"period_start,omitempty"`
	PeriodEnd    *time.Time      `gorm:"column:period_end" json:"period_end,omitempty"`
	Currency     string          `gorm:"column:currency;default:CNY" json:"currency"`
	TotalRevenue float64         `gorm:"column:total_revenue;default:0" json:"total_revenue"`
	TotalFee     float64         `gorm:"column:total_fee;default:0" json:"total_fee"`
	TotalRefund  float64         `gorm:"column:total_refund;default:0" json:"total_refund"`
	TotalNet     float64         `gorm:"column:total_net;default:0" json:"total_net"`
	Status       string          `gorm:"column:status;default:pending" json:"status"`
	RawData      json.RawMessage `gorm:"column:raw_data;type:jsonb" json:"raw_data,omitempty"`
	ImportedAt   *time.Time      `gorm:"column:imported_at" json:"imported_at,omitempty"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Settlement) TableName() string { return "settlement" }

// SettlementItem maps to "settlement_item".
type SettlementItem struct {
	ID                   int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SettlementID         int64      `gorm:"column:settlement_id;not null;index" json:"settlement_id"`
	TransactionType      string     `gorm:"column:transaction_type" json:"transaction_type"`
	TransactionID        string     `gorm:"column:transaction_id" json:"transaction_id"`
	OrderNo              string     `gorm:"column:order_no" json:"order_no"`
	OrderID              *int64     `gorm:"column:order_id" json:"order_id,omitempty"`
	SkuID                *int64     `gorm:"column:sku_id" json:"sku_id,omitempty"`
	Amount               float64    `gorm:"column:amount;default:0" json:"amount"`
	Fee                  float64    `gorm:"column:fee;default:0" json:"fee"`
	Net                  float64    `gorm:"column:net;default:0" json:"net"`
	Quantity             int        `gorm:"column:quantity;default:0" json:"quantity"`
	OccurredAt           *time.Time `gorm:"column:occurred_at" json:"occurred_at,omitempty"`
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	ReconciliationStatus string     `gorm:"column:reconciliation_status;default:pending" json:"reconciliation_status"`
	ReconciliationNote   string     `gorm:"column:reconciliation_note" json:"reconciliation_note"`
	ReconciledAt         *time.Time `gorm:"column:reconciled_at" json:"reconciled_at,omitempty"`
	ReconciledBy         string     `gorm:"column:reconciled_by" json:"reconciled_by"`
}

func (SettlementItem) TableName() string { return "settlement_item" }

// SettlementDetail is the composite detail payload.
type SettlementDetail struct {
	Settlement Settlement        `json:"settlement"`
	Items      []SettlementItem  `json:"items"`
}

// ---------- Input / DTO structs ----------

// CreateSettlementInput is the payload for POST /settlement.
type CreateSettlementInput struct {
	PlatformID   *int64          `json:"platform_id"`
	SettlementNo string          `json:"settlement_no" binding:"required"`
	PeriodStart  *time.Time      `json:"period_start"`
	PeriodEnd    *time.Time      `json:"period_end"`
	Currency     string          `json:"currency"`
	TotalRevenue *float64        `json:"total_revenue"`
	TotalFee     *float64        `json:"total_fee"`
	TotalRefund  *float64        `json:"total_refund"`
	TotalNet     *float64        `json:"total_net"`
	Status       string          `json:"status"`
	RawData      json.RawMessage `json:"raw_data"`
	ImportedAt   *time.Time      `json:"imported_at"`
	Items        []SettlementItemInput `json:"items"`
}

// SettlementItemInput is one line in CreateSettlementInput.
type SettlementItemInput struct {
	TransactionType string     `json:"transaction_type"`
	TransactionID   string     `json:"transaction_id"`
	OrderNo         string     `json:"order_no"`
	OrderID         *int64     `json:"order_id"`
	SkuID           *int64     `json:"sku_id"`
	Amount          *float64   `json:"amount"`
	Fee             *float64   `json:"fee"`
	Net             *float64   `json:"net"`
	Quantity        *int       `json:"quantity"`
	OccurredAt      *time.Time `json:"occurred_at"`
}

// UpdateSettlementInput allows partial updates.
type UpdateSettlementInput struct {
	PlatformID   *int64          `json:"platform_id"`
	PeriodStart  *time.Time      `json:"period_start"`
	PeriodEnd    *time.Time      `json:"period_end"`
	Currency     *string         `json:"currency"`
	TotalRevenue *float64        `json:"total_revenue"`
	TotalFee     *float64        `json:"total_fee"`
	TotalRefund  *float64        `json:"total_refund"`
	TotalNet     *float64        `json:"total_net"`
	Status       *string         `json:"status"`
	RawData      *json.RawMessage `json:"raw_data"`
}

// SettlementListFilter captures query parameters.
type SettlementListFilter struct {
	Search     string
	Status     string
	PlatformID *int64
}

// ReconcileInput is the payload for POST /settlement/:id/reconcile.
type ReconcileInput struct {
	ItemID               *int64 `json:"item_id"`
	ReconciliationStatus string `json:"reconciliation_status" binding:"required"`
	ReconciliationNote   string `json:"reconciliation_note"`
	ReconciledBy         string `json:"reconciled_by"`
}

// SettlementSummary is the aggregation payload for GET /settlement/summary.
type SettlementSummary struct {
	Total         int64              `json:"total"`
	ByStatus      map[string]int64   `json:"by_status"`
	NetByPlatform []PlatformNetTotal `json:"net_by_platform"`
}

// PlatformNetTotal is one row in SettlementSummary.NetByPlatform.
type PlatformNetTotal struct {
	PlatformID *int64  `json:"platform_id,omitempty"`
	TotalNet   float64 `json:"total_net"`
}
