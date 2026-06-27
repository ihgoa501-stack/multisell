package productanalysis

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// svc implements the Service interface.
type svc struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new productanalysis service.
func NewService(db *gorm.DB, logger *zap.Logger) Service {
	return &svc{db: db, logger: logger}
}

// sourcingProduct is a minimal view of a Sourcing1688Product for analysis.
// We query it directly rather than importing the sourcing1688 package to
// avoid a circular dependency and keep the domain boundary clean.
type sourcingProduct struct {
	ID     int64
	Price  float64 `gorm:"column:price"`
	Status string
}

func (sourcingProduct) TableName() string { return "sourcing_1688_product" }

// Analyze performs a full product analysis with 24h caching.
func (s *svc) Analyze(in *AnalyzeInput, userID string) (*AnalysisResult, error) {
	// 0. Cache check: same product + same user within 24h → return cached
	var cached ProductAnalysis
	cutoff := time.Now().Add(-24 * time.Hour)
	cacheErr := s.db.Where(
		"sourcing_product_id = ? AND analyzed_by = ? AND created_at >= ?",
		in.SourcingProductID, userID, cutoff,
	).Order("id DESC").First(&cached).Error
	if cacheErr == nil {
		// Rebuild AnalysisResult from cached record
		profitScore := cached.EstimatedMargin
		if profitScore != nil {
			// Re-derive score from stored margin
			s := *profitScore / 50.0 * 100
			if s > 100 {
				s = 100
			}
			if s < 0 {
				s = 0
			}
			profitScore = &s
		}
		return &AnalysisResult{
			Analysis:          cached,
			ProfitScore:       profitScore,
			DemandScore:       cached.DemandScore,
			DemandScoreStatus: cached.DemandStatus,
			CompetitionScore:  cached.CompetitionIdx,
			CompetitionStatus: cached.CompetitionStatus,
			Warning:           "此为估算结果，实际利润可能因市场变化而异。",
		}, nil
	}
	// 1. Fetch the sourcing product
	var src sourcingProduct
	if err := s.db.First(&src, in.SourcingProductID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("sourcing product %d not found", in.SourcingProductID)
		}
		return nil, fmt.Errorf("fetch sourcing product: %w", err)
	}

	// 2. Calculate profit
	marginPct, profitScore := CalculateProfitMargin(in.TargetSalePrice, src.Price)

	// 3. Demand & competition (stubs — return no_data until Phase 0)
	demandScore, demandStatus := CalculateDemandScore()
	compScore, compStatus := CalculateCompetitionScore()

	// 4. Persist the analysis
	estCost := src.Price
	now := time.Now()
	rec := ProductAnalysis{
		SourcingProductID: in.SourcingProductID,
		TargetSalePrice:   in.TargetSalePrice,
		EstimatedCost:     estCost,
		EstimatedMargin:   marginPct,
		DemandScore:       demandScore,
		DemandStatus:      demandStatus,
		CompetitionIdx:    compScore,
		CompetitionStatus: compStatus,
		Status:            "completed",
		AnalyzedBy:        userID,
		AnalyzedAt:        &now,
	}
	if err := s.db.Create(&rec).Error; err != nil {
		return nil, fmt.Errorf("save analysis: %w", err)
	}

	// 5. Build response
	disclaimer := "此为估算结果，实际利润可能因市场变化而异。"

	result := &AnalysisResult{
		Analysis:          rec,
		ProfitScore:       profitScore,
		DemandScore:       demandScore,
		DemandScoreStatus: demandStatus,
		CompetitionScore:  compScore,
		CompetitionStatus: compStatus,
		Warning:           disclaimer,
	}

	s.logger.Info("product analysis completed",
		zap.Int64("sourcing_product_id", in.SourcingProductID),
		zap.String("user_id", userID),
		zap.Float64("margin_pct", valueOrZero(marginPct)),
	)

	return result, nil
}

// GetAnalysis returns one analysis, scoped to user.
func (s *svc) GetAnalysis(id int64, userID string) (*ProductAnalysis, error) {
	var a ProductAnalysis
	if err := s.db.Where("id = ? AND analyzed_by = ?", id, userID).First(&a).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("analysis %d not found for user", id)
		}
		return nil, fmt.Errorf("fetch analysis: %w", err)
	}
	return &a, nil
}

// ListAnalyses returns paginated analyses for a user.
func (s *svc) ListAnalyses(filter *ListFilter) ([]ProductAnalysis, int64, error) {
	q := s.db.Model(&ProductAnalysis{}).Where("analyzed_by = ?", filter.UserID)
	if filter.Status != "" {
		q = q.Where("analysis_status = ?", filter.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	size := filter.Size
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size
	var items []ProductAnalysis
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// RecordFeedback inserts an immutable audit log entry.
func (s *svc) RecordFeedback(analysisID int64, in *FeedbackInput, userID string) error {
	// Verify the analysis exists and belongs to user
	_, err := s.GetAnalysis(analysisID, userID)
	if err != nil {
		return err
	}

	fb := AnalysisFeedback{
		ProductAnalysisID: analysisID,
		UserID:            userID,
		Decision:          in.Decision,
		ActualMargin:      in.ActualMargin,
		Notes:             in.Notes,
	}
	if err := s.db.Create(&fb).Error; err != nil {
		return fmt.Errorf("save feedback: %w", err)
	}
	return nil
}

func valueOrZero(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
