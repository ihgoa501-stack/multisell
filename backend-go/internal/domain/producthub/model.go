package producthub

import (
	"encoding/json"
	"time"
)

// ProductVersion maps to the "product_version" table — snapshot history for rollback.
type ProductVersion struct {
	ID          int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID   int64           `gorm:"column:product_id;not null;index" json:"product_id"`
	VersionData json.RawMessage `gorm:"column:version_data;type:jsonb" json:"version_data"`
	Snapshot    json.RawMessage `gorm:"column:snapshot;type:jsonb" json:"snapshot"`
	AgentID     string          `gorm:"column:agent_id" json:"agent_id"`
	Reason      string          `gorm:"column:reason" json:"reason"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName explicitly sets the table name.
func (ProductVersion) TableName() string { return "product_version" }

// ProductRelation maps to the "product_relation" table — relationship graph between products.
type ProductRelation struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourceID       int64     `gorm:"column:source_id;not null;index" json:"source_id"`
	TargetID       int64     `gorm:"column:target_id;not null;index" json:"target_id"`
	RelationType   string    `gorm:"column:relation_type;not null" json:"relation_type"` // variant, replacement, bundle, cross_sell, alternative, accessory
	Weight         float64   `gorm:"column:weight;default:0" json:"weight"`
	AutoDiscovered bool      `gorm:"column:auto_discovered;default:true" json:"auto_discovered"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName explicitly sets the table name.
func (ProductRelation) TableName() string { return "product_relation" }

// --- Request / Response types ---

// VersionListResponse wraps paginated version results.
type VersionListResponse struct {
	Items []ProductVersion `json:"items"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
}

// DecisionRecordInput is the payload to record an agent decision with snapshot.
type DecisionRecordInput struct {
	ProductID  int64   `json:"product_id" binding:"required"`
	AgentID    string  `json:"agent_id" binding:"required"`
	Action     string  `json:"action"`
	Reasoning  string  `json:"reasoning"`
	Confidence float64 `json:"confidence"`
}

// DecisionRecord is the recorded decision trace entry.
type DecisionRecord struct {
	AgentID    string  `json:"agent_id"`
	Action     string  `json:"action"`
	Reasoning  string  `json:"reasoning"`
	Confidence float64 `json:"confidence"`
	CreatedAt  string  `json:"created_at"`
}

// RelationRequest is the payload to create a manual product relation.
type RelationRequest struct {
	SourceID     int64   `json:"source_id" binding:"required"`
	TargetID     int64   `json:"target_id" binding:"required"`
	RelationType string  `json:"relation_type" binding:"required"`
	Weight       float64 `json:"weight"`
}

// RelatedProductResponse wraps a related product with its relation metadata.
type RelatedProductResponse struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	MainImage      string  `json:"main_image"`
	RelationType   string  `json:"relation_type"`
	Weight         float64 `json:"weight"`
	AutoDiscovered bool    `json:"auto_discovered"`
}

// RelationListResponse wraps grouped relation results for a product.
type RelationListResponse struct {
	SourceID int64           `json:"source_id"`
	Groups   []RelationGroup `json:"groups"`
}

// RelationGroup groups related products by type.
type RelationGroup struct {
	RelationType string                   `json:"relation_type"`
	Label        string                   `json:"label"`
	Items        []RelatedProductResponse `json:"items"`
}

// RelationLabels maps relation types to Chinese display labels.
var RelationLabels = map[string]string{
	"variant":     "变体",
	"replacement": "替换",
	"bundle":      "捆绑销售",
	"cross_sell":  "关联购买",
	"alternative": "替代品",
	"accessory":   "配件",
}
