package producthub

import (
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/listing"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/loop"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/profit"
)

// EvidenceTraceResponse is the full evidence chain for a product.
type EvidenceTraceResponse struct {
	ProductID             int64                           `json:"product_id"`
	CandidateInfo         *candidate.CandidateProduct     `json:"candidate_info,omitempty"`
	Completeness          *completeness.CompletenessCheck `json:"completeness,omitempty"`
	ProfitSummary         *profit.ProfitSummary           `json:"profit_summary,omitempty"`
	ListingRecommendation *loop.ListingRecommendation     `json:"listing_recommendation,omitempty"`
	ApprovalRequests      []approval.ApprovalRequest      `json:"approval_requests,omitempty"`
	ListingTasks          []listingtask.ListingTask       `json:"listing_tasks,omitempty"`
	ListingRecords        []listing.ProductListing        `json:"listing_records,omitempty"`
	ExecutionResults      []listingtask.ListingTaskItem   `json:"execution_results,omitempty"`
	OperationLogSummary   []operationlog.OperationLog     `json:"operation_log_summary,omitempty"`
	CompleteChain         bool                            `json:"complete_chain"`
}

// GetEvidenceTrace aggregates the full lifecycle of a product from candidate
// through completeness, profit, recommendation, approval, and execution.
func (s *Service) GetEvidenceTrace(productID int64) (*EvidenceTraceResponse, error) {
	resp := &EvidenceTraceResponse{ProductID: productID}

	// 1. Candidate product info
	var cand candidate.CandidateProduct
	if err := s.db.Where("id = ?", productID).First(&cand).Error; err == nil {
		resp.CandidateInfo = &cand
	}

	// 2. Latest completeness check
	var comp completeness.CompletenessCheck
	if err := s.db.Where("product_id = ?", productID).Order("created_at DESC").First(&comp).Error; err == nil {
		resp.Completeness = &comp
	}

	// 3. Latest profit summary
	var prof profit.ProfitSummary
	if err := s.db.Where("product_id = ?", productID).Order("created_at DESC").First(&prof).Error; err == nil {
		resp.ProfitSummary = &prof
	}

	// 4. Latest listing recommendation
	var rec loop.ListingRecommendation
	if err := s.db.Where("product_id = ?", productID).Order("created_at DESC").First(&rec).Error; err == nil {
		resp.ListingRecommendation = &rec
	}

	// 5. All listing tasks for this product
	var tasks []listingtask.ListingTask
	s.db.Where("product_id = ?", productID).Order("created_at DESC").Find(&tasks)
	resp.ListingTasks = tasks

	// 6. Approval requests for this product
	var approvals []approval.ApprovalRequest
	s.db.Where("product_id = ?", productID).Order("created_at DESC").Find(&approvals)
	resp.ApprovalRequests = approvals

	// 7. Listing records (product_listing entries linked via listing_task.product_listing_id)
	if listingIDs := collectNonNilIDs(tasks, func(t listingtask.ListingTask) *int64 { return t.ProductListingID }); len(listingIDs) > 0 {
		var records []listing.ProductListing
		s.db.Where("id IN ?", listingIDs).Find(&records)
		resp.ListingRecords = records
	}

	// 8. Execution results (listing_task_item records for these tasks)
	taskIDs := collectIDs(tasks)
	if len(taskIDs) > 0 {
		var items []listingtask.ListingTaskItem
		s.db.Where("task_id IN ?", taskIDs).Order("id ASC").Find(&items)
		resp.ExecutionResults = items
	}

	// 9. Operation log summary — latest 20 entries for this product's lifecycle
	var logs []operationlog.OperationLog
	logQuery := s.db.Where("entity_type = 'product' AND entity_id = ?", productID)
	if len(taskIDs) > 0 {
		logQuery = logQuery.Or("entity_type = 'listing_task' AND entity_id IN ?", taskIDs)
	}
	logQuery.Order("created_at DESC").Limit(20).Find(&logs)
	resp.OperationLogSummary = logs

	// 10. Chain completeness: only true with approved approval + completed task + results
	hasApproved := false
	for _, ar := range resp.ApprovalRequests {
		if ar.Status == "approved" {
			hasApproved = true
			break
		}
	}
	hasCompletedTask := false
	for _, t := range resp.ListingTasks {
		if t.Status == "completed" {
			hasCompletedTask = true
			break
		}
	}
	resp.CompleteChain = resp.ListingRecommendation != nil &&
		hasApproved &&
		hasCompletedTask &&
		len(resp.ExecutionResults) > 0

	return resp, nil
}

func collectIDs(tasks []listingtask.ListingTask) []int64 {
	ids := make([]int64, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	return ids
}

func collectNonNilIDs(tasks []listingtask.ListingTask, extract func(listingtask.ListingTask) *int64) []int64 {
	var ids []int64
	for _, t := range tasks {
		if v := extract(t); v != nil {
			ids = append(ids, *v)
		}
	}
	return ids
}
