package mock

import (
	"encoding/json"
	"time"
)

// MockOrder represents a simulated platform order.
type MockOrder struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID   int64           `gorm:"column:platform_id;not null" json:"platform_id"`
	OrderNo      string          `gorm:"column:order_no;not null" json:"order_no"`
	ProductName  string          `gorm:"column:product_name" json:"product_name"`
	Quantity     int             `gorm:"column:quantity;default:1" json:"quantity"`
	TotalAmount  float64         `gorm:"column:total_amount;default:0" json:"total_amount"`
	Currency     string          `gorm:"column:currency;default:USD" json:"currency"`
	Status       string          `gorm:"column:status;default:pending" json:"status"` // pending, shipped, delivered, cancelled
	OrderDate    time.Time       `gorm:"column:order_date" json:"order_date"`
	IsSeedData   bool            `gorm:"column:is_seed_data;default:false" json:"is_seed_data"`
	ExtraData    json.RawMessage `gorm:"column:extra_data;type:jsonb" json:"extra_data,omitempty"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (MockOrder) TableName() string { return "mock_order" }

// MockSettlement represents a simulated settlement record.
type MockSettlement struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID      int64     `gorm:"column:platform_id;not null" json:"platform_id"`
	Period          string    `gorm:"column:period;not null" json:"period"` // e.g. "2026-06"
	TotalRevenue    float64   `gorm:"column:total_revenue;default:0" json:"total_revenue"`
	TotalFee        float64   `gorm:"column:total_fee;default:0" json:"total_fee"`
	NetAmount       float64   `gorm:"column:net_amount;default:0" json:"net_amount"`
	Currency        string    `gorm:"column:currency;default:USD" json:"currency"`
	OrderCount      int       `gorm:"column:order_count;default:0" json:"order_count"`
	IsSeedData      bool      `gorm:"column:is_seed_data;default:false" json:"is_seed_data"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (MockSettlement) TableName() string { return "mock_settlement" }

// MockSyncStatus represents the sync status for a platform.
type MockSyncStatus struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID    int64      `gorm:"column:platform_id;not null" json:"platform_id"`
	PlatformName  string     `gorm:"column:platform_name" json:"platform_name"`
	SyncType      string     `gorm:"column:sync_type;not null" json:"sync_type"` // orders, products, fees, settlements
	Status        string     `gorm:"column:status;default:success" json:"status"` // success, failed, pending, in_progress
	RecordsSynced int        `gorm:"column:records_synced;default:0" json:"records_synced"`
	ErrorMessage  string     `gorm:"column:error_message" json:"error_message"`
	LastSyncAt    *time.Time `gorm:"column:last_sync_at" json:"last_sync_at,omitempty"`
	IsMockData    bool       `gorm:"column:is_mock_data;default:true" json:"is_mock_data"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (MockSyncStatus) TableName() string { return "mock_sync_status" }

// OwnerRiskSummary is the aggregated risk summary for the Owner cockpit.
type OwnerRiskSummary struct {
	LowProfitProducts   int64 `json:"low_profit_products"`
	MissingDataProducts int64 `json:"missing_data_products"`
	PendingApprovals    int64 `json:"pending_approvals"`
	SyncErrors          int64 `json:"sync_errors"`
	TotalCandidates     int64 `json:"total_candidates"`
	TotalRecommendations int64 `json:"total_recommendations"`
	ListReadyProducts   int64 `json:"list_ready_products"`
}

// OwnerSuggestion is an agent suggestion for the Owner cockpit.
type OwnerSuggestion struct {
	ID            int64   `json:"id"`
	ProductID     int64   `json:"product_id"`
	ProductTitle  string  `json:"product_title"`
	AgentSource   string  `json:"agent_source"`
	Suggestion    string  `json:"suggestion"`
	Reason        string  `json:"reason"`
	Confidence    float64 `json:"confidence"`
	RiskLevel     string  `json:"risk_level"`
	CreatedAt     string  `json:"created_at"`
}

// OwnerPlatformSyncStatus shows the sync status for each platform.
type OwnerPlatformSyncStatus struct {
	PlatformID    int64  `json:"platform_id"`
	PlatformName  string `json:"platform_name"`
	Mode          string `json:"mode"` // mock, sandbox, production
	OrdersSync    string `json:"orders_sync"`
	ProductsSync  string `json:"products_sync"`
	FeesSync      string `json:"fees_sync"`
	SettlementsSync string `json:"settlements_sync"`
	LastSyncTime  string `json:"last_sync_time"`
}
