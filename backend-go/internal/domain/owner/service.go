package owner

import (
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides Owner cockpit aggregated data.
type Service struct {
	db            *gorm.DB
	logger        *zap.Logger
	trustscoreSvc *trustscore.Service   // optional, may be nil
	oplogSvc      *operationlog.Service // optional, may be nil
}

// NewService creates a new Owner cockpit service.
// trustscoreSvc may be nil (TrustScore integration disabled).
// oplogSvc may be nil (audit logging disabled).
func NewService(db *gorm.DB, logger *zap.Logger, trustscoreSvc *trustscore.Service, oplogSvc *operationlog.Service) *Service {
	return &Service{db: db, logger: logger, trustscoreSvc: trustscoreSvc, oplogSvc: oplogSvc}
}

// RiskSummary returns aggregated risk metrics for the Owner cockpit.
func (s *Service) RiskSummary() (map[string]interface{}, error) {
	result := map[string]interface{}{
		"low_profit_products":        0,
		"missing_data_products":      0,
		"pending_approvals":          0,
		"pending_approval_count":     0,
		"blocked_listing_task_count": 0,
		"recommended_listing_count":  0,
		"sync_errors":                0,
		"total_candidates":           0,
		"total_recommendations":      0,
		"list_ready_products":        0,
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
	result["blocked_listing_task_count"] = pendingApp

	// New: count pending approval requests from approval_request table
	var pendingApprovalCount int64
	s.db.Model(&struct{}{}).Table("approval_request").
		Where("status = ?", "pending").Count(&pendingApprovalCount)
	result["pending_approval_count"] = pendingApprovalCount

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
	result["recommended_listing_count"] = listReady

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
		ID                   int64
		ProductID            int64
		CompletenessScore    float64
		ProfitMargin         float64
		EstimatedProfit      float64
		Decision             string
		Confidence           float64
		Reason               string
		RiskFlags            string
		FeedbackStatus       string
		FeedbackNote         string
		CreatedListingTaskID *int64
		CreatedAt            time.Time
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

// recordTrustScoreFeedback notifies TrustScore when feedback is recorded.
func (s *Service) recordTrustScoreFeedback(agentID string) {
	if s.trustscoreSvc == nil || agentID == "" {
		return
	}
	if err := s.trustscoreSvc.RecordAgentFeedback(agentID); err != nil {
		s.logger.Warn("failed to record trust score feedback",
			zap.String("agent_id", agentID),
			zap.Error(err))
	}
}

// RecordFeedback records Owner feedback on a listing recommendation.
// - adopt: updates listing task status to pending_approval, creates approval request
// - reject: updates listing task status to rejected, sets feedback note
func (s *Service) RecordFeedback(recommendationID int64, input *FeedbackInput) error {
	// Fetch the recommendation
	type Rec struct {
		ID                   int64
		ProductID            int64
		Decision             string
		CreatedListingTaskID *int64
		TriggeredBy          string
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

			approvalSvc := approval.NewService(s.db, s.logger, nil)

			// Use existing pending approval for this listing_task if one exists,
			// rather than creating a duplicate. The loop already created one.
			var approvalRec *approval.ApprovalRequest
			existing, err := s.HasPendingForListingTask(*rec.CreatedListingTaskID)
			if err != nil {
				return fmt.Errorf("check existing approval: %w", err)
			}
			if existing {
				existingID, _ := s.findPendingApprovalID(*rec.CreatedListingTaskID)
				if existingID > 0 {
					s.logger.Info("reusing existing pending approval for listing_task",
						zap.Int64("approval_id", existingID),
						zap.Int64("listing_task_id", *rec.CreatedListingTaskID))
					approvalRec = &approval.ApprovalRequest{ID: existingID}
				}
			}
			if approvalRec == nil {
				reason := "Owner adopted the recommendation, pending approval for listing"
				if input.Note != "" {
					reason = input.Note
				}
				approvalRec, err = approvalSvc.Create(&approval.CreateApprovalInput{
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

			// Audit: record structured audit log for adopt action
			if s.oplogSvc != nil {
				_ = s.oplogSvc.LogStructured(&operationlog.StructuredLogInput{
					Module:            "owner",
					Action:            "owner.feedback",
					ResourceID:        fmt.Sprintf("%d", recommendationID),
					Operator:          "owner",
					Content:           fmt.Sprintf("recommendation_id=%d action=adopt product_id=%d listing_task_id=%d note=%s", recommendationID, rec.ProductID, *rec.CreatedListingTaskID, input.Note),
					Result:            "adopted",
					TriggerType:       "owner_approval",
					AgentSuggestionID: &rec.ID,
					ApprovalID:        &approvalRec.ID,
					EntityType:        "listing_task",
					EntityID:          *rec.CreatedListingTaskID,
				})
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

		// Audit: record structured audit log for reject action
		if s.oplogSvc != nil {
			listingTaskID := int64(0)
			if rec.CreatedListingTaskID != nil {
				listingTaskID = *rec.CreatedListingTaskID
			}
			_ = s.oplogSvc.LogStructured(&operationlog.StructuredLogInput{
				Module:            "owner",
				Action:            "owner.feedback",
				ResourceID:        fmt.Sprintf("%d", recommendationID),
				Operator:          "owner",
				Content:           fmt.Sprintf("recommendation_id=%d action=reject product_id=%d listing_task_id=%d note=%s", recommendationID, rec.ProductID, listingTaskID, input.Note),
				Result:            "rejected",
				TriggerType:       "owner_approval",
				AgentSuggestionID: &rec.ID,
				EntityType:        "listing_task",
				EntityID:          listingTaskID,
			})
		}
	}

	// Notify TrustScore about this feedback for agent evaluation
	s.recordTrustScoreFeedback(rec.TriggeredBy)

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
				"platform_id":      r.PlatformID,
				"platform_name":    r.PlatformName,
				"mode":             mode,
				"orders_sync":      "-",
				"products_sync":    "-",
				"fees_sync":        "-",
				"settlements_sync": "-",
				"last_sync_time":   lastSyncStr,
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

// ---------------------------------------------------------------------------
// AgentActivity and PipelineChain types and methods
// ---------------------------------------------------------------------------

// AgentActivityResponse shows what agents are doing.
type AgentActivityResponse struct {
	CurrentlyRunning int64           `json:"currently_running"`
	CompletedToday   int64           `json:"completed_today"`
	FailedToday      int64           `json:"failed_today"`
	RecentEvents     []ActivityEvent `json:"recent_events"`
}

// ActivityEvent is a recent agent event for the Owner dashboard.
type ActivityEvent struct {
	ID        int64  `json:"id"`
	AgentID   string `json:"agent_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	RiskLevel string `json:"risk_level"`
	CreatedAt string `json:"created_at"`
	Summary   string `json:"summary"`
}

// PipelineChainResponse shows pipeline chain health.
type PipelineChainResponse struct {
	Chains []ChainStatus `json:"chains"`
}

// ChainStatus shows one pipeline chain.
type ChainStatus struct {
	Name          string      `json:"name"`
	Steps         []ChainStep `json:"steps"`
	OverallHealth string      `json:"overall_health"`
}

// ChainStep is one step in a pipeline chain.
type ChainStep struct {
	AgentID     string `json:"agent_id"`
	Description string `json:"description"`
	Status      string `json:"status"` // pending, running, completed, failed, blocked
}

// AgentActivity returns agent activity for the Owner.
func (s *Service) AgentActivity() (*AgentActivityResponse, error) {
	resp := &AgentActivityResponse{
		RecentEvents: []ActivityEvent{},
	}

	// 1. Count currently_running: actions currently executing.
	s.db.Table("unified_action").
		Where("status = ?", "executing").
		Count(&resp.CurrentlyRunning)

	// 2. Count completed_today: status in ('completed','executed') and created_at > today.
	todayStart := time.Now().Truncate(24 * time.Hour)
	s.db.Table("unified_action").
		Where("status IN ? AND created_at >= ?", []string{"completed", "executed"}, todayStart).
		Count(&resp.CompletedToday)

	// 3. Count failed_today: status = 'failed' and created_at > today.
	s.db.Table("unified_action").
		Where("status = ? AND created_at >= ?", "failed", todayStart).
		Count(&resp.FailedToday)

	// 4. Get latest 20 actions as recent events.
	type rawEvent struct {
		ID          int64
		AgentID     string
		Title       string
		Status      string
		RiskLevel   string
		CreatedAt   time.Time
		Description string
	}
	var events []rawEvent
	if err := s.db.Table("unified_action").
		Select("id, agent_id, title, status, COALESCE(risk_level,'medium') AS risk_level, created_at, COALESCE(description,'') AS description").
		Order("created_at DESC").
		Limit(20).
		Scan(&events).Error; err != nil {
		// Log but don't fail — return partial data.
		s.logger.Warn("agent activity recent events query failed", zap.Error(err))
	} else {
		for _, e := range events {
			summary := e.Description
			if len(summary) > 150 {
				summary = summary[:150] + "..."
			}
			resp.RecentEvents = append(resp.RecentEvents, ActivityEvent{
				ID:        e.ID,
				AgentID:   e.AgentID,
				Title:     e.Title,
				Status:    e.Status,
				RiskLevel: e.RiskLevel,
				CreatedAt: e.CreatedAt.Format("2006-01-02 15:04:05"),
				Summary:   summary,
			})
		}
	}

	return resp, nil
}

// PipelineChain returns pipeline chain health.
func (s *Service) PipelineChain() (*PipelineChainResponse, error) {
	// Define known pipeline chains.
	type chainStepDef struct {
		AgentID     string
		Description string
	}
	type chainDef struct {
		Name  string
		Steps []chainStepDef
	}

	chains := []chainDef{
		{
			Name: "Stock Alert → Discount Check → Profit Watch",
			Steps: []chainStepDef{
				{AgentID: "A5", Description: "库存预警检查"},
				{AgentID: "G3", Description: "促销折扣风险验证"},
				{AgentID: "A6", Description: "利润监控分析"},
			},
		},
		{
			Name: "Profit Watch → Listing Optimize",
			Steps: []chainStepDef{
				{AgentID: "A6", Description: "利润看护与成本优化"},
				{AgentID: "A2", Description: "Listing 优化建议"},
			},
		},
		{
			Name: "System Health → Dashboard Overview",
			Steps: []chainStepDef{
				{AgentID: "G0", Description: "系统健康扫描"},
				{AgentID: "G1", Description: "驾驶舱聚合展示"},
			},
		},
	}

	result := &PipelineChainResponse{
		Chains: make([]ChainStatus, 0, len(chains)),
	}

	for _, chain := range chains {
		cs := ChainStatus{
			Name:          chain.Name,
			Steps:         make([]ChainStep, 0, len(chain.Steps)),
			OverallHealth: "unknown",
		}

		// For each step, get the latest action status.
		for _, step := range chain.Steps {
			var latestStatus string
			s.db.Table("unified_action").
				Select("COALESCE(status, 'pending')").
				Where("agent_id = ?", step.AgentID).
				Order("created_at DESC").
				Limit(1).
				Scan(&latestStatus)

			cs.Steps = append(cs.Steps, ChainStep{
				AgentID:     step.AgentID,
				Description: step.Description,
				Status:      latestStatus,
			})
		}

		// Compute overall health: any failed = fail, all completed = ok, any blocked = blocked, else pending.
		hasFailed := false
		hasBlocked := false
		allCompleted := true
		for _, step := range cs.Steps {
			switch step.Status {
			case "failed":
				hasFailed = true
			case "blocked":
				hasBlocked = true
			case "completed":
				// completed
			default:
				allCompleted = false
			}
		}
		switch {
		case hasFailed:
			cs.OverallHealth = "failed"
		case hasBlocked:
			cs.OverallHealth = "blocked"
		case allCompleted:
			cs.OverallHealth = "ok"
		default:
			cs.OverallHealth = "pending"
		}

		result.Chains = append(result.Chains, cs)
	}

	return result, nil
}

// HasPendingForListingTask checks if there is a pending approval for the given listing_task.
func (s *Service) HasPendingForListingTask(listingTaskID int64) (bool, error) {
	var count int64
	err := s.db.Table("approval_request").
		Where("entity_type = ? AND entity_id = ? AND status = ?", "listing_task", listingTaskID, approval.StatusPending).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// findPendingApprovalID returns the ID of the pending approval for the given listing_task.
func (s *Service) findPendingApprovalID(listingTaskID int64) (int64, error) {
	var id int64
	if err := s.db.Table("approval_request").
		Select("id").
		Where("entity_type = ? AND entity_id = ? AND status = ?", "listing_task", listingTaskID, approval.StatusPending).
		Order("id DESC").
		Limit(1).
		Scan(&id).Error; err != nil {
		return 0, err
	}
	return id, nil
}
