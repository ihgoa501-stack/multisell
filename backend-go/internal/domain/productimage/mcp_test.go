package productimage

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/imageservice"
)

func mcpRouter(service *Service, ownerID any) *gin.Engine {
	r := gin.New()
	r.POST("/mcp", func(c *gin.Context) {
		if ownerID != nil {
			c.Set("user_id", ownerID)
		}
		NewMCPHandler(service).ServeHTTP(c)
	})
	return r
}

func callMCPRaw(t *testing.T, r http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body)))
	return w
}

func callMCP(t *testing.T, r http.Handler, body string) (int, map[string]any) {
	t.Helper()
	w := callMCPRaw(t, r, body)
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return w.Code, result
}

func toolCall(name, arguments string) string {
	return `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + name + `","arguments":` + arguments + `}}`
}

func structured(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing tool result: %v", response)
	}
	value, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("missing structuredContent: %v", result)
	}
	return value
}

func TestMCPRequiresJWTContextOwner(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	status, out := callMCP(t, mcpRouter(svc, nil), `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	if status != http.StatusUnauthorized || out["error"].(map[string]any)["message"] != "Owner authentication required" {
		t.Fatalf("status=%d out=%v", status, out)
	}
}

func TestMCPInitializeNegotiatesProtocol(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	status, out := callMCP(t, mcpRouter(svc, int64(7)), `{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	if status != 200 || out["result"].(map[string]any)["protocolVersion"] != mcpProtocolVersion {
		t.Fatalf("status=%d out=%v", status, out)
	}
}

func TestMCPToolsExposeSixStrictAnnotatedContracts(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	_, listed := callMCP(t, mcpRouter(svc, int64(7)), `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	tools := listed["result"].(map[string]any)["tools"].([]any)
	want := []string{"lingmirror_image_list_capabilities", "lingmirror_image_estimate", "lingmirror_image_get_task", "lingmirror_image_list_outputs", "lingmirror_image_reconcile_task", "lingmirror_image_submit_approved_task"}
	if len(tools) != len(want) {
		t.Fatalf("tools=%v", tools)
	}
	for i, raw := range tools {
		tool := raw.(map[string]any)
		if tool["name"] != want[i] || strings.Contains(mustJSON(tool), "execution_token") {
			t.Fatalf("tool=%v", tool)
		}
		for _, key := range []string{"inputSchema", "outputSchema"} {
			if tool[key].(map[string]any)["additionalProperties"] != false {
				t.Fatalf("tool=%v schema=%v", tool["name"], tool[key])
			}
		}
		annotations := tool["annotations"].(map[string]any)
		for _, key := range []string{"readOnlyHint", "destructiveHint", "idempotentHint", "openWorldHint"} {
			if _, ok := annotations[key]; !ok {
				t.Fatalf("tool=%v missing annotation %s", tool["name"], key)
			}
		}
		if tool["name"] == "lingmirror_image_submit_approved_task" && annotations["destructiveHint"] != true {
			t.Fatalf("paid submit must be marked destructive: %v", annotations)
		}
	}
}

func TestMCPNotificationReturns204WithoutBody(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	w := callMCPRaw(t, mcpRouter(svc, int64(7)), `{"jsonrpc":"2.0","method":"tools/list","params":{}}`)
	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestMCPInvalidNotificationAlsoReturns204(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	w := callMCPRaw(t, mcpRouter(svc, int64(7)), `{"jsonrpc":"2.0","method":"missing","params":{}}`)
	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestMCPMutationNotificationIsRefusedBeforeExecution(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	task := createDeterministicMCPTask(t, svc, 7)
	body := `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"lingmirror_image_submit_approved_task","arguments":{"task_id":` + jsonNumber(task.ID) + `,"idempotency_key":"notification"}}}`
	w := callMCPRaw(t, mcpRouter(svc, int64(7)), body)
	if w.Code != http.StatusNoContent || w.Body.Len() != 0 || remote.executeCalls != 0 || len(remote.attempts[task.ImageServiceJobID]) != 0 {
		t.Fatalf("status=%d body=%q calls=%d attempts=%v", w.Code, w.Body.String(), remote.executeCalls, remote.attempts)
	}
}

func TestMCPExplicitNullIDReceivesResponse(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	w := callMCPRaw(t, mcpRouter(svc, int64(7)), `{"jsonrpc":"2.0","id":null,"method":"tools/list","params":{}}`)
	if w.Code != 200 || w.Body.Len() == 0 {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestMCPListCapabilitiesIsPaginatedAndFailClosed(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	_, out := callMCP(t, mcpRouter(svc, int64(7)), toolCall("lingmirror_image_list_capabilities", `{"page":2,"page_size":2}`))
	data := structured(t, out)
	items := data["capabilities"].([]any)
	if data["total"] != float64(4) || len(items) != 2 || items[0].(map[string]any)["availability"] != "unavailable" {
		t.Fatalf("data=%v", data)
	}
}

func TestMCPDeterministicEstimateIsExactZeroAndReadOnly(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	task := createDeterministicMCPTask(t, svc, 7)
	before := remote.executeCalls
	_, out := callMCP(t, mcpRouter(svc, int64(7)), toolCall("lingmirror_image_estimate", `{"task_id":`+jsonNumber(task.ID)+`}`))
	estimate := structured(t, out)["estimate"].(map[string]any)
	if estimate["amount"] != "0" || estimate["exact"] != true || estimate["creates_approval_or_paid_intent"] != false || remote.executeCalls != before {
		t.Fatalf("estimate=%v calls=%d", estimate, remote.executeCalls)
	}
}

func TestMCPExternalEstimateIsUnavailableAndCreatesNoApproval(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Task{}, &ExecutionApproval{}, &CostEntry{})
	remote := newFakeImageService()
	svc := NewService(db, dbtest.NewLogger(t), remote, strings.Repeat("s", 32))
	task := createExternalMCPTask(t, svc, remote, 7)
	_, out := callMCP(t, mcpRouter(svc, int64(7)), toolCall("lingmirror_image_estimate", `{"task_id":`+jsonNumber(task.ID)+`}`))
	estimate := structured(t, out)["estimate"].(map[string]any)
	var approvals int64
	_ = db.Model(&ExecutionApproval{}).Count(&approvals).Error
	if estimate["availability"] != "unavailable" || estimate["creates_approval_or_paid_intent"] != false || approvals != 0 || remote.authorizedExecuteCalls != 0 {
		t.Fatalf("estimate=%v approvals=%d calls=%d", estimate, approvals, remote.authorizedExecuteCalls)
	}
}

func TestMCPGetTaskUsesOwnerScope(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	foreign := Task{OwnerID: 8, AssetID: 1, ImageServiceJobID: "foreign", IdempotencyKey: "foreign", ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Width: 10, Height: 10, Format: "png", Status: "QUEUED"}
	if err := svc.db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	_, out := callMCP(t, mcpRouter(svc, int64(7)), toolCall("lingmirror_image_get_task", `{"task_id":`+jsonNumber(foreign.ID)+`}`))
	result := out["result"].(map[string]any)
	if result["isError"] != true || !strings.Contains(mustJSON(result), "NOT_FOUND") || strings.Contains(mustJSON(result), "record not found") {
		t.Fatalf("result=%v", result)
	}
}

func TestMCPListOutputsReturnsOnlyControlledReference(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	task := createDeterministicMCPTask(t, svc, 7)
	job := remote.jobs[task.ImageServiceJobID]
	job.Status, job.OutputBlobID = "READY", strings.Repeat("a", 64)
	remote.jobs[task.ImageServiceJobID] = job
	_, out := callMCP(t, mcpRouter(svc, int64(7)), toolCall("lingmirror_image_list_outputs", `{"task_id":`+jsonNumber(task.ID)+`,"page":1,"page_size":10}`))
	data := structured(t, out)
	item := data["outputs"].([]any)[0].(map[string]any)
	ref := item["media_reference"].(string)
	if !strings.HasPrefix(ref, "/api/v1/product-images/tasks/") || strings.Contains(ref, "://") || item["blob_sha256"] != strings.Repeat("a", 64) {
		t.Fatalf("item=%v", item)
	}
}

func TestMCPReconcileUnsupportedFailsClosedWithNextAction(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	task := createDeterministicMCPTask(t, svc, 7)
	_, out := callMCP(t, mcpRouter(svc, int64(7)), toolCall("lingmirror_image_reconcile_task", `{"task_id":`+jsonNumber(task.ID)+`}`))
	result := out["result"].(map[string]any)
	if result["isError"] != true || !strings.Contains(mustJSON(result), "RECONCILE_NOT_SUPPORTED") || !strings.Contains(mustJSON(result), "next_action") || remote.executeCalls != 0 {
		t.Fatalf("result=%v calls=%d", result, remote.executeCalls)
	}
}

func TestMCPDeterministicSubmitIsIdempotent(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	task := createDeterministicMCPTask(t, svc, 7)
	body := toolCall("lingmirror_image_submit_approved_task", `{"task_id":`+jsonNumber(task.ID)+`,"idempotency_key":"same"}`)
	_, first := callMCP(t, mcpRouter(svc, int64(7)), body)
	_, second := callMCP(t, mcpRouter(svc, int64(7)), body)
	a := structured(t, first)["attempt"].(map[string]any)
	b := structured(t, second)["attempt"].(map[string]any)
	if a["id"] != b["id"] || len(remote.attempts[task.ImageServiceJobID]) != 1 {
		t.Fatalf("first=%v second=%v attempts=%v", a, b, remote.attempts)
	}
}

func TestMCPUnconfiguredPaidSubmitFailsBeforeTokenOrRemoteCall(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Task{}, &ExecutionApproval{}, &CostEntry{})
	remote := newFakeImageService()
	svc := NewService(db, dbtest.NewLogger(t), remote, strings.Repeat("s", 32))
	task := createExternalMCPTask(t, svc, remote, 7)
	_, out := callMCP(t, mcpRouter(svc, int64(7)), toolCall("lingmirror_image_submit_approved_task", `{"task_id":`+jsonNumber(task.ID)+`,"idempotency_key":"paid"}`))
	result := out["result"].(map[string]any)
	if result["isError"] != true || !strings.Contains(mustJSON(result), "PROVIDER_UNAVAILABLE") || remote.authorizedExecuteCalls != 0 || remote.lastExecutionToken != "" {
		t.Fatalf("result=%v calls=%d", result, remote.authorizedExecuteCalls)
	}
}

func TestMCPPaidSubmitFindsApprovalAndKeepsTokenOutOfTranscript(t *testing.T) {
	db := dbtest.NewDB(t, &Task{}, &ExecutionApproval{}, &CostEntry{}, &BudgetPolicy{}, &BudgetReservation{}, &BudgetCharge{})
	remote := newFakeImageService()
	remote.available["paid-test"] = true
	svc := NewService(db, dbtest.NewLogger(t), remote, strings.Repeat("s", 32))
	task := createExternalMCPTask(t, svc, remote, 7)
	task.Operation, task.Processor, task.Region, task.ProviderEnvironment = "PAID_TEST", "paid-test", "", ""
	task.MaxCost, task.Currency = "", ""
	if err := db.Save(task).Error; err != nil {
		t.Fatal(err)
	}
	job := remote.jobs[task.ImageServiceJobID]
	job.Operation, job.Processor, job.Region, job.ProviderEnvironment = task.Operation, task.Processor, "", ""
	job.MaxCost, job.Currency = "", ""
	remote.jobs[task.ImageServiceJobID] = job
	now := time.Now().UTC()
	approval := ExecutionApproval{ExecutionID: "approved", OwnerID: 7, TaskID: task.ID, TaskVersion: task.Version, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, MaxCost: "1.00", Currency: "USD", Nonce: strings.Repeat("n", 64), ApprovedAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := db.Create(&approval).Error; err != nil {
		t.Fatal(err)
	}
	policy := BudgetPolicy{OwnerID: 7, Currency: "USD", PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), TotalAmount: "10.00", IdempotencyKey: "mcp-policy", RequestHash: strings.Repeat("d", 64)}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatal(err)
	}
	reservation := BudgetReservation{OwnerID: 7, PolicyID: policy.ID, ApprovalID: approval.ID, TaskID: task.ID, TaskVersion: task.Version, ManifestHash: task.ManifestHash, Provider: task.Processor, Currency: "USD", ReservedAmount: "1.00", State: "reserved"}
	if err := db.Create(&reservation).Error; err != nil {
		t.Fatal(err)
	}
	cost := CostEntry{OwnerID: 7, TaskID: task.ID, Kind: "estimated", Category: "provider", Provider: task.Processor, Amount: "1.00", Currency: "USD", ExchangeRate: "1", ExchangeRateSource: "test", ObservedAt: now, BillingStatus: "estimated", IdempotencyKey: "cost", RequestHash: strings.Repeat("c", 64), ExpectedTaskVersion: task.Version}
	if err := db.Create(&cost).Error; err != nil {
		t.Fatal(err)
	}
	request := toolCall("lingmirror_image_submit_approved_task", `{"task_id":`+jsonNumber(task.ID)+`,"idempotency_key":"paid"}`)
	_, out := callMCP(t, mcpRouter(svc, int64(7)), request)
	transcript := mustJSON(out)
	if remote.authorizedExecuteCalls != 1 || remote.lastExecutionToken == "" || strings.Contains(request, remote.lastExecutionToken) || strings.Contains(transcript, remote.lastExecutionToken) || strings.Contains(transcript, "execution_token") {
		t.Fatalf("calls=%d request=%s transcript=%s", remote.authorizedExecuteCalls, request, transcript)
	}
}

func TestMCPRejectsUnknownFieldsWithoutExecuting(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	_, out := callMCP(t, mcpRouter(svc, int64(7)), toolCall("lingmirror_image_submit_approved_task", `{"task_id":1,"idempotency_key":"x","owner_id":99}`))
	if out["error"].(map[string]any)["code"] != float64(-32602) || remote.executeCalls != 0 {
		t.Fatalf("out=%v calls=%d", out, remote.executeCalls)
	}
}

func createDeterministicMCPTask(t *testing.T, svc *Service, owner int64) *Task {
	t.Helper()
	asset, err := svc.CreateAsset(t.Context(), owner, "p.png", "image/png", []byte("image"))
	if err != nil {
		t.Fatal(err)
	}
	seedTaskRights(t, svc, owner, asset, "listing_main", "ozon")
	in := validTaskInput(asset, "task-"+strconv.FormatInt(owner, 10))
	in.Width, in.Height = 20, 20
	task, err := svc.CreateTask(t.Context(), owner, in)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func createExternalMCPTask(t *testing.T, svc *Service, remote *fakeImageService, owner int64) *Task {
	t.Helper()
	task := &Task{OwnerID: owner, AssetID: 1, ImageServiceJobID: "external-job", IdempotencyKey: "external", ManifestHash: strings.Repeat("b", 64), Operation: openAIOperation, Processor: openAIProcessor, Region: openAIRegion, ProviderEnvironment: openAIEnvironment, MaxCost: "0.20", Currency: "USD", Version: 1, Width: 1024, Height: 1024, Format: "png", Status: "QUEUED"}
	if err := svc.db.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	remote.jobs[task.ImageServiceJobID] = imageservice.Job{ID: task.ImageServiceJobID, OwnerID: owner, LingMirrorTaskID: strconv.FormatInt(task.ID, 10), LingMirrorTaskVersion: task.Version, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, Region: task.Region, ProviderEnvironment: task.ProviderEnvironment, MaxCost: task.MaxCost, Currency: task.Currency, Status: task.Status}
	return task
}

func mustJSON(v any) string     { b, _ := json.Marshal(v); return string(b) }
func jsonNumber(v int64) string { return string(bytes.TrimSpace([]byte(strconv.FormatInt(v, 10)))) }
