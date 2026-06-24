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
