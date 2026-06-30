package owner

import (
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides Owner cockpit aggregated data.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new Owner cockpit service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// RiskSummary returns aggregated risk metrics for the Owner cockpit.
func (s *Service) RiskSummary() (map[string]interface{}, error) {
	result := map[string]interface{}{
		"low_profit_products":   0,
		"missing_data_products": 0,
		"pending_approvals":     0,
		"sync_errors":           0,
		"total_candidates":      0,
		"total_recommendations": 0,
		"list_ready_products":   0,
	}

	var totalCandidates int64
	s.db.Raw("SELECT COUNT(*) FROM candidate_product").Scan(&totalCandidates)
	result["total_candidates"] = totalCandidates

	var missingCount int64
	s.db.Model(&struct{}{}).Table("completeness_check").
		Where("status = ?", "incomplete").
		Distinct("product_id").Count(&missingCount)
	result["missing_data_products"] = missingCount

	var lowProfit int64
	s.db.Raw(`
		SELECT COUNT(DISTINCT product_id) FROM profit_summary
		WHERE status IN ('marginal', 'unprofitable')
	`).Scan(&lowProfit)
	result["low_profit_products"] = lowProfit

	var pendingApp int64
	s.db.Model(&struct{}{}).Table("listing_task").
		Where("status = ?", "blocked").Count(&pendingApp)
	result["pending_approvals"] = pendingApp

	var syncErr int64
	s.db.Model(&struct{}{}).Table("mock_sync_status").
		Where("status = ?", "failed").Count(&syncErr)
	result["sync_errors"] = syncErr

	var totalRec int64
	s.db.Model(&struct{}{}).Table("listing_recommendation").Count(&totalRec)
	result["total_recommendations"] = totalRec

	var listReady int64
	s.db.Raw(`
		SELECT COUNT(DISTINCT product_id) FROM listing_recommendation
		WHERE decision = 'list'
	`).Scan(&listReady)
	result["list_ready_products"] = listReady

	return result, nil
}

// SuggestionResponse is the typed response for a single suggestion.
type SuggestionResponse struct {
	ID                int64   `json:"id"`
	ProductID         int64   `json:"product_id"`
	ProductTitle      string  `json:"product_title"`
	CompletenessScore float64 `json:"completeness_score"`
	ProfitMargin      float64 `json:"profit_margin"`
	EstimatedProfit   float64 `json:"estimated_profit"`
	Decision          string  `json:"decision"`
	Confidence        float64 `json:"confidence"`
	Reason            string  `json:"reason"`
	RiskFlags         string  `json:"risk_flags"`
	RiskLevel         string  `json:"risk_level"`

	// Agent feedback status
	FeedbackStatus string `json:"feedback_status"`
	FeedbackNote   string `json:"feedback_note"`

	// Approval status
	ListingTaskID  *int64 `json:"listing_task_id"`
	TaskStatus     string `json:"task_status"`
	ApprovalID     *int64 `json:"approval_id"`
	ApprovalStatus string `json:"approval_status"`

	CreatedAt string `json:"created_at"`
}

// Suggestions returns the latest listing recommendations as Owner-facing suggestions.
func (s *Service) Suggestions(limit int) ([]SuggestionResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	type Rec struct {
		ID                  int64
		ProductID           int64
		CompletenessScore   float64
		ProfitMargin        float64
		EstimatedProfit     float64
		Decision            string
		Confidence          float64
		Reason              string
		RiskFlags           string
		FeedbackStatus      string
		FeedbackNote        string
		CreatedListingTaskID *int64
		CreatedAt           time.Time
	}
	var recs []Rec
	if err := s.db.Table("listing_recommendation").
		Order("id DESC").Limit(limit).Find(&recs).Error; err != nil {
		return nil, err
	}

	// Batch load product titles
	productIDs := make([]int64, 0, len(recs))
	taskIDs := make([]int64, 0, len(recs))
	for _, r := range recs {
		productIDs = append(productIDs, r.ProductID)
		if r.CreatedListingTaskID != nil {
			taskIDs = append(taskIDs, *r.CreatedListingTaskID)
		}
	}
	type ProdInfo struct {
		ID    int64
		Title string
	}
	var prodInfos []ProdInfo
	if len(productIDs) > 0 {
		s.db.Table("candidate_product").Select("id, title").Where("id IN ?", productIDs).Scan(&prodInfos)
	}
	titleMap := make(map[int64]string, len(prodInfos))
	for _, p := range prodInfos {
		titleMap[p.ID] = p.Title
	}

	// Batch load listing task statuses
	taskStatusMap := make(map[int64]string)
	if len(taskIDs) > 0 {
		type TaskInfo struct {
			ID     int64
			Status string
		}
		var tasks []TaskInfo
		s.db.Table("listing_task").Select("id, status").Where("id IN ?", taskIDs).Scan(&tasks)
		for _, t := range tasks {
			taskStatusMap[t.ID] = t.Status
		}
	}

	// Batch load approval requests for listing tasks
	type ApprovalInfo struct {
		ID       int64
		EntityID int64
		Status   string
	}
	approvalMap := make(map[int64]*ApprovalInfo)
	if len(taskIDs) > 0 {
		var apps []ApprovalInfo
		s.db.Table("approval_request").
			Select("id, entity_id, status").
			Where("entity_type = ? AND entity_id IN ?", "listing_task", taskIDs).
			Scan(&apps)
		for _, a := range apps {
			approvalMap[a.EntityID] = &ApprovalInfo{ID: a.ID, EntityID: a.EntityID, Status: a.Status}
		}
	}

	var result []SuggestionResponse
	for _, r := range recs {
		title := titleMap[r.ProductID]

		riskLevel := "low"
		if r.Decision == "skip" {
			riskLevel = "high"
		} else if r.Confidence < 0.6 {
			riskLevel = "medium"
		}
		taskStatus := ""
		if r.CreatedListingTaskID != nil {
			taskStatus = taskStatusMap[*r.CreatedListingTaskID]
		}


		var approvalID *int64
		approvalStatus := ""
		if r.CreatedListingTaskID != nil {
			if a, ok := approvalMap[*r.CreatedListingTaskID]; ok {
				approvalID = &a.ID
				approvalStatus = a.Status
			}
		}

		result = append(result, SuggestionResponse{
			ID:                r.ID,
			ProductID:         r.ProductID,
			ProductTitle:      title,
			CompletenessScore: r.CompletenessScore,
			ProfitMargin:      r.ProfitMargin,
			EstimatedProfit:   r.EstimatedProfit,
			Decision:          r.Decision,
			Confidence:        r.Confidence,
			Reason:            r.Reason,
			RiskFlags:         r.RiskFlags,
			RiskLevel:         riskLevel,
			FeedbackStatus:    r.FeedbackStatus,
			FeedbackNote:      r.FeedbackNote,
			ListingTaskID:     r.CreatedListingTaskID,
			TaskStatus:        taskStatus,
			ApprovalID:        approvalID,
			ApprovalStatus:    approvalStatus,
			CreatedAt:         r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

// FeedbackInput is the request body for recording Owner feedback on a suggestion.
type FeedbackInput struct {
	Action string `json:"action" binding:"required"` // adopt, reject
	Note   string `json:"note"`
}

// RecordFeedback records Owner feedback on a listing recommendation.
// - adopt: updates listing task status to pending_approval, creates approval request
// - reject: updates listing task status to rejected, sets feedback note
func (s *Service) RecordFeedback(recommendationID int64, input *FeedbackInput) error {
	// Fetch the recommendation
	type Rec struct {
		ID                  int64
		ProductID           int64
		Decision            string
		CreatedListingTaskID *int64
	}
	var rec Rec
	if err := s.db.Table("listing_recommendation").First(&rec, recommendationID).Error; err != nil {
		return fmt.Errorf("recommendation not found: %w", err)
	}

	// Validate action
	if input.Action != "adopt" && input.Action != "reject" {
		return fmt.Errorf("invalid action: %s (must be adopt or reject)", input.Action)
	}

	switch input.Action {
	case "adopt":
		// Update recommendation feedback_status
		if err := s.db.Table("listing_recommendation").
			Where("id = ?", recommendationID).
			Updates(map[string]interface{}{
				"feedback_status": "adopted",
				"feedback_note":   input.Note,
			}).Error; err != nil {
			return fmt.Errorf("update recommendation feedback: %w", err)
		}

		// Update listing task status to pending_approval
		if rec.CreatedListingTaskID != nil && *rec.CreatedListingTaskID > 0 {
			// Check if listing task exists
			var task listingtask.ListingTask
			if err := s.db.First(&task, *rec.CreatedListingTaskID).Error; err != nil {
				s.logger.Warn("listing task not found for recommendation",
					zap.Int64("recommendation_id", recommendationID),
					zap.Int64("listing_task_id", *rec.CreatedListingTaskID))
			} else {
				if err := s.db.Model(&task).Update("status", "pending_approval").Error; err != nil {
					return fmt.Errorf("update listing task status: %w", err)
				}
			}

			// Create approval request for the listing task
			approvalSvc := approval.NewService(s.db, s.logger, nil)
			reason := "Owner adopted the recommendation, pending approval for listing"
			if input.Note != "" {
				reason = input.Note
			}
			_, err := approvalSvc.Create(&approval.CreateApprovalInput{
				ProductID:   rec.ProductID,
				RequestType: "listing_task",
				Requester:   "owner",
				Reason:      reason,
				EntityType:  "listing_task",
				EntityID:    *rec.CreatedListingTaskID,
			})
			if err != nil {
				return fmt.Errorf("create approval request: %w", err)
			}
		}

	case "reject":
		// Update recommendation feedback_status and note
		updates := map[string]interface{}{
			"feedback_status": "rejected",
		}
		if input.Note != "" {
			updates["feedback_note"] = input.Note
		}
		if err := s.db.Table("listing_recommendation").
			Where("id = ?", recommendationID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("update recommendation feedback: %w", err)
		}

		// Update listing task status to rejected
		if rec.CreatedListingTaskID != nil && *rec.CreatedListingTaskID > 0 {
			var task listingtask.ListingTask
			if err := s.db.First(&task, *rec.CreatedListingTaskID).Error; err != nil {
				s.logger.Warn("listing task not found for recommendation",
					zap.Int64("recommendation_id", recommendationID),
					zap.Int64("listing_task_id", *rec.CreatedListingTaskID))
			} else {
				if err := s.db.Model(&task).Update("status", "rejected").Error; err != nil {
					return fmt.Errorf("update listing task status: %w", err)
				}
			}
		}
	}

	return nil
}

// PlatformSyncStatus returns sync status for Owner cockpit.
func (s *Service) PlatformSyncStatus() ([]map[string]interface{}, error) {
	type SyncRow struct {
		PlatformID    int64
		PlatformName  string
		SyncType      string
		Status        string
		RecordsSynced int
		ErrorMessage  string
		LastSyncAt    *time.Time
		IsMockData    bool
	}
	var rows []SyncRow
	s.db.Table("mock_sync_status").
		Where("id IN (SELECT DISTINCT ON (platform_id, sync_type) id FROM mock_sync_status ORDER BY platform_id, sync_type, id DESC)").
		Order("platform_id, sync_type").Find(&rows)

	platforms := make(map[int64]map[string]interface{})
	for _, r := range rows {
		if _, ok := platforms[r.PlatformID]; !ok {
			mode := "mock"
			if !r.IsMockData {
				mode = "sandbox"
			}
			lastSyncStr := ""
			if r.LastSyncAt != nil {
				lastSyncStr = r.LastSyncAt.Format("2006-01-02 15:04:05")
			}
			platforms[r.PlatformID] = map[string]interface{}{
				"platform_id":    r.PlatformID,
				"platform_name":  r.PlatformName,
				"mode":           mode,
				"orders_sync":    "-",
				"products_sync":  "-",
				"fees_sync":      "-",
				"settlements_sync": "-",
				"last_sync_time":  lastSyncStr,
			}
		}
		entry := platforms[r.PlatformID]
		switch r.SyncType {
		case "orders":
			entry["orders_sync"] = r.Status
		case "products":
			entry["products_sync"] = r.Status
		case "fees":
			entry["fees_sync"] = r.Status
		}
	}

	var result []map[string]interface{}
	for _, v := range platforms {
		result = append(result, v)
	}
	return result, nil
}
