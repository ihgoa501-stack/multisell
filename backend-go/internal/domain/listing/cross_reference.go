package listing

import (
	"fmt"

	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"gorm.io/gorm"
)

// GetListingByTaskID retrieves the ProductListing associated with a listing task.
func (s *Service) GetListingByTaskID(taskID int64) (*ProductListing, error) {
	var task listingtask.ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, fmt.Errorf("get listing task: %w", err)
	}
	if task.ProductListingID == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var l ProductListing
	if err := s.db.First(&l, *task.ProductListingID).Error; err != nil {
		return nil, fmt.Errorf("get product listing from task: %w", err)
	}
	return &l, nil
}

// GetListingByExternalRef retrieves a ProductListing by its platform-assigned product ID.
func (s *Service) GetListingByExternalRef(externalRefID string) (*ProductListing, error) {
	var l ProductListing
	if err := s.db.Where("platform_product_id = ?", externalRefID).First(&l).Error; err != nil {
		return nil, fmt.Errorf("get listing by external ref %q: %w", externalRefID, err)
	}
	return &l, nil
}
