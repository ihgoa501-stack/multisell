package listingtask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/platform"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"github.com/lingmirror/backend-go/internal/prismadapter"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides listing task business logic.
type Service struct {
	db         *gorm.DB
	logger     *zap.Logger
	prismSvc   prismadapter.PrismService // nil = Prism disabled at runtime
	prismStrict bool                      // block on Prism error vs warn+continue
	trustScoreSvc *trustscore.Service
	auditSvc   *operationlog.Service
	sandbox    bool
}

// NewService creates a new listingtask service.
// prismSvc may be nil, in which case Prism is never called.
func NewService(db *gorm.DB, logger *zap.Logger, prismSvc prismadapter.PrismService, prismStrict bool, auditSvc *operationlog.Service, trustScoreSvc *trustscore.Service, sandbox bool) *Service {
	return &Service{db: db, logger: logger, prismSvc: prismSvc, prismStrict: prismStrict, auditSvc: auditSvc, sandbox: sandbox}
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

// Update patches a listing task by id.
func (s *Service) Update(id int64, in *UpdateTaskInput) (*ListingTask, error) {
	var t ListingTask
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	oldStatus := t.Status
	updates := map[string]interface{}{}
	if in.Status != nil {
		sm := NewListingTaskStateMachine()
		if err := sm.MustTransition(context.Background(), t.Status, *in.Status, &t); err != nil {
			return nil, err
		}
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
	// Audit status transition if status changed
	if s.auditSvc != nil && in.Status != nil && oldStatus != *in.Status {
		operator := "system"
		if in.UpdatedBy != nil {
			operator = *in.UpdatedBy
		}
		s.auditSvc.Log("listingtask", "status_update", fmt.Sprintf("%d", id), operator,
			fmt.Sprintf("Status changed from %s to %s", oldStatus, *in.Status))
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
	if res.RowsAffected > 0 && s.auditSvc != nil {
		s.auditSvc.Log("listingtask", "retry_all_failed", "all", "system",
			fmt.Sprintf("Reset %d failed tasks to pending", res.RowsAffected))
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
	if s.auditSvc != nil {
		s.auditSvc.Log("listingtask", "retry_failed", fmt.Sprintf("%d", taskID), "system",
			fmt.Sprintf("Reset failed task %d to pending", taskID))
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
	if s.auditSvc != nil {
		s.auditSvc.Log("listingtask", "retry_item", fmt.Sprintf("%d", itemID), "system",
			fmt.Sprintf("Reset item %d of task %d to pending", itemID, taskID))
	}
	return &item, nil
}

// ---------- Listing publish chain ----------

// ExecuteTask triggers execution of a listing task. Before platform publishing it
// optionally calls Prism for image generation + compliance check, branching by result:
//   - pass:      use Prism output image, proceed with platform publish
//   - warning:   proceed but record risks
//   - fail:      block the task and listing, set last_error
//   - error:     block if strict mode, else warn+continue with original image
func (s *Service) ExecuteTask(taskID int64) (*ListingTask, error) {
	var task ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}

	// Sandbox mode check
	if s.sandbox {
		if s.auditSvc != nil {
			s.auditSvc.Log("listingtask", "sandbox_block", fmt.Sprintf("%d", taskID), "system",
				fmt.Sprintf("Sandbox mode: task %d execution blocked", taskID))
		}
		return nil, fmt.Errorf("sandbox mode: execution blocked for task %d", taskID)
	}

	sm := NewListingTaskStateMachine()
	if !sm.CanTransition(task.Status, "executing") {
		if task.Status == "completed" {
			return nil, fmt.Errorf("task %d is already completed (idempotency guard)", taskID)
		}
		if task.Status == "cancelled" {
			return nil, fmt.Errorf("task %d is cancelled, cannot execute", taskID)
		}
		return nil, fmt.Errorf("task %d cannot execute from status %s", taskID, task.Status)
	}

	// Sandbox gate: block execution in sandbox mode
	if s.sandbox {
		if s.auditSvc != nil {
			s.auditSvc.Log("listingtask", "sandbox_block", fmt.Sprintf("%d", taskID), "system",
				fmt.Sprintf("Sandbox mode: task %d execution blocked", taskID))
		}
		return nil, fmt.Errorf("sandbox mode: execution blocked for task %d", taskID)
	}

	// Approval gate: must have an approved approval
	if err := s.checkApproval(&task); err != nil {
		return nil, err
	}

	// Audit: write pre-execution entry
	if s.auditSvc != nil {
		s.auditSvc.Log("listingtask", "execute", fmt.Sprintf("%d", taskID), "system",
			fmt.Sprintf("Executing listing task %d for product %d on platform %d", taskID, task.ProductID, task.PlatformID))
	}

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
			return &task, nil
		}
		return nil, err
	}
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// checkApproval verifies there's an approved approval for this task's product.
func (s *Service) checkApproval(task *ListingTask) error {
	var count int64
	s.db.Model(&approval.ApprovalRequest{}).
		Where("product_id = ? AND request_type = 'publish' AND status = '" + approval.StatusApproved + "' AND (expires_at IS NULL OR expires_at > ?)",
			task.ProductID, time.Now()).
		Count(&count)
	if count == 0 {
		// Check if there's a pending approval (informational)
		var pending int64
		s.db.Model(&approval.ApprovalRequest{}).
			Where("product_id = ? AND request_type = 'publish' AND status = '" + approval.StatusPending + "'", task.ProductID).
			Count(&pending)
		if pending > 0 {
			return fmt.Errorf("task %d requires approval: approval for product %d is still pending", task.ID, task.ProductID)
		}
		return fmt.Errorf("task %d requires approval: no approved approval found for product %d", task.ID, task.ProductID)
	}
	return nil
}

// SubmitFeedback records whether an agent recommendation was accepted or rejected.
// This feeds into TrustScore evaluation for agent assessment.
func (s *Service) SubmitFeedback(taskID int64, status, note, updatedBy string) (*ListingTask, error) {
	if status != "accepted" && status != "rejected" {
		return nil, fmt.Errorf("invalid feedback status: %s (must be accepted or rejected)", status)
	}
	var task ListingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&task).Updates(map[string]interface{}{
		"agent_feedback_status": status,
		"agent_feedback_note":   note,
		"updated_by":            updatedBy,
	}).Error; err != nil {
		return nil, err
	}
	task.AgentFeedbackStatus = &status
	task.AgentFeedbackNote = note
	task.UpdatedBy = updatedBy

	// Build structured audit content with all fields needed for TrustScore processing.
	accepted := status == "accepted"
	agentID := task.SourceType
	execResult := task.Status

	// Write structured audit log
	if s.auditSvc != nil {
		auditContent := fmt.Sprintf(`{"agent":"%s","feedback":"%s","execution":"%s","task_id":%d,"product_id":%d,"note":"%s"}`,
			agentID, status, execResult, taskID, task.ProductID, note)
		s.auditSvc.Log("listingtask", "feedback_"+status, fmt.Sprintf("%d", taskID), updatedBy, auditContent)
	}

	// Notify TrustScore for agent evaluation.
	if s.trustScoreSvc != nil {
		if err := s.trustScoreSvc.RecordRecommendationFeedback(agentID, accepted, taskID, task.ProductID, updatedBy, note); err != nil {
			s.logger.Warn("Failed to record TrustScore feedback", zap.Int64("task_id", taskID), zap.Error(err))
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
