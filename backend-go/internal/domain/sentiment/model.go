package sentiment

import "time"

// ProductSentiment maps to "product_sentiment" table.
type ProductSentiment struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID       int64     `gorm:"column:product_id;not null;uniqueIndex" json:"product_id"`
	AvgRating       float64   `gorm:"column:avg_rating;default:0" json:"avg_rating"`
	ReviewCount     int       `gorm:"column:review_count;default:0" json:"review_count"`
	PositivePct     float64   `gorm:"column:positive_pct;default:0" json:"positive_pct"`
	NegativePct     float64   `gorm:"column:negative_pct;default:0" json:"negative_pct"`
	ReturnRate      float64   `gorm:"column:return_rate;default:0" json:"return_rate"`
	TopComplaints   string    `gorm:"column:top_complaints;type:text" json:"top_complaints,omitempty"`
	SentimentScore  float64   `gorm:"column:sentiment_score;default:0" json:"sentiment_score"`
	LastUpdated     time.Time `gorm:"column:last_updated" json:"last_updated"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (ProductSentiment) TableName() string { return "product_sentiment" }

// --- Response types ---

// SentimentResponse is the public API response for a product sentiment record.
type SentimentResponse struct {
	ProductSentiment
	ProductName string `json:"product_name,omitempty"`
}

// NegativeSentimentItem is a lightweight DTO for the negative sentiment list endpoint.
type NegativeSentimentItem struct {
	ProductID      int64   `json:"product_id"`
	ProductName    string  `json:"product_name"`
	SentimentScore float64 `json:"sentiment_score"`
	ReviewCount    int     `json:"review_count"`
	ReturnRate     float64 `json:"return_rate"`
	AvgRating      float64 `json:"avg_rating"`
	PositivePct    float64 `json:"positive_pct"`
	NegativePct    float64 `json:"negative_pct"`
}

// TopComplaint is a single complaint category with frequency.
type TopComplaint struct {
	Category  string  `json:"category"`
	Frequency int     `json:"frequency"`
	Pct       float64 `json:"pct"`
}

// TopComplaintsResponse wraps the complaints list.
type TopComplaintsResponse struct {
	ProductID  int64          `json:"product_id"`
	Complaints []TopComplaint `json:"complaints"`
}
