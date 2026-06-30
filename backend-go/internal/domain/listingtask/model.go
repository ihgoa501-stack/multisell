package listingtask

import (
	"encoding/json"
	"time"
)

// ListingTask maps to the "listing_task" table — the listing task queue.
type ListingTask struct {
	ID                  int64            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID           int64            `gorm:"column:product_id;not null" json:"product_id"`
	PlatformID          int64            `gorm:"column:platform_id;not null" json:"platform_id"`
	SkuID               *int64           `gorm:"column:sku_id" json:"sku_id,omitempty"`
	ProductListingID    *int64           `gorm:"column:product_listing_id" json:"product_listing_id,omitempty"`
	SourceType          string           `gorm:"column:source_type;default:decision" json:"source_type"`
	SourceItemKey       string           `gorm:"column:source_item_key" json:"source_item_key"`
	Status              string           `gorm:"column:status;default:blocked" json:"status"`
	MissingRequirements json.RawMessage  `gorm:"column:missing_requirements;type:jsonb" json:"missing_requirements,omitempty"`
	DecisionSnapshot    json.RawMessage  `gorm:"column:decision_snapshot;type:jsonb" json:"decision_snapshot,omitempty"`
	TargetSalePrice     *float64         `gorm:"column:target_sale_price" json:"target_sale_price,omitempty"`
	TargetProfitMargin  *float64         `gorm:"column:target_profit_margin" json:"target_profit_margin,omitempty"`
	DestinationCountry  string           `gorm:"column:destination_country" json:"destination_country"`
	ApprovalID          *int64           `gorm:"column:approval_id" json:"approval_id,omitempty"`
	LastError           string           `gorm:"column:last_error" json:"last_error"`
	CreatedBy           string           `gorm:"column:created_by" json:"created_by"`
	UpdatedBy           string           `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt           time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName explicitly sets the table name (no AutoMigrate).
func (ListingTask) TableName() string { return "listing_task" }

// ListingTaskItem maps to the "listing_task_item" table — per product x platform entry.
type ListingTaskItem struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TaskID       int64           `gorm:"column:task_id;not null" json:"task_id"`
	ProductID    int64           `gorm:"column:product_id;not null" json:"product_id"`
	PlatformID   int64           `gorm:"column:platform_id;not null" json:"platform_id"`
	Status       string          `gorm:"column:status;default:pending" json:"status"`
	Result       json.RawMessage `gorm:"column:result;type:jsonb" json:"result,omitempty"`
	ErrorMessage string          `gorm:"column:error_message" json:"error_message"`
	RetryCount   int             `gorm:"column:retry_count;default:0" json:"retry_count"`
	ExecutedAt   *time.Time      `gorm:"column:executed_at" json:"executed_at,omitempty"`
}

// TableName explicitly sets the table name.
func (ListingTaskItem) TableName() string { return "listing_task_item" }

// CreateTaskInput is the payload for creating a listing task.
type CreateTaskInput struct {
	ProductID           int64            `json:"product_id" binding:"required"`
	PlatformID          int64            `json:"platform_id" binding:"required"`
	SkuID               *int64           `json:"sku_id"`
	ProductListingID    *int64           `json:"product_listing_id"`
	SourceType          string           `json:"source_type"`
	SourceItemKey       string           `json:"source_item_key"`
	Status              string           `json:"status"`
	MissingRequirements json.RawMessage  `json:"missing_requirements"`
	DecisionSnapshot    json.RawMessage  `json:"decision_snapshot"`
	TargetSalePrice     *float64         `json:"target_sale_price"`
	TargetProfitMargin  *float64         `json:"target_profit_margin"`
	DestinationCountry  string           `json:"destination_country"`
	ApprovalID          *int64           `json:"approval_id"`
	CreatedBy           string           `json:"created_by"`
}

// UpdateTaskInput is the payload for updating a listing task.
type UpdateTaskInput struct {
	Status              *string           `json:"status"`
	SourceItemKey       *string           `json:"source_item_key"`
	MissingRequirements *json.RawMessage  `json:"missing_requirements"`
	DecisionSnapshot    *json.RawMessage  `json:"decision_snapshot"`
	TargetSalePrice     *float64          `json:"target_sale_price"`
	TargetProfitMargin  *float64          `json:"target_profit_margin"`
	DestinationCountry  *string           `json:"destination_country"`
	ApprovalID          *int64            `json:"approval_id"`
	LastError           *string           `json:"last_error"`
	ProductListingID    *int64            `json:"product_listing_id"`
	UpdatedBy           *string           `json:"updated_by"`
}

// CreateTaskItemInput is the payload for creating a task item.
type CreateTaskItemInput struct {
	TaskID     int64           `json:"task_id" binding:"required"`
	ProductID  int64           `json:"product_id" binding:"required"`
	PlatformID int64           `json:"platform_id" binding:"required"`
	Status     string          `json:"status"`
	Result     json.RawMessage `json:"result"`
}

// UpdateTaskItemInput is the payload for updating a task item.
type UpdateTaskItemInput struct {
	Status       *string          `json:"status"`
	Result       *json.RawMessage `json:"result"`
	ErrorMessage *string          `json:"error_message"`
	RetryCount   *int             `json:"retry_count"`
}
