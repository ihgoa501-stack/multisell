package orderimport

import (
	"encoding/json"
	"time"
)

// OrderImport maps to "order_import".
type OrderImport struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID   *int64          `gorm:"column:platform_id" json:"platform_id,omitempty"`
	SourceType   string          `gorm:"column:source_type;default:manual" json:"source_type"`
	FileName     string          `gorm:"column:file_name" json:"file_name"`
	TotalRows    int             `gorm:"column:total_rows;default:0" json:"total_rows"`
	SuccessCount int             `gorm:"column:success_count;default:0" json:"success_count"`
	ErrorCount   int             `gorm:"column:error_count;default:0" json:"error_count"`
	ErrorDetail  json.RawMessage `gorm:"column:error_detail;type:jsonb" json:"error_detail,omitempty"`
	Status       string          `gorm:"column:status;default:pending" json:"status"`
	CreatedBy    string          `gorm:"column:created_by" json:"created_by"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (OrderImport) TableName() string { return "order_import" }

// OrderImportBatch maps to "order_import_batch".
type OrderImportBatch struct {
	ID                     int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AdapterCode            string     `gorm:"column:adapter_code;not null" json:"adapter_code"`
	Platform               string     `gorm:"column:platform" json:"platform"`
	StoreName              string     `gorm:"column:store_name" json:"store_name"`
	SourceFilename         string     `gorm:"column:source_filename;not null" json:"source_filename"`
	RowCount               int        `gorm:"column:row_count;default:0" json:"row_count"`
	CreatedOrderCount      int        `gorm:"column:created_order_count;default:0" json:"created_order_count"`
	SkippedDuplicateCount  int        `gorm:"column:skipped_duplicate_count;default:0" json:"skipped_duplicate_count"`
	FailedCount            int        `gorm:"column:failed_count;default:0" json:"failed_count"`
	ImportedBy             string     `gorm:"column:imported_by" json:"imported_by"`
	ChainStatus            string     `gorm:"column:chain_status;default:chain_pending" json:"chain_status"`
	LedgerRebuiltCount     int        `gorm:"column:ledger_rebuilt_count;default:0" json:"ledger_rebuilt_count"`
	ExceptionGeneratedCount int       `gorm:"column:exception_generated_count;default:0" json:"exception_generated_count"`
	ChainFailureCount      int        `gorm:"column:chain_failure_count;default:0" json:"chain_failure_count"`
	ProcessedAt            *time.Time `gorm:"column:processed_at" json:"processed_at,omitempty"`
	CreatedAt              time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt              time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (OrderImportBatch) TableName() string { return "order_import_batch" }

// OrderImportItem maps to "order_import_item".
type OrderImportItem struct {
	ID                int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BatchID           int64           `gorm:"column:batch_id;not null;index" json:"batch_id"`
	RowNumber         int             `gorm:"column:row_number;not null" json:"row_number"`
	Platform          string          `gorm:"column:platform" json:"platform"`
	StoreName         string          `gorm:"column:store_name" json:"store_name"`
	PlatformOrderNo   string          `gorm:"column:platform_order_no" json:"platform_order_no"`
	OrderNo           string          `gorm:"column:order_no" json:"order_no"`
	OrderID           *int64          `gorm:"column:order_id" json:"order_id,omitempty"`
	SkuCode           string          `gorm:"column:sku_code;not null" json:"sku_code"`
	Quantity          int             `gorm:"column:quantity;not null" json:"quantity"`
	UnitPrice         *float64        `gorm:"column:unit_price" json:"unit_price,omitempty"`
	Currency          string          `gorm:"column:currency;default:CNY" json:"currency"`
	RecipientName     string          `gorm:"column:recipient_name" json:"recipient_name"`
	RecipientPhone    string          `gorm:"column:recipient_phone" json:"recipient_phone"`
	CountryCode       string          `gorm:"column:country_code" json:"country_code"`
	ShippingAddress   string          `gorm:"column:shipping_address" json:"shipping_address"`
	ShippingFee       *float64        `gorm:"column:shipping_fee" json:"shipping_fee,omitempty"`
	TrackingNumber    string          `gorm:"column:tracking_number" json:"tracking_number"`
	PaidAt            string          `gorm:"column:paid_at" json:"paid_at"`
	Status            string          `gorm:"column:status;not null" json:"status"`
	FailureReason     string          `gorm:"column:failure_reason" json:"failure_reason"`
	ChainStatus       string          `gorm:"column:chain_status;default:chain_pending" json:"chain_status"`
	ChainFailureReason string         `gorm:"column:chain_failure_reason" json:"chain_failure_reason"`
	RawPayload        json.RawMessage `gorm:"column:raw_payload;type:jsonb" json:"raw_payload,omitempty"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (OrderImportItem) TableName() string { return "order_import_item" }

// CreateInput is the payload for POST /order-import.
type CreateInput struct {
	PlatformID *int64  `json:"platform_id"`
	SourceType string  `json:"source_type"`
	FileName   string  `json:"file_name" binding:"required"`
	TotalRows  *int    `json:"total_rows"`
	Status     string  `json:"status"`
	CreatedBy  string  `json:"created_by"`
}

// UpdateInput allows partial updates.
type UpdateInput struct {
	PlatformID   *int64           `json:"platform_id"`
	SourceType   *string          `json:"source_type"`
	FileName     *string          `json:"file_name"`
	TotalRows    *int             `json:"total_rows"`
	SuccessCount *int             `json:"success_count"`
	ErrorCount   *int             `json:"error_count"`
	ErrorDetail  *json.RawMessage `json:"error_detail"`
	Status       *string          `json:"status"`
}

// ListFilter captures query parameters.
type ListFilter struct {
	Search     string
	Status     string
	PlatformID *int64
	SourceType string
}

// CompleteInput is the body for POST /order-import/:id/complete.
type CompleteInput struct {
	SuccessCount int             `json:"success_count"`
	ErrorCount   int             `json:"error_count"`
	ErrorDetail  json.RawMessage `json:"error_detail"`
	Status       string          `json:"status"`
}

// Summary is the aggregation payload.
type Summary struct {
	Total        int64            `json:"total"`
	ByStatus     map[string]int64 `json:"by_status"`
	TotalRows    int64            `json:"total_rows"`
	SuccessCount int64            `json:"success_count"`
	ErrorCount   int64            `json:"error_count"`
}
