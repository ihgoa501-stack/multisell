package integrationtest

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/loop"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/owner"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TestTrustedClosedLoop_HappyPath tests the full trusted closed loop via HTTP:
//
//  1. Register + login (get JWT)
//  2. POST /v1/candidates — create candidate product
//  3. POST /v1/loop/evaluate/:id — evaluate (complete → profit → recommend → listingtask)
//  4. POST /v1/owner/suggestions/:id/feedback — adopt suggestion
//  5. PUT /v1/approval/:id/review — approve
//  6. PUT /v1/listing-tasks/:id — set approved status + approval_id
//  7. POST /v1/listing-task/:id/execute — execute listing
//  8. Verify state transitions and audit records
func TestTrustedClosedLoop_HappyPath(t *testing.T) {
	ts := NewTestServer(t, registerClosedLoopRoutes,
		&candidate.CandidateProduct{},
		&completeness.CompletenessCheck{},
		&profit.ProfitSummary{},
		&listingtask.ListingTask{},
		&listingtask.ListingTaskItem{},
		&loop.ListingRecommendation{},
		&approval.ApprovalRequest{},
		&operationlog.OperationLog{},
	)
	defer ts.Close()

	token := ts.Login(t)

	// ── Step 1: Create candidate product ──
	price := 50.0
	targetPrice := 100.0
	weight := 0.5
	length := 10.0
	width := 10.0
	height := 10.0
	catID := int64(1)
	brandID := int64(1)

	candidateBody := map[string]interface{}{
		"title":               "集成测试商品",
		"description":         "这是一款用于可信经营闭环集成测试的商品",
		"main_image":          "http://example.com/test.jpg",
		"images":              []string{"http://example.com/test1.jpg"},
		"category_id":         catID,
		"brand_id":            brandID,
		"spec_json":           map[string]string{"color": "red", "size": "M"},
		"purchase_price":      price,
		"purchase_currency":   "CNY",
		"package_weight_kg":   weight,
		"package_length_cm":   length,
		"package_width_cm":    width,
		"package_height_cm":   height,
		"hs_code":             "847130",
		"origin_country":      "CN",
		"target_sale_price":   targetPrice,
		"target_currency":     "USD",
		"destination_country": "US",
	}
	resp := ts.Post(t, "/api/v1/candidates", candidateBody, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create candidate: status %d (expected 200)", resp.StatusCode)
	}
	var createResult response.Result
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResult.Code != 0 {
		t.Fatalf("create candidate code=%d msg=%s", createResult.Code, createResult.Message)
	}
	// Parse candidate from data
	createData, _ := json.Marshal(createResult.Data)
	var createdCandidate candidate.CandidateProduct
	json.Unmarshal(createData, &createdCandidate)
	productID := createdCandidate.ID
	if productID == 0 {
		t.Fatal("created candidate ID is 0")
	}

	// ── Step 2: Evaluate candidate ──
	resp = ts.Post(t, "/api/v1/loop/evaluate/"+jsonEncodeID(productID), map[string]interface{}{
		"triggered_by": "tester",
	}, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("evaluate: status %d (expected 200)", resp.StatusCode)
	}
	var evalResult response.Result
	if err := json.NewDecoder(resp.Body).Decode(&evalResult); err != nil {
		t.Fatalf("decode evaluate response: %v", err)
	}
	if evalResult.Code != 0 {
		t.Fatalf("evaluate code=%d msg=%s", evalResult.Code, evalResult.Message)
	}
	// Parse evaluate result to get listing_task_id
	evalData, _ := json.Marshal(evalResult.Data)
	var evalDataMap map[string]interface{}
	json.Unmarshal(evalData, &evalDataMap)

	decision, _ := evalDataMap["decision"].(string)
	if decision != "list" {
		t.Fatalf("expected decision 'list', got '%s' - check product data for completeness/profit thresholds", decision)
	}

	listingTaskIDFloat, ok := evalDataMap["listing_task_id"].(float64)
	if !ok || listingTaskIDFloat == 0 {
		t.Fatal("expected listing_task_id in evaluate result")
	}
	listingTaskID := int64(listingTaskIDFloat)

	// ── Step 3: Get suggestions and adopt ──
	resp = ts.Get(t, "/api/v1/owner/suggestions?limit=10", token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("suggestions: status %d (expected 200)", resp.StatusCode)
	}
	var suggestionsResult response.Result
	if err := json.NewDecoder(resp.Body).Decode(&suggestionsResult); err != nil {
		t.Fatalf("decode suggestions response: %v", err)
	}
	suggData, _ := json.Marshal(suggestionsResult.Data)
	var suggestions []map[string]interface{}
	json.Unmarshal(suggData, &suggestions)

	if len(suggestions) == 0 {
		t.Fatal("expected at least 1 suggestion")
	}
	suggestionID := int64(suggestions[0]["id"].(float64))

	// Adopt the suggestion
	resp = ts.Post(t, "/api/v1/owner/suggestions/"+jsonEncodeID(suggestionID)+"/feedback", map[string]interface{}{
		"action": "adopt",
		"note":   "owner approved this listing",
	}, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("adopt feedback: status %d (expected 200)", resp.StatusCode)
	}

	// Verify listing task status is now "pending_approval"
	var task listingtask.ListingTask
	if err := ts.DB.First(&task, listingTaskID).Error; err != nil {
		t.Fatalf("find listing task: %v", err)
	}
	if task.Status != "pending_approval" {
		t.Fatalf("listing task status = %s (expected pending_approval)", task.Status)
	}

	// ── Step 4: Approve via approval API ──
	// Find the approval request created by RecordFeedback
	var apprReqs []approval.ApprovalRequest
	ts.DB.Where("entity_type = ? AND entity_id = ?", "listing_task", listingTaskID).Find(&apprReqs)
	if len(apprReqs) == 0 {
		t.Fatal("no approval request found for listing task")
	}
	approvalID := apprReqs[0].ID
	if apprReqs[0].Status != "pending" {
		t.Fatalf("approval status = %s (expected pending)", apprReqs[0].Status)
	}

	resp = ts.Put(t, "/api/v1/approval/"+jsonEncodeID(approvalID)+"/review", map[string]interface{}{
		"action":   "approve",
		"reviewer": "owner",
	}, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: status %d (expected 200)", resp.StatusCode)
	}

	// Set listing task status to "approved" and link the approval_id
	resp = ts.Put(t, "/api/v1/listing-tasks/"+jsonEncodeID(listingTaskID), map[string]interface{}{
		"status":      "approved",
		"approval_id": approvalID,
	}, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update listing task: status %d (expected 200)", resp.StatusCode)
	}

	// ── Step 5: Execute listing task ──
	resp = ts.Post(t, "/api/v1/listing-task/"+jsonEncodeID(listingTaskID)+"/execute", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("execute: status %d (expected 200)", resp.StatusCode)
	}
	var execResult response.Result
	if err := json.NewDecoder(resp.Body).Decode(&execResult); err != nil {
		t.Fatalf("decode execute response: %v", err)
	}
	if execResult.Code != 0 {
		t.Fatalf("execute code=%d msg=%s", execResult.Code, execResult.Message)
	}

	// ── Verify final state ──
	// Listing task should be "completed"
	var updatedTask listingtask.ListingTask
	if err := ts.DB.First(&updatedTask, listingTaskID).Error; err != nil {
		t.Fatalf("find updated listing task: %v", err)
	}
	if updatedTask.Status != "completed" {
		t.Fatalf("listing task status = %s (expected completed)", updatedTask.Status)
	}

	// Recommendation feedback_status should be "executed"
	var rec loop.ListingRecommendation
	ts.DB.Where("product_id = ?", productID).First(&rec)
	if rec.FeedbackStatus != "executed" {
		t.Fatalf("recommendation feedback_status = %s (expected executed)", rec.FeedbackStatus)
	}

	// Audit records should exist for listing_task execution
	var auditCount int64
	ts.DB.Model(&operationlog.OperationLog{}).
		Where("module = ? AND action = ?", "listing_task", "listing_task.execute").
		Count(&auditCount)
	if auditCount == 0 {
		t.Fatal("expected at least one audit record for listing_task.execute")
	}
}

// TestTrustedClosedLoop_Unauthenticated tests that closed loop API endpoints return 401 without auth.
func TestTrustedClosedLoop_Unauthenticated(t *testing.T) {
	ts := NewTestServer(t, registerClosedLoopRoutes,
		&candidate.CandidateProduct{},
		&completeness.CompletenessCheck{},
		&profit.ProfitSummary{},
		&listingtask.ListingTask{},
		&loop.ListingRecommendation{},
		&approval.ApprovalRequest{},
	)
	defer ts.Close()

	paths := []struct {
		method string
		path   string
		body   interface{}
	}{
		{http.MethodGet, "/api/v1/candidates", nil},
		{http.MethodPost, "/api/v1/candidates", map[string]interface{}{"title": "test"}},
		{http.MethodPost, "/api/v1/loop/evaluate/1", nil},
		{http.MethodGet, "/api/v1/owner/suggestions", nil},
		{http.MethodPost, "/api/v1/owner/suggestions/1/feedback", map[string]interface{}{"action": "adopt"}},
		{http.MethodGet, "/api/v1/approval", nil},
		{http.MethodPut, "/api/v1/approval/1/review", map[string]interface{}{"action": "approve"}},
		{http.MethodGet, "/api/v1/listing-tasks", nil},
		{http.MethodPost, "/api/v1/listing-task/1/execute", nil},
	}

	for _, p := range paths {
		t.Run(p.method+" "+p.path, func(t *testing.T) {
			var resp *http.Response
			switch p.method {
			case http.MethodGet:
				resp = ts.Get(t, p.path, "")
			case http.MethodPost:
				resp = ts.Post(t, p.path, p.body, "")
			case http.MethodPut:
				resp = ts.Put(t, p.path, p.body, "")
			default:
				t.Fatalf("unsupported method: %s", p.method)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d for %s %s", resp.StatusCode, p.method, p.path)
			}
		})
	}
}

// TestTrustedClosedLoop_BlockedTaskCannotExecute tests that a blocked listing task
// cannot be executed even with an approval.
func TestTrustedClosedLoop_BlockedTaskCannotExecute(t *testing.T) {
	ts := NewTestServer(t, registerClosedLoopRoutes,
		&listingtask.ListingTask{},
		&listingtask.ListingTaskItem{},
		&approval.ApprovalRequest{},
	)
	defer ts.Close()

	token := ts.Login(t)

	// Create a listing task with blocked status
	resp := ts.Post(t, "/api/v1/listing-tasks", map[string]interface{}{
		"product_id":  1,
		"platform_id": 10,
	}, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create listing task: status %d", resp.StatusCode)
	}
	var createResult response.Result
	json.NewDecoder(resp.Body).Decode(&createResult)
	createData, _ := json.Marshal(createResult.Data)
	var taskMap map[string]interface{}
	json.Unmarshal(createData, &taskMap)
	taskID := int64(taskMap["id"].(float64))

	// Try to execute — should fail (blocked status)
	resp = ts.Post(t, "/api/v1/listing-task/"+jsonEncodeID(taskID)+"/execute", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("expected error executing blocked task, got 200")
	}
}

// TestTrustedClosedLoop_NoApprovalRejected tests that a task without an approval
// cannot be executed even if status is "approved".
func TestTrustedClosedLoop_NoApprovalRejected(t *testing.T) {
	ts := NewTestServer(t, registerClosedLoopRoutes,
		&listingtask.ListingTask{},
	)
	defer ts.Close()

	token := ts.Login(t)

	// Create a listing task
	resp := ts.Post(t, "/api/v1/listing-tasks", map[string]interface{}{
		"product_id":  1,
		"platform_id": 10,
	}, token)
	defer resp.Body.Close()
	var createResult response.Result
	json.NewDecoder(resp.Body).Decode(&createResult)
	createData, _ := json.Marshal(createResult.Data)
	var taskMap map[string]interface{}
	json.Unmarshal(createData, &taskMap)
	taskID := int64(taskMap["id"].(float64))

	// Update status to "approved" but no approval_id
	resp = ts.Put(t, "/api/v1/listing-tasks/"+jsonEncodeID(taskID), map[string]interface{}{
		"status": "approved",
	}, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update listing task: status %d", resp.StatusCode)
	}

	// Try to execute — should fail (no approval_id)
	resp = ts.Post(t, "/api/v1/listing-task/"+jsonEncodeID(taskID)+"/execute", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("expected error executing task without approval, got 200")
	}
}

// registerClosedLoopRoutes registers all routes needed for closed loop integration tests.
// It initializes all services with the correct dependencies.
func registerClosedLoopRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	// Create services with proper dependency injection
	opsvc := operationlog.NewService(db, logger)
	apprSvc := approval.NewService(db, logger, opsvc)
	loopSvc := loop.NewService(db, logger, nil, false)

	// Register listing task routes with full dependency chain
	listingtask.RegisterRoutes(rg, db, logger, nil, false, apprSvc, opsvc, nil, loopSvc)

	// Register loop, approval, owner, candidate routes
	approval.RegisterRoutes(rg, db, logger, opsvc)
	owner.RegisterRoutes(rg, db, logger)
	candidate.RegisterRoutes(rg, db, logger)
	loop.RegisterRoutes(rg, db, logger, nil, false)
}

// jsonEncodeID encodes an int64 as a JSON number string for path construction.
func jsonEncodeID(id int64) string {
	b, _ := json.Marshal(id)
	return string(b)
}
