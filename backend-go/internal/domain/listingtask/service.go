package listingtask

import (
	"errors"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides listing task business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new listingtask service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ---------- ListingTask ----------

// List returns paginated listing tasks with filters.
func (s *Service) List(c *common.Pagination, platformID, productID *int64, status, search string) ([]ListingTask, int64, error) {
	var items []ListingTask
	var total int64

	q := s.db.Model(&ListingTask{})
	if platformID != nil {
		q = q.Where("platform_id = ?", *platformID)
	}
	if productID != nil {
		q = q.Where("product_id = ?", *productID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("source_item_key ILIKE ? OR destination_country ILIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a single listing task by id, with its items.
func (s *Service) GetByID(id int64) (*ListingTask, []ListingTaskItem, error) {
	var t ListingTask
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, nil, err
	}
	var items []ListingTaskItem
	s.db.Where("task_id = ?", id).Order("id ASC").Find(&items)
	return &t, items, nil
}

// Create inserts a new listing task.
func (s *Service) Create(in *CreateTaskInput) (*ListingTask, error) {
	t := ListingTask{
		ProductID:           in.ProductID,
		PlatformID:          in.PlatformID,
		SkuID:               in.SkuID,
		ProductListingID:    in.ProductListingID,
		SourceType:          in.SourceType,
		SourceItemKey:       in.SourceItemKey,
		Status:              in.Status,
		MissingRequirements: in.MissingRequirements,
		DecisionSnapshot:    in.DecisionSnapshot,
		TargetSalePrice:     in.TargetSalePrice,
		TargetProfitMargin:  in.TargetProfitMargin,
		DestinationCountry:  in.DestinationCountry,
		CreatedBy:           in.CreatedBy,
	}
	if t.SourceType == "" {
		t.SourceType = "decision"
	}
	if t.Status == "" {
		t.Status = "blocked"
	}
	if err := s.db.Create(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// Update patches a listing task by id.
func (s *Service) Update(id int64, in *UpdateTaskInput) (*ListingTask, error) {
	var t ListingTask
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.SourceItemKey != nil {
		updates["source_item_key"] = *in.SourceItemKey
	}
	if in.MissingRequirements != nil {
		updates["missing_requirements"] = *in.MissingRequirements
	}
	if in.DecisionSnapshot != nil {
		updates["decision_snapshot"] = *in.DecisionSnapshot
	}
	if in.TargetSalePrice != nil {
		updates["target_sale_price"] = *in.TargetSalePrice
	}
	if in.TargetProfitMargin != nil {
		updates["target_profit_margin"] = *in.TargetProfitMargin
	}
	if in.DestinationCountry != nil {
		updates["destination_country"] = *in.DestinationCountry
	}
	if in.LastError != nil {
		updates["last_error"] = *in.LastError
	}
	if in.ProductListingID != nil {
		updates["product_listing_id"] = *in.ProductListingID
	}
	if in.UpdatedBy != nil {
		updates["updated_by"] = *in.UpdatedBy
	}
	if len(updates) == 0 {
		return &t, nil
	}
	if err := s.db.Model(&t).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// Delete removes a listing task by id (and its items via cascade expectation).
func (s *Service) Delete(id int64) error {
	res := s.db.Delete(&ListingTask{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetStats returns aggregate counts grouped by status across all listing tasks.
func (s *Service) GetStats() (map[string]int64, error) {
	type StatusCount struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:cnt"`
	}
	var rows []StatusCount
	if err := s.db.Model(&ListingTask{}).
		Select("status, COUNT(*) as cnt").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	stats := map[string]int64{
		"total":      0,
		"pending":    0,
		"processing": 0,
		"completed":  0,
		"failed":     0,
	}
	for _, r := range rows {
		switch r.Status {
		case "executing":
			stats["processing"] += r.Count
		default:
			stats[r.Status] = r.Count
		}
		stats["total"] += r.Count
	}
	return stats, nil
}

// RetryAllTasks finds all failed tasks and resets them to pending.
func (s *Service) RetryAllTasks() (int64, error) {
	res := s.db.Model(&ListingTask{}).
		Where("status = ?", "failed").
		Updates(map[string]interface{}{
			"status":     "pending",
			"last_error": "",
		})
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected > 0 {
		if err := s.db.Model(&ListingTaskItem{}).
			Where("status = ?", "failed").
			Updates(map[string]interface{}{
				"status":        "pending",
				"error_message": "",
				"retry_count":   gorm.Expr("retry_count + 1"),
			}).Error; err != nil {
			return res.RowsAffected, err
		}
	}
	return res.RowsAffected, nil
}

// ListItems returns paginated items for a task.
func (s *Service) ListItems(c *common.Pagination, taskID int64) ([]ListingTaskItem, int64, error) {
	var items []ListingTaskItem
	var total int64
	q := s.db.Model(&ListingTaskItem{}).Where("task_id = ?", taskID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id ASC").Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CreateItem inserts a new task item.
func (s *Service) CreateItem(in *CreateTaskItemInput) (*ListingTaskItem, error) {
	item := ListingTaskItem{
		TaskID:     in.TaskID,
		ProductID:  in.ProductID,
		PlatformID: in.PlatformID,
		Status:     in.Status,
		Result:     in.Result,
	}
	if item.Status == "" {
		item.Status = "pending"
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateItem patches a task item by id.
func (s *Service) UpdateItem(id int64, in *UpdateTaskItemInput) (*ListingTaskItem, error) {
	var item ListingTaskItem
	if err := s.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.Result != nil {
		updates["result"] = *in.Result
	}
	if in.ErrorMessage != nil {
		updates["error_message"] = *in.ErrorMessage
	}
	if in.RetryCount != nil {
		updates["retry_count"] = *in.RetryCount
	}
	if len(updates) == 0 {
		return &item, nil
	}
	if err := s.db.Model(&item).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// DeleteItem removes a task item by id.
func (s *Service) DeleteItem(id int64) error {
	res := s.db.Delete(&ListingTaskItem{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetItem returns a single task item by id.
func (s *Service) GetItem(id int64) (*ListingTaskItem, error) {
	var item ListingTaskItem
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &item, nil
}

// ---------- Listing publish chain ----------

// ExecuteTask triggers execution of a listing task: transitions status to
// executing, runs each item, and aggregates the final status to completed or
// failed based on item outcomes. In this stub each item is marked completed.
func (s *Service) ExecuteTask(taskID int64) (*ListingTask, error) {
	var task ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	if task.Status == "completed" || task.Status == "cancelled" {
		return &task, nil
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&task).Update("status", "executing").Error; err != nil {
			return err
		}
		var items []ListingTaskItem
		if err := tx.Where("task_id = ?", taskID).Find(&items).Error; err != nil {
			return err
		}
		now := time.Now()
		finalStatus := "completed"
		for i := range items {
			// In a full implementation this would call the platform adapter.
			updates := map[string]interface{}{
				"status":      "completed",
				"executed_at": &now,
			}
			if err := tx.Model(&items[i]).Updates(updates).Error; err != nil {
				return err
			}
		}
		if len(items) == 0 {
			// nothing to execute — leave completed only if no items failed
		}
		return tx.Model(&task).Update("status", finalStatus).Error
	})
	if err != nil {
		return nil, err
	}
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// RetryFailed resets all failed items of a task back to pending and increments
// their retry_count. The task status is reset to pending so it can be re-run.
func (s *Service) RetryFailed(taskID int64) (*ListingTask, error) {
	var task ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&ListingTaskItem{}).
			Where("task_id = ? AND status = ?", taskID, "failed").
			Updates(map[string]interface{}{
				"status":       "pending",
				"error_message": "",
				"retry_count":  gorm.Expr("retry_count + 1"),
			})
		if res.Error != nil {
			return res.Error
		}
		return tx.Model(&task).Updates(map[string]interface{}{
			"status":     "pending",
			"last_error": "",
		}).Error
	})
	if err != nil {
		return nil, err
	}
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// RetryItem resets a single listing_task_item to pending and increments its
// retry_count. The item must belong to the given task.
func (s *Service) RetryItem(taskID, itemID int64) (*ListingTaskItem, error) {
	var item ListingTaskItem
	if err := s.db.Where("id = ? AND task_id = ?", itemID, taskID).First(&item).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&item).Updates(map[string]interface{}{
		"status":       "pending",
		"error_message": "",
		"retry_count":  item.RetryCount + 1,
	}).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&item, itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
