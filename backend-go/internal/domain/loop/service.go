package loop

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/exchangerate"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides the evaluation loop business logic.
type Service struct {
	db              *gorm.DB
	logger          *zap.Logger
	completenessSvc *completeness.Service
	profitSvc       *profit.Service
	listingtaskSvc  *listingtask.Service
	approvalSvc     *approval.Service
	oplogSvc        *operationlog.Service
}

// NewService creates a new evaluation loop service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	oplogSvc := operationlog.NewService(db, logger)
	approvalSvc := approval.NewService(db, logger, oplogSvc)
	rateSvc := exchangerate.NewService(db, logger)
	return &Service{
		db:              db,
		logger:          logger,
		completenessSvc: completeness.NewService(db, logger),
		profitSvc:       profit.NewService(db, logger, rateSvc, 7.2),
		listingtaskSvc:  listingtask.NewService(db, logger, approvalSvc, oplogSvc, nil),
		approvalSvc:     approvalSvc,
		oplogSvc:        oplogSvc,
	}
}

// Evaluate runs the full evaluation pipeline for a single product:
// 1. Completeness check
// 2. Profit calculation
// 3. Listing recommendation (agent-like decision)
// 4. If recommended → create listingtask (blocked, pending approval)
// 5. Record decision
func (s *Service) Evaluate(productID int64, triggeredBy string) (*EvaluateResult, error) {
	if triggeredBy == "" {
		triggeredBy = "system"
	}

	// 1. Check candidate product exists
	var prod candidate.CandidateProduct
	if err := s.db.First(&prod, productID).Error; err != nil {
		return nil, err
	}

	// 2. Run completeness check
	compResult, err := s.completenessSvc.Check(productID, triggeredBy)
	if err != nil {
		return nil, fmt.Errorf("completeness check failed: %w", err)
	}

	// 3. Run profit calculation
	profitResult, err := s.profitSvc.Calculate(productID, triggeredBy)
	if err != nil {
		return nil, fmt.Errorf("profit calculation failed: %w", err)
	}

	// 4. Generate recommendation (agent-like decision rules)
	decision, confidence, reason, riskFlags := s.generateRecommendation(compResult, profitResult)

	// 5. If decision is "list", create a blocked listingtask + pending approval in one transaction
	var listingTaskID *int64
	var approvalID *int64
	if decision == "list" {
		err := s.db.Transaction(func(tx *gorm.DB) error {
			task, err := s.createListingTask(tx, &prod, profitResult, compResult, triggeredBy)
			if err != nil {
				return err
			}
			listingTaskID = &task.ID

			as := approval.NewService(tx, s.logger, s.oplogSvc)
			req, err := as.Create(&approval.CreateApprovalInput{
				ProductID:   prod.ID,
				RequestType: "publish",
				Requester:   triggeredBy,
				TargetType:  "listing_task",
				TargetID:    task.ID,
				EntityType:  "listing_task",
				EntityID:    task.ID,
				RiskLevel:   "high",
				NewValue:    string(task.DecisionSnapshot),
				Reason:      reason,
			})
			if err != nil {
				return err
			}
			approvalID = &req.ID

			// Link the listing task to its approval, required by validateExecutePreconditions
			if err := tx.Model(&listingtask.ListingTask{}).Where("id = ?", task.ID).
				Update("approval_id", req.ID).Error; err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("creating listing approval: %w", err)
		}

		// Write audit log for the approval-gated transition
		entityID := int64(0)
		if listingTaskID != nil {
			entityID = *listingTaskID
		}
		if err := s.oplogSvc.LogStructured(&operationlog.StructuredLogInput{
			Module:      "loop",
			Action:      "evaluate_list",
			ResourceID:  fmt.Sprintf("candidate:%d", productID),
			Operator:    triggeredBy,
			Content:     reason,
			Result:      "pending_approval",
			TriggerType: "agent",
			EntityType:  "listing_task",
			EntityID:    entityID,
			ApprovalID:  approvalID,
		}); err != nil {
			s.logger.Error("failed to write evaluate_list audit",
				zap.Int64("candidate_id", productID),
				zap.Error(err),
			)
		}
	}

	// 6. Store the recommendation record
	riskJSON, _ := json.Marshal(riskFlags)
	rec := ListingRecommendation{
		ProductID:            productID,
		CompletenessScore:    compResult.Score,
		ProfitMargin:         profitResult.ProfitMargin,
		EstimatedProfit:      profitResult.EstimatedProfit,
		Decision:             decision,
		Confidence:           confidence,
		Reason:               reason,
		RiskFlags:            string(riskJSON),
		CreatedListingTaskID: listingTaskID,
		ApprovalID:           approvalID,
		TriggeredBy:          triggeredBy,
		FeedbackStatus:       "pending",
	}
	if err := s.db.Create(&rec).Error; err != nil {
		s.logger.Error("failed to save listing recommendation",
			zap.Int64("product_id", productID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("save listing recommendation: %w", err)
	}

	return &EvaluateResult{
		ProductID:          productID,
		Title:              prod.Title,
		CompletenessScore:  compResult.Score,
		CompletenessStatus: compResult.Status,
		MissingItems:       compResult.MissingItems,
		ProfitMargin:       profitResult.ProfitMargin,
		EstimatedProfit:    profitResult.EstimatedProfit,
		ProfitStatus:       profitResult.Status,
		Decision:           decision,
		Confidence:         confidence,
		Reason:             reason,
		RiskFlags:          riskFlags,
		ListingTaskID:      listingTaskID,
		ApprovalID:         approvalID,
	}, nil
}

// generateRecommendation implements the agent-like decision logic.
func (s *Service) generateRecommendation(comp *completeness.CheckResult, profit *profit.ProfitResult) (decision string, confidence float64, reason string, riskFlags []string) {
	var risks []string
	points := 0.0
	maxPoints := 100.0

	// Completeness factor (40% weight)
	if comp.Score >= 80 {
		points += 40
	} else if comp.Score >= 50 {
		points += 20
		risks = append(risks, "资料不完整")
	} else {
		points += 5
		risks = append(risks, "资料严重缺失")
	}

	// Profit margin factor (40% weight)
	if profit.ProfitMargin >= 15 {
		points += 40
	} else if profit.ProfitMargin >= 5 {
		points += 20
		risks = append(risks, "利润偏低")
	} else if profit.ProfitMargin >= 0 {
		points += 10
		risks = append(risks, "利润极低")
	} else {
		points += 0
		risks = append(risks, "负利润")
	}

	// Revenue factor (20% weight)
	if profit.TargetRevenue >= 100 {
		points += 20
	} else if profit.TargetRevenue >= 50 {
		points += 10
		risks = append(risks, "售价偏低")
	}

	// Normalize confidence
	confidence = math.Round(points/maxPoints*100) / 100
	if confidence > 1 {
		confidence = 1
	}

	// Decision logic
	if comp.Score < 50 {
		decision = "skip"
		reason = "资料完整度过低（评分 <50），不建议上架。请先补充：" + strings.Join(comp.MissingItems, "、")
		return
	}

	if profit.ProfitMargin < 0 {
		decision = "skip"
		reason = fmt.Sprintf("利润为负（%s: %.2f%%），不建议上架。成本（$%.2f）高于目标售价（$%.2f）。",
			profit.Status, profit.ProfitMargin, profit.TotalCost, profit.TargetRevenue)
		return
	}

	if profit.ProfitMargin < 5 && comp.Score < 80 {
		decision = "cautious"
		reason = fmt.Sprintf("利润较低（%.2f%%），且资料完整度不足（%.0f%%）。建议补充资料并优化成本后再上架。",
			profit.ProfitMargin, comp.Score)
		return
	}

	if profit.ProfitMargin >= 15 && comp.Score >= 80 {
		decision = "list"
		reason = fmt.Sprintf("资料完整（%.0f%%），利润良好（%.2f%%），建议上架。", comp.Score, profit.ProfitMargin)
		return
	}

	// Default: cautious
	decision = "cautious"
	reason = fmt.Sprintf("条件适中：完整度 %.0f%%，利润率 %.2f%%。建议在补充资料或确认成本后决定。",
		comp.Score, profit.ProfitMargin)
	return
}

// createListingTask creates a blocked listing task requiring approval.
func (s *Service) createListingTask(db *gorm.DB, prod *candidate.CandidateProduct, profitResult *profit.ProfitResult, compResult *completeness.CheckResult, triggeredBy string) (*listingtask.ListingTask, error) {
	// Build decision snapshot
	ds := map[string]interface{}{
		"completeness_score": compResult.Score,
		"profit_margin":      profitResult.ProfitMargin,
		"estimated_profit":   profitResult.EstimatedProfit,
		"total_cost":         profitResult.TotalCost,
		"status":             compResult.Status,
		"evaluated_by":       triggeredBy,
	}
	dsJSON, _ := json.Marshal(ds)

	missingJSON, _ := json.Marshal(compResult.MissingItems)

	platformID := int64(1) // default platform (1 = Ozon mock)
	if prod.TargetPlatformID != nil && *prod.TargetPlatformID > 0 {
		platformID = *prod.TargetPlatformID
	}

	targetMargin := profitResult.ProfitMargin
	targetPrice := prod.TargetSalePrice

	task, err := listingtask.NewService(db, s.logger, s.approvalSvc, s.oplogSvc, nil).Create(&listingtask.CreateTaskInput{
		ProductID:           prod.ID,
		PlatformID:          platformID,
		SourceType:          "campaign",
		SourceItemKey:       fmt.Sprintf("candidate:%d", prod.ID),
		Status:              "blocked",
		MissingRequirements: missingJSON,
		DecisionSnapshot:    dsJSON,
		TargetSalePrice:     &targetPrice,
		TargetProfitMargin:  &targetMargin,
		DestinationCountry:  prod.DestinationCountry,
		CreatedBy:           triggeredBy,
	})
	return task, err
}

// GetRecommendations returns paginated listing recommendations.
func (s *Service) GetRecommendations(page, size int, decision string) ([]ListingRecommendation, int64, error) {
	var items []ListingRecommendation
	var total int64
	q := s.db.Model(&ListingRecommendation{})
	if decision != "" {
		q = q.Where("decision = ?", decision)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// RecordExecutionResult updates the feedback status based on execution outcome.
// Called by listingtask service after ExecuteTask completes.
func (s *Service) RecordExecutionResult(productID int64, listingTaskID int64, success bool, errorMsg string) error {
	updates := map[string]interface{}{
		"feedback_status": "executed",
	}
	if !success {
		updates["feedback_status"] = "execution_failed"
		updates["feedback_note"] = errorMsg
	}

	res := s.db.Model(&ListingRecommendation{}).
		Where("product_id = ? AND created_listing_task_id = ?", productID, listingTaskID).
		Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("record execution result: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		s.logger.Warn("RecordExecutionResult: no recommendation found",
			zap.Int64("product_id", productID),
			zap.Int64("listing_task_id", listingTaskID))
	}
	return nil
}
