package sentiment

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service holds business logic for product sentiment analysis.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new sentiment service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// CalculateSentiment computes and persists aggregated sentiment for a product.
// It pulls data from orders (ratings), returns (reason codes), and support
// conversations to produce a composite sentiment score (0-100).
func (s *Service) CalculateSentiment(productID int64) (*ProductSentiment, error) {
	s.logger.Info("calculating sentiment", zap.Int64("product_id", productID))

	// 1. Aggregate order ratings for this product.
	type ratingRow struct {
		AvgRating    float64
		ReviewCount  int
		PositiveRate float64
		NegativeRate float64
	}
	var rating ratingRow
	err := s.db.Raw(`
		SELECT
			COALESCE(AVG(rating), 0)                                    AS avg_rating,
			COUNT(*)                                                     AS review_count,
			COALESCE(SUM(CASE WHEN rating >= 4 THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(*), 0), 0) AS positive_rate,
			COALESCE(SUM(CASE WHEN rating <= 2 THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(*), 0), 0) AS negative_rate
		FROM order_item
		WHERE product_id = ? AND rating IS NOT NULL AND rating > 0
	`, productID).Scan(&rating).Error
	if err != nil {
		return nil, fmt.Errorf("aggregate ratings: %w", err)
	}

	// 2. Aggregate return rate for this product.
	type returnRow struct {
		TotalCount int64
		ReturnQty  int64
	}
	var ret returnRow
	err = s.db.Raw(`
		SELECT
			COUNT(*)                                               AS total_count,
			COALESCE(SUM(CASE WHEN oi.return_qty > 0 THEN 1 ELSE 0 END), 0) AS return_qty
		FROM order_item oi
		WHERE oi.product_id = ?
	`, productID).Scan(&ret).Error
	if err != nil {
		return nil, fmt.Errorf("aggregate returns: %w", err)
	}

	returnRate := 0.0
	if ret.TotalCount > 0 {
		returnRate = float64(ret.ReturnQty) / float64(ret.TotalCount) * 100
	}

	// 3. Compute composite sentiment score (0-100).
	//    Weighting: rating (50%) + return rate (30%) + review volume signal (20%)
	ratingScore := rating.AvgRating / 5.0 * 100 // 0-100

	// Invert: lower return rate = better score
	returnScore := 100.0 - returnRate
	if returnScore < 0 {
		returnScore = 0
	}

	// Volume bonus: more reviews = more confidence, but cap the bonus
	volumeBonus := math.Min(float64(rating.ReviewCount)*2, 20)

	sentimentScore := math.Round(ratingScore*0.50 + returnScore*0.30 + volumeBonus*0.20)
	if sentimentScore > 100 {
		sentimentScore = 100
	}
	if sentimentScore < 0 {
		sentimentScore = 0
	}

	// 4. Determine top complaints (stub for now — will use LLM categorization in production).
	complaints := []TopComplaint{
		{Category: "quality", Frequency: 0, Pct: 0},
		{Category: "shipping", Frequency: 0, Pct: 0},
		{Category: "size_fit", Frequency: 0, Pct: 0},
	}
	complaintsJSON, _ := json.Marshal(complaints)

	// 5. Upsert the sentiment record.
	now := time.Now()
	sentiment := ProductSentiment{
		ProductID:      productID,
		AvgRating:      math.Round(rating.AvgRating*100) / 100,
		ReviewCount:    rating.ReviewCount,
		PositivePct:    math.Round(rating.PositiveRate*100) / 100,
		NegativePct:    math.Round(rating.NegativeRate*100) / 100,
		ReturnRate:     math.Round(returnRate*100) / 100,
		TopComplaints:  string(complaintsJSON),
		SentimentScore: sentimentScore,
		LastUpdated:    now,
	}

	// Upsert: insert or update on product_id conflict.
	err = s.db.Where("product_id = ?", productID).Assign(sentiment).FirstOrCreate(&sentiment).Error
	if err != nil {
		return nil, fmt.Errorf("upsert sentiment: %w", err)
	}

	s.logger.Info("sentiment calculated",
		zap.Int64("product_id", productID),
		zap.Float64("score", sentimentScore),
		zap.Int("reviews", rating.ReviewCount),
		zap.Float64("return_rate", returnRate),
	)

	return &sentiment, nil
}

// GetSentiment returns the current sentiment for a product.
func (s *Service) GetSentiment(productID int64) (*SentimentResponse, error) {
	var record ProductSentiment
	err := s.db.Where("product_id = ?", productID).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No data yet — not an error
		}
		return nil, fmt.Errorf("fetch sentiment: %w", err)
	}

	// Enrich with product name.
	var productName string
	_ = s.db.Table("product").Select("name").Where("id = ?", productID).Scan(&productName).Error

	return &SentimentResponse{
		ProductSentiment: record,
		ProductName:      productName,
	}, nil
}

// TopComplaints returns parsed complaint categories for a product.
func (s *Service) TopComplaints(productID int64) (*TopComplaintsResponse, error) {
	var record ProductSentiment
	err := s.db.Where("product_id = ?", productID).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &TopComplaintsResponse{
				ProductID:  productID,
				Complaints: []TopComplaint{},
			}, nil
		}
		return nil, fmt.Errorf("fetch sentiment: %w", err)
	}

	var complaints []TopComplaint
	if record.TopComplaints != "" && record.TopComplaints != "[]" {
		if err := json.Unmarshal([]byte(record.TopComplaints), &complaints); err != nil {
			s.logger.Warn("failed to parse top_complaints JSON", zap.Error(err))
			complaints = []TopComplaint{}
		}
	}

	return &TopComplaintsResponse{
		ProductID:  productID,
		Complaints: complaints,
	}, nil
}

// ListNegativeSentiment returns all products with sentiment score < 50,
// ordered by score ascending.
func (s *Service) ListNegativeSentiment() ([]NegativeSentimentItem, error) {
	var items []NegativeSentimentItem

	err := s.db.Table("product_sentiment ps").
		Select(`
			ps.product_id,
			COALESCE(p.name, '') AS product_name,
			ps.sentiment_score,
			ps.review_count,
			ps.return_rate,
			ps.avg_rating,
			ps.positive_pct,
			ps.negative_pct
		`).
		Joins("LEFT JOIN product p ON p.id = ps.product_id").
		Where("ps.sentiment_score < 50").
		Order("ps.sentiment_score ASC").
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("list negative sentiment: %w", err)
	}

	if items == nil {
		items = []NegativeSentimentItem{}
	}
	return items, nil
}
