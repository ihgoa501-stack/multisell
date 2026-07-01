package approval

import (
	"time"
)

// Approval status constants.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// ApprovalRequest maps to the "approval_request" table.
type ApprovalRequest struct {
	ID          int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID   int64      `gorm:"column:product_id;not null" json:"product_id"`
	RequestType string     `gorm:"column:request_type;not null" json:"request_type"` // publish, price_change, delist, content_update, listing_task
	Requester   string     `gorm:"column:requester;not null" json:"requester"`       // agent_id or user_id
	Reviewer    string     `gorm:"column:reviewer" json:"reviewer"`
	Status      string     `gorm:"column:status;default:pending" json:"status"` // pending, approved, rejected
	OldValue    string     `gorm:"column:old_value;type:text" json:"old_value,omitempty"`
	NewValue    string     `gorm:"column:new_value;type:text" json:"new_value,omitempty"`
	Reason      string     `gorm:"column:reason;type:text" json:"reason,omitempty"`
	ReviewNote  string     `gorm:"column:review_note;type:text" json:"review_note,omitempty"`
	TargetType  string     `gorm:"column:target_type" json:"target_type,omitempty"`
	TargetID    int64      `gorm:"column:target_id" json:"target_id,omitempty"`
	RiskLevel   string     `gorm:"column:risk_level" json:"risk_level,omitempty"`
	ExpiresAt   *time.Time `gorm:"column:expires_at" json:"expires_at,omitempty"`
	EntityType  string     `gorm:"column:entity_type" json:"entity_type,omitempty"`
	EntityID    int64      `gorm:"column:entity_id;default:0" json:"entity_id,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ApprovalRequest) TableName() string { return "approval_request" }

// CreateApprovalInput is the JSON body for creating an approval request.
type CreateApprovalInput struct {
	ProductID   int64      `json:"product_id" binding:"required"`
	RequestType string     `json:"request_type" binding:"required"`
	Requester   string     `json:"requester" binding:"required"`
	OldValue    string     `json:"old_value"`
	NewValue    string     `json:"new_value"`
	Reason      string     `json:"reason"`
	TargetType  string     `json:"target_type"`
	TargetID    int64      `json:"target_id"`
	RiskLevel   string     `json:"risk_level"`
	ExpiresAt   *time.Time `json:"expires_at"`
	EntityType  string     `json:"entity_type"`
	EntityID    int64      `json:"entity_id"`
}

// ReviewApprovalInput is the JSON body for reviewing an approval request.
type ReviewApprovalInput struct {
	Action     string `json:"action" binding:"required"` // approve, reject
	Reviewer   string `json:"reviewer" binding:"required"`
	ReviewNote string `json:"review_note"`
}

// ApprovalStats represents approval statistics.
type ApprovalStats struct {
	PendingCount   int64   `json:"pending_count"`
	ApprovedCount  int64   `json:"approved_count"`
	RejectedCount  int64   `json:"rejected_count"`
	TotalCount     int64   `json:"total_count"`
	AvgReviewHours float64 `json:"avg_review_hours"`
	EscalatedCount int64   `json:"escalated_count"`
}
