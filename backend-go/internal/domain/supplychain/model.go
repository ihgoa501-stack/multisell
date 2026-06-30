package supplychain

import (
	"encoding/json"
	"time"
)

// SupplyChainFlow maps to the PostgreSQL "supply_chain_flow" table.
type SupplyChainFlow struct {
	ID               string           `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	SourceType       string           `gorm:"column:source_type;type:varchar(50)" json:"source_type"`
	SourceID         string           `gorm:"column:source_id;type:varchar(100)" json:"source_id"`
	Status           string           `gorm:"column:status;type:varchar(20);default:pending" json:"status"`
	Context          *json.RawMessage `gorm:"column:context;type:jsonb" json:"context,omitempty"`
	CarrierSummary   *json.RawMessage `gorm:"column:carrier_summary;type:jsonb" json:"carrier_summary,omitempty"`
	FinancialSummary *json.RawMessage `gorm:"column:financial_summary;type:jsonb" json:"financial_summary,omitempty"`
	ErrorLog         *json.RawMessage `gorm:"column:error_log;type:jsonb" json:"error_log,omitempty"`
	CreatedAt        time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (SupplyChainFlow) TableName() string { return "supply_chain_flow" }

// ListFlowsRequest is the request query for listing flows.
type ListFlowsRequest struct {
	Page       int    `form:"page"`
	Size       int    `form:"size"`
	SourceType string `form:"source_type"`
	Status     string `form:"status"`
}

// UpdateFlowRequest is the request body for updating a flow.
type UpdateFlowRequest struct {
	Status           string           `json:"status"`
	Context          *json.RawMessage `json:"context,omitempty"`
	CarrierSummary   *json.RawMessage `json:"carrier_summary,omitempty"`
	FinancialSummary *json.RawMessage `json:"financial_summary,omitempty"`
	ErrorLog         *json.RawMessage `json:"error_log,omitempty"`
}
