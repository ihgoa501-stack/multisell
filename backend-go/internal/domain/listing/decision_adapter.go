package listing

import (
	"context"

	"github.com/lingmirror/backend-go/internal/domain/decision"
	"gorm.io/gorm"
)

// decisionReaderAdapter implements DecisionReader by querying the decision table via GORM.
type decisionReaderAdapter struct {
	db *gorm.DB
}

// NewDecisionReader creates a new DecisionReader backed by a *gorm.DB.
func NewDecisionReader(db *gorm.DB) DecisionReader {
	return &decisionReaderAdapter{db: db}
}

// GetByIDs fetches pre-listing decisions by their IDs and returns them as
// listing.PreListingDecision values.
func (a *decisionReaderAdapter) GetByIDs(ctx context.Context, ids []int64) ([]PreListingDecision, error) {
	var rows []decision.PreListingDecision
	if err := a.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]PreListingDecision, len(rows))
	for i, r := range rows {
		result[i] = PreListingDecision{
			ID:          r.ID,
			SkuID:       r.SkuID,
			PlatformID:  r.PlatformID,
			CountryCode: r.CountryCode,
		}
	}
	return result, nil
}
