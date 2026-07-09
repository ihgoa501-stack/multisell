package producthub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/listing"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/loop"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"go.uber.org/zap"
)

func TestEvidenceTraceAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t,
		&candidate.CandidateProduct{},
		&completeness.CompletenessCheck{},
		&profit.ProfitSummary{},
		&loop.ListingRecommendation{},
		&approval.ApprovalRequest{},
		&listingtask.ListingTask{},
		&listingtask.ListingTaskItem{},
		&listing.ProductListing{},
		&operationlog.OperationLog{},
	)
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc, nil, nil, nil, nil, nil, nil, nil, db)

	// Seed: candidate product
	cand := &candidate.CandidateProduct{Title: "Evidence Test Product", Status: "complete"}
	if err := db.Create(cand).Error; err != nil {
		t.Fatal(err)
	}
	productID := cand.ID

	// Seed: completeness check
	if err := db.Create(&completeness.CompletenessCheck{
		ProductID: productID, Score: 85.0, Status: "complete",
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Seed: profit summary
	if err := db.Create(&profit.ProfitSummary{
		ProductID: productID, EstimatedProfit: 100.0, ProfitMargin: 15.0, Status: "profitable",
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Seed: listing recommendation
	if err := db.Create(&loop.ListingRecommendation{
		ProductID: productID, Decision: "list", Confidence: 0.9,
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Seed: listing task
	task := &listingtask.ListingTask{ProductID: productID, PlatformID: 1, Status: "pending"}
	if err := db.Create(task).Error; err != nil {
		t.Fatal(err)
	}

	// Seed: approval request
	if err := db.Create(&approval.ApprovalRequest{
		ProductID: productID, RequestType: "listing_task", Status: "approved",
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Seed: product listing
	listingRec := &listing.ProductListing{ProductID: productID, PlatformID: 1, Status: "published"}
	if err := db.Create(listingRec).Error; err != nil {
		t.Fatal(err)
	}

	// Link listing record to task
	if err := db.Model(task).Update("product_listing_id", listingRec.ID).Error; err != nil {
		t.Fatal(err)
	}

	// Seed: listing task item (execution result)
	if err := db.Create(&listingtask.ListingTaskItem{
		TaskID: task.ID, ProductID: productID, PlatformID: 1, Status: "success",
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Seed: operation log
	if err := db.Create(&operationlog.OperationLog{
		Module: "listing", Action: "publish",
		EntityType: "listing_task", EntityID: task.ID,
		Operator: "test", Result: "success",
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Set up router
	r := gin.New()
	rg := r.Group("/api/v1/product-hub")
	rg.GET("/:id/evidence", h.GetEvidence)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/product-hub/1/evidence", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int                   `json:"code"`
		Data EvidenceTraceResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Data.CandidateInfo == nil {
		t.Fatal("expected candidate_info")
	}
	if resp.Data.CandidateInfo.Title != "Evidence Test Product" {
		t.Fatalf("expected title 'Evidence Test Product', got '%s'", resp.Data.CandidateInfo.Title)
	}
	if resp.Data.Completeness == nil {
		t.Fatal("expected completeness")
	}
	if resp.Data.Completeness.Score != 85.0 {
		t.Fatalf("expected completeness score 85.0, got %f", resp.Data.Completeness.Score)
	}
	if resp.Data.ProfitSummary == nil {
		t.Fatal("expected profit_summary")
	}
	if resp.Data.ListingRecommendation == nil {
		t.Fatal("expected listing_recommendation")
	}
	if len(resp.Data.ApprovalRequests) == 0 {
		t.Fatal("expected approval_requests")
	}
	if len(resp.Data.ListingTasks) == 0 {
		t.Fatal("expected listing_tasks")
	}
	if len(resp.Data.ListingRecords) == 0 {
		t.Fatal("expected listing_records")
	}
	if len(resp.Data.ExecutionResults) == 0 {
		t.Fatal("expected execution_results")
	}
	if len(resp.Data.OperationLogSummary) == 0 {
		t.Fatal("expected operation_log_summary")
	}
	if !resp.Data.CompleteChain {
		t.Fatal("expected complete_chain to be true")
	}
}
