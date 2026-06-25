package productanalysis

import "time"

// ProductAnalysis maps to "product_analysis" table.
type ProductAnalysis struct {
	ID                int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourcingProductID int64      `gorm:"column:sourcing_product_id;not null;index" json:"sourcing_product_id"`
	TargetSalePrice   float64    `gorm:"column:target_sale_price;type:decimal(12,2);not null" json:"target_sale_price"`
	EstimatedCost     float64    `gorm:"column:estimated_cost;type:decimal(12,2);not null;default:0" json:"estimated_cost"`
	EstimatedMargin   *float64   `gorm:"column:estimated_profit_margin;type:decimal(5,2)" json:"estimated_profit_margin,omitempty"`
	DemandScore       *float64   `gorm:"column:demand_score;type:decimal(5,2)" json:"demand_score,omitempty"`
	DemandStatus      string     `gorm:"column:demand_score_status;type:varchar(20);default:no_data" json:"demand_score_status"`
	CompetitionIdx    *float64   `gorm:"column:competition_index;type:decimal(5,2)" json:"competition_index,omitempty"`
	CompetitionStatus string     `gorm:"column:competition_status;type:varchar(20);default:no_data" json:"competition_status"`
	Status            string     `gorm:"column:analysis_status;type:varchar(20);default:pending" json:"analysis_status"`
	ErrorMessage      string     `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	AnalyzedBy        string     `gorm:"column:analyzed_by;type:varchar(255);not null" json:"analyzed_by"`
	AnalyzedAt        *time.Time `gorm:"column:analyzed_at" json:"analyzed_at,omitempty"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ProductAnalysis) TableName() string { return "product_analysis" }

// AnalysisFeedback is an immutable audit log entry.
type AnalysisFeedback struct {
	ID                int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductAnalysisID int64     `gorm:"column:product_analysis_id;not null;index" json:"product_analysis_id"`
	UserID            string    `gorm:"column:user_id;type:varchar(255);not null" json:"user_id"`
	Decision          string    `gorm:"column:decision;type:varchar(20);not null" json:"decision"`
	ActualMargin      *float64  `gorm:"column:actual_profit_margin;type:decimal(5,2)" json:"actual_profit_margin,omitempty"`
	Notes             string    `gorm:"column:notes;type:text" json:"notes,omitempty"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (AnalysisFeedback) TableName() string { return "analysis_feedback" }

// --- request / response types ---

type AnalyzeInput struct {
	SourcingProductID int64   `json:"sourcing_product_id" binding:"required"`
	TargetSalePrice   float64 `json:"target_sale_price" binding:"required,gt=0"`
}

type FeedbackInput struct {
	Decision     string   `json:"decision" binding:"required,oneof=imported abandoned"`
	ActualMargin *float64 `json:"actual_profit_margin"`
	Notes        string   `json:"notes"`
}

type ListFilter struct {
	UserID string
	Status string
	Page   int
	Size   int
}

// AnalysisResult wraps a completed analysis for API responses.
type AnalysisResult struct {
	Analysis          ProductAnalysis `json:"analysis"`
	ProfitScore       *float64        `json:"profit_score,omitempty"`
	DemandScore       *float64        `json:"demand_score,omitempty"`
	DemandScoreStatus string          `json:"demand_score_status"`
	CompetitionScore  *float64        `json:"competition_score,omitempty"`
	CompetitionStatus string          `json:"competition_status"`
	Warning           string          `json:"warning,omitempty"`
}
