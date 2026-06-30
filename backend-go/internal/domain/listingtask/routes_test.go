package listingtask

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/integrationtest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// --------------- helpers ---------------

// createTaskViaAPI creates a listing task via POST /api/v1/listing-tasks and returns its ID.
func createTaskViaAPI(t *testing.T, ts *integrationtest.TestServer, body, token string) int64 {
	t.Helper()
	resp := ts.Post(t, "/api/v1/listing-tasks", body, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create task: HTTP %d", resp.StatusCode)
	}
	var r struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if r.Code != 0 {
		t.Fatalf("create task failed: code=%d msg=%s", r.Code, r.Message)
	}
	return r.Data.ID
}

// executeTaskViaAPI posts to /api/v1/listing-task/:id/execute and returns the HTTP status
// code and the message from the response envelope.
func executeTaskViaAPI(t *testing.T, ts *integrationtest.TestServer, taskID int64, token string) (statusCode int, message string) {
	t.Helper()
	path := fmt.Sprintf("/api/v1/listing-task/%d/execute", taskID)

	resp := ts.Post(t, path, `{}`, token)
	defer resp.Body.Close()

	var r struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&r)
	return resp.StatusCode, r.Message
}

// createItemViaAPI creates a listing task item via POST /api/v1/listing-tasks/:id/items.
// taskID is used both as the URL param and as the body's task_id to pass binding validation.
func createItemViaAPI(t *testing.T, ts *integrationtest.TestServer, taskID int64, productID, platformID int64, token string) {
	t.Helper()
	path := fmt.Sprintf("/api/v1/listing-tasks/%d/items", taskID)
	body := fmt.Sprintf(`{"task_id":%d,"product_id":%d,"platform_id":%d}`, taskID, productID, platformID)
	resp := ts.Post(t, path, body, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Read body for debug
		var r struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&r)
		t.Fatalf("create item: HTTP %d: code=%d msg=%s", resp.StatusCode, r.Code, r.Message)
	}
	var r struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&r)
	if r.Code != 0 {
		t.Fatalf("create item failed: code=%d msg=%s", r.Code, r.Message)
	}
}

// newTestServer creates a test server with listingtask routes registered.
func newTestServer(t *testing.T) *integrationtest.TestServer {
	t.Helper()
	return integrationtest.NewTestServer(t,
		func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
			RegisterRoutes(rg, db, logger, nil, false, false)
		},
		&ListingTask{}, &ListingTaskItem{}, &approval.ApprovalRequest{},
	)
}

// --------------- tests ---------------

// TestListingTaskRoutes_Unauthenticated verifies that all execution-chain endpoints
// return 401 when called without a JWT token.
func TestListingTaskRoutes_Unauthenticated(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	defer ts.Close()

	// Execute without token → 401
	resp := ts.Post(t, "/api/v1/listing-task/999/execute", `{}`, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// TestListingTaskRoutes_NoApproval verifies that ExecuteTask returns an error
// when no approval request exists for the task's product.
func TestListingTaskRoutes_NoApproval(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	defer ts.Close()

	token := ts.Login(t)

	// Create a pending task (default status is "blocked", so we specify "pending").
	taskID := createTaskViaAPI(t, ts, `{"product_id":1,"platform_id":10,"status":"pending"}`, token)

	// Execute without any approval → should fail.
	status, msg := executeTaskViaAPI(t, ts, taskID, token)
	if status != http.StatusInternalServerError {
		t.Fatalf("expected 500 for no approval, got %d", status)
	}
	if msg == "" {
		t.Fatal("expected error message, got empty")
	}
}

// TestListingTaskRoutes_BlockedStatus verifies that ExecuteTask rejects a task
// whose status is "blocked" (blocked cannot transition to executing).
func TestListingTaskRoutes_BlockedStatus(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	defer ts.Close()

	token := ts.Login(t)

	// Create a task without specifying status — defaults to "blocked".
	taskID := createTaskViaAPI(t, ts, `{"product_id":1,"platform_id":10}`, token)

	// Execute from blocked status → should fail.
	status, msg := executeTaskViaAPI(t, ts, taskID, token)
	if status != http.StatusInternalServerError {
		t.Fatalf("expected 500 for blocked status, got %d", status)
	}
	if msg == "" {
		t.Fatal("expected error message, got empty")
	}
}

// TestListingTaskRoutes_ApprovalThenExecute verifies the full happy path:
// create task → create items → create approved approval → execute → success.
func TestListingTaskRoutes_ApprovalThenExecute(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	defer ts.Close()

	token := ts.Login(t)

	// 1. Create a pending task.
	taskID := createTaskViaAPI(t, ts, `{"product_id":1,"platform_id":10,"status":"pending"}`, token)

	// 2. Create an item for the task so execution has something to process.
	createItemViaAPI(t, ts, taskID, 1, 10, token)

	// 3. Create an approved approval directly in the DB.
	now := time.Now()
	future := now.Add(24 * time.Hour)
	ts.GetDB().Create(&approval.ApprovalRequest{
		ProductID:   1,
		RequestType: "publish",
		Requester:   "test",
		Status:      approval.StatusApproved,
		ExpiresAt:   &future,
	})

	// 4. Execute — should succeed.
	status, msg := executeTaskViaAPI(t, ts, taskID, token)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d with message: %s", status, msg)
	}
}

// TestListingTaskRoutes_Idempotency verifies that a completed task cannot be
// executed a second time.
func TestListingTaskRoutes_Idempotency(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t)
	defer ts.Close()

	token := ts.Login(t)

	// 1. Create a pending task.
	taskID := createTaskViaAPI(t, ts, `{"product_id":1,"platform_id":10,"status":"pending"}`, token)

	// 2. Create items.
	createItemViaAPI(t, ts, taskID, 1, 10, token)
	createItemViaAPI(t, ts, taskID, 2, 10, token)

	// 3. Create approved approval.
	now := time.Now()
	future := now.Add(24 * time.Hour)
	ts.GetDB().Create(&approval.ApprovalRequest{
		ProductID:   1,
		RequestType: "publish",
		Requester:   "test",
		Status:      approval.StatusApproved,
		ExpiresAt:   &future,
	})

	// 4. First execution should succeed.
	status, msg := executeTaskViaAPI(t, ts, taskID, token)
	if status != http.StatusOK {
		t.Fatalf("first execute: expected 200, got %d: %s", status, msg)
	}

	// 5. Second execution should fail with idempotency guard.
	status, msg = executeTaskViaAPI(t, ts, taskID, token)
	if status != http.StatusInternalServerError {
		t.Fatalf("second execute: expected 500, got %d", status)
	}
	if msg == "" {
		t.Fatal("expected error message for idempotency, got empty")
	}
}
