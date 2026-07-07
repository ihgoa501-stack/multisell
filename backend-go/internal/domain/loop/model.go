package loop

import (
	"time"
)

// ListingRecommendation is the result of the evaluation loop for a single product.
type ListingRecommendation struct {
	ID                int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID         int64     `gorm:"column:product_id;not null;index" json:"product_id"`
	CompletenessScore float64   `gorm:"column:completeness_score;default:0" json:"completeness_score"`
	ProfitMargin      float64   `gorm:"column:profit_margin;default:0" json:"profit_margin"`
	EstimatedProfit   float64   `gorm:"column:estimated_profit;default:0" json:"estimated_profit"`
	Decision          string    `gorm:"column:decision;not null" json:"decision"` // list, cautious, skip
	Confidence        float64   `gorm:"column:confidence;default:0" json:"confidence"`
	Reason            string    `gorm:"column:reason;type:text" json:"reason"`
	RiskFlags         string    `gorm:"column:risk_flags;type:text" json:"risk_flags"` // JSON array of risk flags
	CreatedListingTaskID *int64 `gorm:"column:created_listing_task_id" json:"created_listing_task_id,omitempty"`
	TriggeredBy       string    `gorm:"column:triggered_by" json:"triggered_by"`
	FeedbackStatus    string    `gorm:"column:feedback_status;default:pending" json:"feedback_status"`
	// pending, adopted, rejected, executed, execution_failed
	FeedbackNote      string    `gorm:"column:feedback_note;type:text" json:"feedback_note,omitempty"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ListingRecommendation) TableName() string { return "listing_recommendation" }

// EvaluateResult is the API response for a complete evaluation.
type EvaluateResult struct {
	ProductID           int64           `json:"product_id"`
	Title               string          `json:"title"`
	CompletenessScore   float64         `json:"completeness_score"`
	CompletenessStatus  string          `json:"completeness_status"`
	MissingItems        []string        `json:"missing_items"`
	ProfitMargin        float64         `json:"profit_margin"`
	EstimatedProfit     float64         `json:"estimated_profit"`
	ProfitStatus        string          `json:"profit_status"`
	Decision            string          `json:"decision"`
	Confidence          float64         `json:"confidence"`
	Reason              string          `json:"reason"`
	RiskFlags           []string        `json:"risk_flags"`
	ListingTaskID       *int64          `json:"listing_task_id,omitempty"`
	Error               string          `json:"error,omitempty"`
}

// EvaluateInput is the payload for triggering an evaluation.
type EvaluateInput struct {
	TriggeredBy string `json:"triggered_by"`
}

// RecommendationFeedbackSummary aggregates feedback on listing recommendations.
type RecommendationFeedbackSummary struct {
	TotalRecommendations int          `json:"total_recommendations"`
	AdoptedCount         int          `json:"adopted_count"`
	RejectedCount        int          `json:"rejected_count"`
	ExecutedCount        int          `json:"executed_count"`
	FailedCount          int          `json:"failed_count"`
	AdoptRate            float64      `json:"adopt_rate"`
	SuccessRate          float64      `json:"success_rate"`
	Reviews              []ReviewItem `json:"reviews"`
	LastUpdated          time.Time    `json:"last_updated"`
}

// ExecutionReviewData carries detailed execution metadata for V2 results.
type ExecutionReviewData struct {
	ExecutionMode       string `json:"execution_mode"`
	DurationMs          int64  `json:"duration_ms"`
	FailureType         string `json:"failure_type,omitempty"`
	PlatformReferenceID string `json:"platform_reference_id,omitempty"`
	IsRetry             bool   `json:"is_retry,omitempty"`
	ExternalReferenceID string `json:"external_reference_id,omitempty"`
}

// ReviewItem is a single listing recommendation review entry.
type ReviewItem struct {
	RecommendationID int64     `json:"recommendation_id"`
	Decision         string    `json:"decision"`
	Confidence       float64   `json:"confidence"`
	FeedbackStatus   string    `json:"feedback_status"`
	ExecutionSuccess *bool     `json:"execution_success,omitempty"`
	FailureType      string    `json:"failure_type,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}
