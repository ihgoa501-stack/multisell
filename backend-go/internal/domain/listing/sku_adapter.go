package listing

import (
	"context"

	"github.com/lingmirror/backend-go/internal/domain/sku"
	"gorm.io/gorm"
)

// skuProviderAdapter implements SKUProvider by querying the sku table via GORM.
type skuProviderAdapter struct {
	db *gorm.DB
}

// NewSKUProvider creates a new SKUProvider backed by a *gorm.DB.
func NewSKUProvider(db *gorm.DB) SKUProvider {
	return &skuProviderAdapter{db: db}
}

// GetByIDs fetches skus by their IDs and returns them as listing.Sku values.
func (a *skuProviderAdapter) GetByIDs(ctx context.Context, ids []int64) ([]Sku, error) {
	var rows []sku.Sku
	if err := a.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]Sku, len(rows))
	for i, r := range rows {
		result[i] = Sku{
			ID:        r.ID,
			ProductID: r.ProductID,
		}
	}
	return result, nil
}
