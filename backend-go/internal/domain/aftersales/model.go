package aftersales

import (
	"time"
)

// AfterSalesOrder maps to "after_sales_order".
type AfterSalesOrder struct {
	ID                int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID           int64      `gorm:"column:order_id;not null;index" json:"order_id"`
	ItemID            *int64     `gorm:"column:item_id" json:"item_id,omitempty"`
	SkuID             *int64     `gorm:"column:sku_id" json:"sku_id,omitempty"`
	ReturnQuantity    int        `gorm:"column:return_quantity;default:0" json:"return_quantity"`
	Reason            string     `gorm:"column:reason" json:"reason"`
	Status            string     `gorm:"column:status;default:pending" json:"status"`
	RefundAmount      float64    `gorm:"column:refund_amount;default:0" json:"refund_amount"`
	InspectionResult  string     `gorm:"column:inspection_result" json:"inspection_result"`
	RejectionReason   string     `gorm:"column:rejection_reason" json:"rejection_reason"`
	CreatedBy         string     `gorm:"column:created_by" json:"created_by"`
	ApprovedBy        string     `gorm:"column:approved_by" json:"approved_by"`
	ApprovedAt        *time.Time `gorm:"column:approved_at" json:"approved_at,omitempty"`
	RejectedBy        string     `gorm:"column:rejected_by" json:"rejected_by"`
	RejectedAt        *time.Time `gorm:"column:rejected_at" json:"rejected_at,omitempty"`
	ReceivedBy        string     `gorm:"column:received_by" json:"received_by"`
	ReceivedAt        *time.Time `gorm:"column:received_at" json:"received_at,omitempty"`
	RefundedBy        string     `gorm:"column:refunded_by" json:"refunded_by"`
	RefundedAt        *time.Time `gorm:"column:refunded_at" json:"refunded_at,omitempty"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AfterSalesOrder) TableName() string { return "after_sales_order" }

// CreateInput is the payload for POST /aftersales.
type CreateInput struct {
	OrderID        int64   `json:"order_id" binding:"required"`
	ItemID         *int64  `json:"item_id"`
	SkuID          *int64  `json:"sku_id"`
	ReturnQuantity *int    `json:"return_quantity"`
	Reason         string  `json:"reason"`
	Status         string  `json:"status"`
	RefundAmount   *float64 `json:"refund_amount"`
	CreatedBy      string  `json:"created_by"`
}

// UpdateInput allows partial updates.
type UpdateInput struct {
	ReturnQuantity   *int     `json:"return_quantity"`
	Reason           *string  `json:"reason"`
	Status           *string  `json:"status"`
	RefundAmount     *float64 `json:"refund_amount"`
	InspectionResult *string  `json:"inspection_result"`
	RejectionReason  *string  `json:"rejection_reason"`
}

// ListFilter captures query parameters.
type ListFilter struct {
	Search  string
	Status  string
	OrderID *int64
}

// ApproveInput is the body for POST /aftersales/:id/approve.
type ApproveInput struct {
	ApprovedBy       string `json:"approved_by"`
	InspectionResult string `json:"inspection_result"`
}

// RejectInput is the body for POST /aftersales/:id/reject.
type RejectInput struct {
	RejectedBy      string `json:"rejected_by"`
	RejectionReason string `json:"rejection_reason"`
}

// ReceiveInput is the body for POST /aftersales/:id/receive.
type ReceiveInput struct {
	ReceivedBy string `json:"received_by"`
}

// RefundInput is the body for POST /aftersales/:id/refund.
type RefundInput struct {
	RefundedBy   string  `json:"refunded_by"`
	RefundAmount float64 `json:"refund_amount"`
}

// Summary is the aggregation payload.
type Summary struct {
	Total         int64            `json:"total"`
	ByStatus      map[string]int64 `json:"by_status"`
	TotalRefunded float64          `json:"total_refunded"`
}
