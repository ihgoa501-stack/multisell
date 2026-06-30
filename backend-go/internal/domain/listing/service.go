package listing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides listing business logic.
type Service struct {
	db              *gorm.DB
	logger          *zap.Logger
	skuProvider     SKUProvider
	decisionReader  DecisionReader
}

// NewService creates a new listing service.
func NewService(db *gorm.DB, logger *zap.Logger, skuProvider SKUProvider, decisionReader DecisionReader) *Service {
	return &Service{db: db, logger: logger, skuProvider: skuProvider, decisionReader: decisionReader}
}

// List returns paginated listings with platform/status filters and search.
func (s *Service) List(c *common.Pagination, platformID *int64, status, search string) ([]ProductListing, int64, error) {
	var items []ProductListing
	var total int64

	q := s.db.Model(&ProductListing{})
	if platformID != nil {
		q = q.Where("platform_id = ?", *platformID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("platform_product_id ILIKE ? OR platform_sku ILIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a single listing by id.
func (s *Service) GetByID(id int64) (*ProductListing, error) {
	var l ProductListing
	if err := s.db.First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// Create inserts a new listing.
func (s *Service) Create(in *CreateListingInput) (*ProductListing, error) {
	l := ProductListing{
		ProductID:         in.ProductID,
		PlatformID:        in.PlatformID,
		PlatformProductID: in.PlatformProductID,
		PlatformSKU:       in.PlatformSKU,
		Status:            in.Status,
		PlatformURL:       in.PlatformURL,
		PublishedData:     in.PublishedData,
	}
	if l.Status == "" {
		l.Status = "draft"
	}
	if err := s.db.Create(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// Update patches a listing by id.
func (s *Service) Update(id int64, in *UpdateListingInput) (*ProductListing, error) {
	var l ProductListing
	if err := s.db.First(&l, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.PlatformProductID != nil {
		updates["platform_product_id"] = *in.PlatformProductID
	}
	if in.PlatformSKU != nil {
		updates["platform_sku"] = *in.PlatformSKU
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.PlatformURL != nil {
		updates["platform_url"] = *in.PlatformURL
	}
	if in.SyncMessage != nil {
		updates["sync_message"] = *in.SyncMessage
	}
	if in.PublishedData != nil {
		updates["published_data"] = *in.PublishedData
	}
	if len(updates) == 0 {
		return &l, nil
	}
	if err := s.db.Model(&l).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// Delete removes a listing by id.
func (s *Service) Delete(id int64) error {
	res := s.db.Delete(&ProductListing{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Publish triggers the publish flow for a listing: sets status to pending and
// records published_data. In a full implementation this would enqueue a platform
// API call; here it transitions state transactionally.
func (s *Service) Publish(id int64, payload json.RawMessage) (*ProductListing, error) {
	var l ProductListing
	if err := s.db.First(&l, id).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":         "pending",
		"published_data": payload,
		"last_sync_at":   &now,
	}
	if err := s.db.Model(&l).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// SyncStatus pulls the latest status from the platform. In a full implementation
// this would call the platform adapter; here it updates last_sync_at and can
// optionally apply a provided status/sync_message.
func (s *Service) SyncStatus(id int64, newStatus, syncMessage string) (*ProductListing, error) {
	var l ProductListing
	if err := s.db.First(&l, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	now := time.Now()
	updates := map[string]interface{}{
		"last_sync_at": &now,
	}
	if newStatus != "" {
		updates["status"] = newStatus
	}
	if syncMessage != "" {
		updates["sync_message"] = syncMessage
	}
	if err := s.db.Model(&l).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// ---------- Listing publish chain ----------

// PublishProduct creates a product_listing record for the given product+platform
// and triggers a listing_task in a single transaction. It stores Prism configuration
// in published_data for use by ExecuteTask downstream.
func (s *Service) PublishProduct(productID, platformID int64, in *PublishProductInput) (*ProductListing, error) {
	status := in.Status
	if status == "" {
		status = "publishing"
	}

	// Build published_data with optional Prism configuration.
	pd := map[string]interface{}{
		"prism": map[string]interface{}{
			"enabled": in.PrismEnabled,
			"options": in.PrismOptions,
		},
	}
	pdBytes, _ := json.Marshal(pd)

	l := ProductListing{
		ProductID:         productID,
		PlatformID:        platformID,
		PlatformProductID: in.ExternalID,
		PlatformURL:       in.ListingURL,
		Status:            status,
		PublishedData:     pdBytes,
	}
	now := time.Now()
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&l).Error; err != nil {
			return err
		}
		task := listingtask.ListingTask{
			ProductID:         productID,
			PlatformID:        platformID,
			ProductListingID:  &l.ID,
			SourceType:        "manual",
			Status:            "pending",
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		return tx.Model(&l).Updates(map[string]interface{}{
			"last_sync_at": &now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// ListByProduct returns all product_listing records for a given product.
func (s *Service) ListByProduct(productID int64) ([]ProductListing, error) {
	var items []ProductListing
	if err := s.db.Where("product_id = ?", productID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CreateTasksFromDecisions batch-creates listing_task records from
// pre_listing_decision records. Resolves product_id via the sku table.
func (s *Service) CreateTasksFromDecisions(decisionIDs []int64) ([]listingtask.ListingTask, error) {
	if len(decisionIDs) == 0 {
		return nil, errors.New("decision_ids is required")
	}
	ctx := context.Background()
	decisions, err := s.decisionReader.GetByIDs(ctx, decisionIDs)
	if err != nil {
		return nil, err
	}
	if len(decisions) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	// Resolve sku_id -> product_id for all unique skus.
	skuIDs := make([]int64, 0, len(decisions))
	for _, d := range decisions {
		skuIDs = append(skuIDs, d.SkuID)
	}
	skus, err := s.skuProvider.GetByIDs(ctx, skuIDs)
	if err != nil {
		return nil, err
	}
	skuProduct := make(map[int64]int64, len(skus))
	for _, sk := range skus {
		skuProduct[sk.ID] = sk.ProductID
	}

	tasks := make([]listingtask.ListingTask, 0, len(decisions))
	err = s.db.Transaction(func(tx *gorm.DB) error {
		for _, d := range decisions {
			productID, ok := skuProduct[d.SkuID]
			if !ok {
				return fmt.Errorf("sku not found for decision %d (sku_id=%d)", d.ID, d.SkuID)
			}
			var platformID int64
			if d.PlatformID != nil {
				platformID = *d.PlatformID
			}
			task := listingtask.ListingTask{
				ProductID:          productID,
				PlatformID:         platformID,
				SourceType:         "decision",
				SourceItemKey:      fmt.Sprintf("decision:%d", d.ID),
				Status:             "pending",
				DestinationCountry: d.CountryCode,
			}
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
			tasks = append(tasks, task)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// RecheckTask re-runs validation on a listing_task and updates its status.
// In this implementation it re-evaluates missing_requirements and transitions
// blocked → pending when requirements are empty.
func (s *Service) RecheckTask(taskID int64) (*listingtask.ListingTask, error) {
	var task listingtask.ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if len(task.MissingRequirements) == 0 || string(task.MissingRequirements) == "[]" {
		updates["status"] = "pending"
	} else {
		updates["status"] = "blocked"
	}
	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// CancelTask marks a listing_task as cancelled and records the reason.
func (s *Service) CancelTask(taskID int64, reason string) (*listingtask.ListingTask, error) {
	var task listingtask.ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"status":     "cancelled",
		"last_error": reason,
	}
	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// PublishTask pushes a pending listing_task into executing state.
func (s *Service) PublishTask(taskID int64) (*listingtask.ListingTask, error) {
	var task listingtask.ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	if task.Status != "pending" {
		return nil, fmt.Errorf("task %d is not pending (current=%s)", taskID, task.Status)
	}
	if err := s.db.Model(&task).Update("status", "executing").Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}
