package listingtask

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/platform"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"github.com/lingmirror/backend-go/internal/prismadapter"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides listing task business logic.

// Service provides listing task business logic.
type Service struct {
	db          *gorm.DB
	logger      *zap.Logger
	prismSvc    prismadapter.PrismService // nil = Prism disabled at runtime
	prismStrict bool                      // block on Prism error vs warn+continue
	approvalSvc *approval.Service
	oplogSvc    *operationlog.Service
	loopRec     LoopRecorder
	publishHook PublishHook // nil = skip platform publish, task stays "completed"
}

// LoopRecorder is the interface for recording feedback back to the evaluation loop.
// Defined here to avoid circular import with the loop package.
type LoopRecorder interface {
	RecordExecutionResult(productID int64, listingTaskID int64, success bool, errorMsg string) error
}

// NewService creates a new listingtask service.
// prismSvc may be nil, in which case Prism is never called.
// oplogSvc may be nil (audit logging disabled).
// loopRec may be nil (feedback recording disabled).
func NewService(db *gorm.DB, logger *zap.Logger, prismSvc prismadapter.PrismService, prismStrict bool, approvalSvc *approval.Service, oplogSvc *operationlog.Service, loopRec LoopRecorder) *Service {
	return &Service{
		db:          db,
		logger:      logger,
		prismSvc:    prismSvc,
		prismStrict: prismStrict,
		approvalSvc: approvalSvc,
		oplogSvc:    oplogSvc,
		loopRec:     loopRec,
	}
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
		ProductID:            in.ProductID,
		PlatformID:           in.PlatformID,
		SkuID:                in.SkuID,
		ProductListingID:     in.ProductListingID,
		SourceType:           in.SourceType,
		SourceItemKey:        in.SourceItemKey,
		Status:               in.Status,
		MissingRequirements:  in.MissingRequirements,
		DecisionSnapshot:     in.DecisionSnapshot,
		TargetSalePrice:      in.TargetSalePrice,
		TargetProfitMargin:   in.TargetProfitMargin,
		DestinationCountry:   in.DestinationCountry,
		ApprovalID:           in.ApprovalID,
		CreatedBy:            in.CreatedBy,
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

// CreateFromSuggestion creates a controlled listing task from a candidate suggestion.
// It validates the candidate exists, requires approval, creates the task with
// pending_approval status, and writes an audit log.
func (s *Service) CreateFromSuggestion(candidateID uint, operator string) (*ListingTask, error) {
	// 1. Look up candidate product.
	type candidateRow struct {
		ID                 int64
		Title              string
		PurchasePrice      float64
		TargetSalePrice    float64
		TargetPlatformID   *int64
		DestinationCountry string
	}
	var row candidateRow
	if err := s.db.Table("candidate_product").
		Select("id, title, purchase_price, target_sale_price, target_platform_id, destination_country").
		Where("id = ?", candidateID).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("candidate lookup failed: %w", err)
	}
	if row.ID == 0 {
		return nil, fmt.Errorf("candidate %d not found", candidateID)
	}

	// 2. Build decision snapshot from candidate data.
	snapshot := map[string]interface{}{
		"candidate_id":      candidateID,
		"title":             row.Title,
		"purchase_price":    row.PurchasePrice,
		"target_sale_price": row.TargetSalePrice,
		"source":            "suggestion",
	}
	snapshotBytes, _ := json.Marshal(snapshot)

	// 3. Require approval via the approval service.
	if s.approvalSvc == nil {
		return nil, errors.New("approval service not configured")
	}
	apprReq, err := s.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
		ProductID:   int64(candidateID),
		RequestType: "listing_task",
		Requester:   operator,
		NewValue:    fmt.Sprintf("create listing task from candidate %d", candidateID),
		Reason:      "listing task creation requires owner approval",
		TargetType:  "candidate",
		TargetID:    int64(candidateID),
		RiskLevel:   "high",
		EntityType:  "listing_task",
	})
	if err != nil {
		return nil, fmt.Errorf("approval required: %w", err)
	}

	// 4. Create listing task with pending_approval status.
	platformID := int64(0)
	if row.TargetPlatformID != nil {
		platformID = *row.TargetPlatformID
	}
	task := ListingTask{
		ProductID:          int64(candidateID),
		PlatformID:         platformID,
		SourceType:         "suggestion",
		SourceItemKey:      fmt.Sprintf("candidate:%d", candidateID),
		Status:             "pending_approval",
		DecisionSnapshot:   snapshotBytes,
		TargetSalePrice:    &row.TargetSalePrice,
		DestinationCountry: row.DestinationCountry,
		ApprovalID:         &apprReq.ID,
		CreatedBy:          operator,
	}
	if err := s.db.Create(&task).Error; err != nil {
		return nil, err
	}

	// Update the approval entity ID now that the task exists.
	s.db.Model(&approval.ApprovalRequest{}).Where("id = ?", apprReq.ID).
		Update("entity_id", task.ID)

	// 5. Write audit log.
	s.writeAudit("listing_task.create_from_suggestion", "success",
		fmt.Sprintf("%d", task.ID), operator,
		fmt.Sprintf("candidate_id=%d listing_task_id=%d platform_id=%d", candidateID, task.ID, task.PlatformID),
		&task)

	return &task, nil
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
	if in.ApprovalID != nil {
		updates["approval_id"] = *in.ApprovalID
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

// RetryFailed resets all failed items of a task and the task itself to pending.
func (s *Service) RetryFailed(taskID int64) (*ListingTask, error) {
	var task ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&ListingTaskItem{}).
			Where("task_id = ? AND status = ?", taskID, "failed").
			Updates(map[string]interface{}{
				"status":        "pending",
				"error_message": "",
				"retry_count":   gorm.Expr("retry_count + 1"),
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

// RetryItem resets a single listing_task_item to pending and increments retry_count.
func (s *Service) RetryItem(taskID, itemID int64) (*ListingTaskItem, error) {
	var item ListingTaskItem
	if err := s.db.Where("id = ? AND task_id = ?", itemID, taskID).First(&item).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&item).Updates(map[string]interface{}{
		"status":        "pending",
		"error_message": "",
		"retry_count":   item.RetryCount + 1,
	}).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&item, itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// ---------- Execution Gate / State Machine ----------

// validateExecutePreconditions checks all preconditions before executing a listing task.
// Order of checks:
//  1. Task exists (caller responsibility — task is already loaded)
//  2. Idempotency: completed → return success without error
//  3. Idempotency: executing → return error
//  4. State machine: status must allow transition to "executing"
//  5. ApprovalID must be set
//  6. Approval record must exist, be approved, and match EntityType/EntityID
func (s *Service) validateExecutePreconditions(task *ListingTask) error {
	// Idempotency: already completed
	if task.Status == "completed" {
		return nil // not an error — caller treats as success
	}
	// Idempotency: already executing
	if task.Status == "executing" {
		return fmt.Errorf("task %d is already being executed", task.ID)
	}
	// State machine: only approved → executing is valid
	sm := NewListingTaskStateMachine()
	if !sm.CanTransition(task.Status, "executing") {
		return fmt.Errorf("listing task %d cannot be executed from status %s (must be approved)", task.ID, task.Status)
	}
	// ApprovalID must be set
	if task.ApprovalID == nil {
		return fmt.Errorf("listing task %d has no approval_id — must be approved before execution", task.ID)
	}
	// Approval record must exist and be approved
	if s.approvalSvc == nil {
		return fmt.Errorf("approval service is not available")
	}
	approvalRec, err := s.approvalSvc.Get(*task.ApprovalID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("approval %d for listing task %d not found", *task.ApprovalID, task.ID)
		}
		return fmt.Errorf("failed to load approval %d: %w", *task.ApprovalID, err)
	}
	if approvalRec.Status != "approved" {
		return fmt.Errorf("approval %d for listing task %d is not approved (status: %s)", *task.ApprovalID, task.ID, approvalRec.Status)
	}
	if approvalRec.EntityType != "listing_task" {
		return fmt.Errorf("approval %d entity_type is %q, expected listing_task", *task.ApprovalID, approvalRec.EntityType)
	}
	if approvalRec.EntityID != task.ID {
		return fmt.Errorf("approval %d entity_id is %d, expected listing task %d", *task.ApprovalID, approvalRec.EntityID, task.ID)
	}
	// 7. Execution mode — check DecisionSnapshot for mode flag
	if len(task.DecisionSnapshot) > 0 {
		var snapshot struct {
			Mode string `json:"mode"`
		}
		if err := json.Unmarshal(task.DecisionSnapshot, &snapshot); err == nil {
			switch snapshot.Mode {
			case "dry_run":
				task.DryRun = true
				task.ExecutionMode = ExecutionModeDryRun
				s.logger.Info("listing task in dry-run mode, skipping real publish",
					zap.Int64("task_id", task.ID),
				)
			case "sandbox":
				task.ExecutionMode = ExecutionModeSandbox
				s.logger.Info("listing task in sandbox mode, platform publish will use sandbox API",
					zap.Int64("task_id", task.ID),
				)
			case "production":
				task.ExecutionMode = ExecutionModeProduction
				s.logger.Info("listing task in production mode, platform publish will use production API",
					zap.Int64("task_id", task.ID),
				)
			}
		}
	}
	return nil
}

// writeAudit writes a structured operation log entry using LogStructured.
func (s *Service) writeAudit(action, result, resourceID, operator, content string, task *ListingTask) {
	if s.oplogSvc == nil {
		return
	}
	input := &operationlog.StructuredLogInput{
		Module:     "listing_task",
		Action:     action,
		ResourceID: resourceID,
		Operator:   operator,
		Content:    content,
		Result:     result,
		TriggerType: "system",
	}
	if task != nil {
		input.EntityType = "listing_task"
		input.EntityID = task.ID
		input.ApprovalID = task.ApprovalID
	}
	_ = s.oplogSvc.LogStructured(input)
}

// statusChangeAudit writes a structured audit entry for a listing task status transition.
func (s *Service) statusChangeAudit(taskID int64, oldStatus, newStatus, operator string) {
	if s.oplogSvc == nil {
		return
	}
	content := fmt.Sprintf("listing_task_id=%d status_change %s → %s operator=%s", taskID, oldStatus, newStatus, operator)
	_ = s.oplogSvc.LogStructured(&operationlog.StructuredLogInput{
		Module:      "listing_task",
		Action:      "listing_task.status_change",
		ResourceID:  fmt.Sprintf("%d", taskID),
		Operator:    operator,
		Content:     content,
		TriggerType: "system",
		EntityType:  "listing_task",
		EntityID:    taskID,
	})
}

// ---------- Review ----------

// ReviewTask returns a review of the listing task: whether it was published,
// any platform errors, and expected vs actual profit.
func (s *Service) ReviewTask(taskID int64) (*TaskReview, error) {
	var task ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}

	review := &TaskReview{
		TaskID:    uint(task.ID),
		Published: task.Status == "completed" && task.LastError == "",
		Status:    task.Status,
		CreatedAt: task.CreatedAt,
	}

	// Platform name
	var plat platform.Platform
	if err := s.db.First(&plat, task.PlatformID).Error; err == nil {
		review.Platform = plat.Code
	}

	// Platform errors from task items
	var items []ListingTaskItem
	s.db.Where("task_id = ? AND error_message != ''", taskID).Find(&items)
	for _, item := range items {
		if item.ErrorMessage != "" {
			review.PlatformErrors = append(review.PlatformErrors, item.ErrorMessage)
		}
	}
	if len(review.PlatformErrors) == 0 {
		review.PlatformErrors = nil
	}

	// Expected profit/margin from task target fields
	if task.TargetProfitMargin != nil {
		review.MarginExpected = task.TargetProfitMargin
	}
	if task.TargetSalePrice != nil && task.TargetProfitMargin != nil {
		pe := *task.TargetSalePrice * *task.TargetProfitMargin
		review.ProfitExpected = &pe
	}

	// Actual profit from orders (sales_order_item → sales_order)
	type orderProfit struct {
		ProfitAmount float64
		ProfitMargin float64
	}
	var orders []orderProfit
	s.db.Table("sales_order_item").
		Select("so.profit_amount, so.profit_margin").
		Joins("JOIN sales_order so ON so.id = sales_order_item.order_id").
		Where("sales_order_item.product_id = ? AND so.platform_id = ?", task.ProductID, task.PlatformID).
		Scan(&orders)
	if len(orders) > 0 {
		var totalProfit, totalMargin float64
		for _, o := range orders {
			totalProfit += o.ProfitAmount
			totalMargin += o.ProfitMargin
		}
		pa := totalProfit
		ma := totalMargin / float64(len(orders))
		review.ProfitActual = &pa
		review.MarginActual = &ma
	}

	return review, nil
}

// ExecuteTask triggers execution of a listing task after passing the execution gate.
//
// The execution gate enforces (in order):
//  1. Task exists
//  2. Idempotency: completed → return success, executing → error
//  3. Status must be "approved"
//  4. ApprovalID must be set
//  5. Approval record exists, approved, EntityType=listing_task, EntityID=task.ID
//
// After passing the gate, it runs Prism (if enabled) followed by platform publishing.
func (s *Service) ExecuteTask(taskID int64, operator string) (*ListingTask, error) {
	var task ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}

	// Idempotency: already completed — return success
	if task.Status == "completed" {
		return &task, nil
	}

	// Validation gate
	if err := s.validateExecutePreconditions(&task); err != nil {
		return nil, err
	}

	// Dry-run mode: validate and audit only, skip Prism and platform publish.
	if task.DryRun {
		s.logger.Info("listing task dry-run: skipping Prism and platform publish",
			zap.Int64("task_id", task.ID),
		)
		// Audit: dry-run execution
		s.writeAudit("listing_task.execute", "dry_run", fmt.Sprintf("%d", taskID), operator,
			fmt.Sprintf("listing_task_id=%d product_id=%d platform_id=%d dry_run=true", task.ID, task.ProductID, task.PlatformID),
			&task)

		err := s.db.Transaction(func(tx *gorm.DB) error {
			var items []ListingTaskItem
			if err := tx.Where("task_id = ?", taskID).Find(&items).Error; err != nil {
				return err
			}
			now := time.Now()
			for i := range items {
				result := map[string]interface{}{"dry_run": true, "executed_at": now}
				resultBytes, _ := json.Marshal(result)
				if err := tx.Model(&items[i]).Updates(map[string]interface{}{
					"status":      "completed",
					"executed_at": &now,
					"result":      resultBytes,
				}).Error; err != nil {
					return err
				}
			}
			return tx.Model(&task).Update("status", "completed").Error
		})
		if err != nil {
			return nil, err
		}
		if err := s.db.First(&task, taskID).Error; err != nil {
			return nil, err
		}
		s.writeAudit("listing_task.execute", "success", fmt.Sprintf("%d", taskID), operator,
			fmt.Sprintf("listing_task_id=%d product_id=%d platform_id=%d dry_run=true", task.ID, task.ProductID, task.PlatformID),
			&task)
		return &task, nil
	}


	oldStatus := task.Status

	// Run Prism check outside the main transaction so transient failures don't
	// block the task-update transaction.
	prismResult := s.runPrismForTask(&task)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&task).Update("status", "executing").Error; err != nil {
			return err
		}

		// If Prism blocked execution, update task + listing states and abort.
		if prismResult != nil && prismResult.action == "block" {
			_ = tx.Model(&task).Updates(map[string]interface{}{
				"status":     "blocked",
				"last_error": prismResult.lastError,
			}).Error
			if task.ProductListingID != nil {
				_ = tx.Model(&ProductListing{}).Where("id = ?", *task.ProductListingID).
					Update("status", "blocked").Error
			}
			return fmt.Errorf("prism compliance failed: %s", prismResult.lastError)
		}

		var items []ListingTaskItem
		if err := tx.Where("task_id = ?", taskID).Find(&items).Error; err != nil {
			return err
		}
		now := time.Now()
		finalStatus := "completed"
		for i := range items {
			result := map[string]interface{}{"executed_at": now}
			if prismResult != nil {
				result["prism"] = prismResult.data
			}
			resultBytes, _ := json.Marshal(result)
			updates := map[string]interface{}{
				"status":      "completed",
				"executed_at": &now,
				"result":      resultBytes,
			}
			if err := tx.Model(&items[i]).Updates(updates).Error; err != nil {
				return err
			}
		}

		// Persist Prism result into the ProductListing's published_data.
		if prismResult != nil && prismResult.data != nil && task.ProductListingID != nil {
			prismMeta, _ := json.Marshal(prismResult.data)
			_ = tx.Model(&ProductListing{}).Where("id = ?", *task.ProductListingID).
				Update("published_data", gorm.Expr("published_data || ?::jsonb", string(prismMeta))).Error
		}

		return tx.Model(&task).Update("status", finalStatus).Error
	})
	if err != nil {
		// If it's a Prism block the DB was updated correctly; return blocked state.
		if prismResult != nil && prismResult.action == "block" {
			if err := s.db.First(&task, taskID).Error; err != nil {
				return nil, err
			}
			// Audit: blocked by Prism
			s.writeAudit("listing_task.execute", "blocked", fmt.Sprintf("%d", taskID), operator,
				fmt.Sprintf("listing_task_id=%d prism_blocked error=%s", taskID, task.LastError),
				&task)
			s.statusChangeAudit(taskID, oldStatus, task.Status, operator)
			if s.loopRec != nil {
				_ = s.loopRec.RecordExecutionResult(task.ProductID, task.ID, false, "prism_blocked: "+task.LastError)
			}
			return &task, nil
		}
		// Audit: execution failed (non-Prism error)
		s.writeAudit("listing_task.execute", "failure", fmt.Sprintf("%d", taskID), operator,
			fmt.Sprintf("listing_task_id=%d error=%v", taskID, err),
			&task)
		if s.loopRec != nil {
			_ = s.loopRec.RecordExecutionResult(task.ProductID, taskID, false, err.Error())
		}
		return nil, err
	}
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	// Audit: execution succeeded
	s.writeAudit("listing_task.execute", "success", fmt.Sprintf("%d", taskID), operator,
		fmt.Sprintf("listing_task_id=%d product_id=%d platform_id=%d", task.ID, task.ProductID, task.PlatformID),
		&task)
	s.statusChangeAudit(taskID, oldStatus, task.Status, operator)
	if s.loopRec != nil {
		_ = s.loopRec.RecordExecutionResult(task.ProductID, task.ID, true, "")
	}

	// Platform publish hook: push product to the platform adapter.
	// Called after the DB transaction commits so a platform failure
	// doesn't roll back the completion. On error, revert to "failed"
	// so the user can retry via the existing RetryFailed/RetryItem APIs.
	if s.publishHook != nil {
		if pubErr := s.publishHook(task.ID, task.ExecutionMode); pubErr != nil {
			s.logger.Error("platform publish failed after task completed, reverting to failed",
				zap.Int64("task_id", task.ID),
				zap.Error(pubErr),
			)
			// Wrap reversion in a transaction so the two writes are atomic.
			if txErr := s.db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&ListingTask{}).Where("id = ?", task.ID).
					Updates(map[string]interface{}{
						"status":     "failed",
						"last_error": pubErr.Error(),
					}).Error; err != nil {
					return err
				}
				if err := tx.Model(&ListingTaskItem{}).Where("task_id = ?", task.ID).
					Where("error_message = '' OR error_message IS NULL").
					Update("error_message", pubErr.Error()).Error; err != nil {
					return err
				}
				if err := tx.First(&task, task.ID).Error; err != nil {
					return err
				}
				return nil
			}); txErr != nil {
				s.logger.Error("failed to revert listing task after publish failure",
					zap.Int64("task_id", task.ID),
					zap.Error(txErr),
				)
			}
		}
	}

	return &task, nil
}

// prismResult carries the action and metadata from a Prism check.
type prismResult struct {
	action    string // "pass", "warn", "block"
	data      map[string]interface{}
	lastError string
}

// runPrismForTask checks whether this task should run Prism and if so, calls it.
// Returns nil if Prism is skipped or disabled.
func (s *Service) runPrismForTask(task *ListingTask) *prismResult {
	if s.prismSvc == nil {
		return nil
	}

	// Read the ProductListing to check if Prism was enabled.
	if task.ProductListingID == nil {
		return nil
	}
	var listing ProductListing
	if err := s.db.First(&listing, *task.ProductListingID).Error; err != nil {
		return nil
	}

	var pd struct {
		Prism struct {
			Enabled bool            `json:"enabled"`
			Options json.RawMessage `json:"options"`
		} `json:"prism"`
	}
	if len(listing.PublishedData) > 0 {
		_ = json.Unmarshal(listing.PublishedData, &pd)
	}
	if !pd.Prism.Enabled {
		return nil
	}

	// Look up platform code.
	var plat platform.Platform
	if err := s.db.First(&plat, task.PlatformID).Error; err != nil {
		s.logger.Warn("prism: platform lookup failed", zap.Int64("platform_id", task.PlatformID), zap.Error(err))
		return s.prismError("platform_not_found")
	}

	// Look up product main_image.
	var prod sku.Product
	if err := s.db.First(&prod, task.ProductID).Error; err != nil {
		s.logger.Warn("prism: product lookup failed", zap.Int64("product_id", task.ProductID), zap.Error(err))
		return s.prismError("product_not_found")
	}
	if prod.MainImage == "" {
		s.logger.Warn("prism: product has no main_image", zap.Int64("product_id", task.ProductID))
		return s.prismError("no_main_image")
	}

	// Call Prism.
	resp, err := s.prismSvc.Generate(nil, &prismadapter.GenerateRequest{
		ImageURL:  prod.MainImage,
		Platform:  plat.Code,
		ProductID: task.ProductID,
	})
	if err != nil {
		s.logger.Warn("prism: generate call failed", zap.Error(err))
		return s.prismError(fmt.Sprintf("prism_service_error: %v", err))
	}

	// Build metadata for storage.
	data := map[string]interface{}{
		"prism": map[string]interface{}{
			"job_id":            resp.JobID,
			"output_url":        resp.OutputURL,
			"compliance_report": resp.ComplianceReport,
			"risk_score":        resp.RiskScore,
			"failure_reasons":   resp.FailureReasons,
		},
	}

	switch resp.ComplianceReport.Status {
	case prismadapter.StatusPass:
		return &prismResult{action: "pass", data: data}
	case prismadapter.StatusWarning:
		return &prismResult{action: "warn", data: data}
	case prismadapter.StatusFail:
		reasons := ""
		if len(resp.FailureReasons) > 0 {
			for i, r := range resp.FailureReasons {
				if i > 0 {
					reasons += "; "
				}
				reasons += r
			}
		}
		return &prismResult{
			action:    "block",
			data:      data,
			lastError: fmt.Sprintf("prism compliance failed: %s", reasons),
		}
	default:
		s.logger.Warn("prism: unknown compliance status", zap.String("status", resp.ComplianceReport.Status))
		return &prismResult{
			action:    "block",
			data:      data,
			lastError: fmt.Sprintf("prism unknown compliance status: %s", resp.ComplianceReport.Status),
		}
	}
}

// prismError returns a block/warn result depending on strict mode.
func (s *Service) prismError(errMsg string) *prismResult {
	if s.prismStrict {
		return &prismResult{action: "block", lastError: errMsg}
	}
	s.logger.Warn("prism: non-strict mode, continuing despite error", zap.String("error", errMsg))
	return nil
}

// ProductListing is a minimal projection for Prism data storage.
type ProductListing struct {
	ID            int64           `gorm:"column:id;primaryKey;autoIncrement"`
	Status        string          `gorm:"column:status"`
	PublishedData json.RawMessage `gorm:"column:published_data;type:jsonb"`
}

// TableName explicitly sets the table name.
func (ProductListing) TableName() string { return "product_listing" }
