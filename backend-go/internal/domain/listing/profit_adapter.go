package listing

import (
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"gorm.io/gorm"
)

// profitReaderAdapter implements ProfitReader by querying the profit_summary table via GORM.
type profitReaderAdapter struct {
	db *gorm.DB
}

// NewProfitReader creates a new ProfitReader backed by a *gorm.DB.
func NewProfitReader(db *gorm.DB) ProfitReader {
	return &profitReaderAdapter{db: db}
}

// GetByProductID fetches the latest profit summary for a product and returns it as a listing.ProfitSummary.
func (a *profitReaderAdapter) GetByProductID(productID int64) (*ProfitSummary, error) {
	var ps profit.ProfitSummary
	if err := a.db.Where("product_id = ?", productID).Order("id DESC").First(&ps).Error; err != nil {
		return nil, err
	}
	return &ProfitSummary{
		TotalCost:       ps.TotalCost,
		TargetRevenue:   ps.TargetRevenue,
		EstimatedProfit: ps.EstimatedProfit,
		ProfitMargin:    ps.ProfitMargin,
		Status:          ps.Status,
	}, nil
}
