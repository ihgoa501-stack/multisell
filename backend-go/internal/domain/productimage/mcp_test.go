package productimage

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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

func callMCP(t *testing.T, r http.Handler, body string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body)))
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return w.Code, result
}

func TestMCPRequiresJWTContextOwner(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	status, out := callMCP(t, mcpRouter(svc, nil), `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	if status != http.StatusUnauthorized {
		t.Fatalf("status=%d out=%v", status, out)
	}
	errObj := out["error"].(map[string]any)
	if errObj["message"] != "Owner authentication required" {
		t.Fatalf("out=%v", out)
	}
}

func TestMCPInitializeAndToolsListExposeStrictSchemas(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(t)
	r := mcpRouter(svc, int64(7))
	status, init := callMCP(t, r, `{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	if status != 200 || init["error"] != nil {
		t.Fatalf("status=%d out=%v", status, init)
	}
	result := init["result"].(map[string]any)
	if result["protocolVersion"] != mcpProtocolVersion {
		t.Fatalf("result=%v", result)
	}

	_, listed := callMCP(t, r, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	tools := listed["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 4 {
		t.Fatalf("tools=%v", tools)
	}
	for _, raw := range tools {
		tool := raw.(map[string]any)
		if strings.Contains(mustJSON(tool), "execution_token") {
			t.Fatalf("MCP schema exposed server-to-server execution token: %v", tool["name"])
		}
		for _, key := range []string{"inputSchema", "outputSchema"} {
			schema := tool[key].(map[string]any)
			if schema["additionalProperties"] != false {
				t.Fatalf("tool=%v schema=%v", tool["name"], schema)
			}
		}
	}
}

func TestMCPToolsUseOwnerScopeAndReturnStructuredContent(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	foreign := Task{OwnerID: 8, AssetID: 1, ImageServiceJobID: "foreign", IdempotencyKey: "foreign", ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", Width: 10, Height: 10, Format: "png", Status: "queued"}
	if err := svc.db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	r := mcpRouter(svc, int64(7))
	_, hidden := callMCP(t, r, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"lingmirror_image_get_task","arguments":{"task_id":1}}}`)
	hiddenResult := hidden["result"].(map[string]any)
	if hiddenResult["isError"] != true {
		t.Fatalf("result=%v", hiddenResult)
	}
	if strings.Contains(mustJSON(hiddenResult), "record not found") {
		t.Fatalf("internal error leaked: %v", hiddenResult)
	}

	asset, err := svc.CreateAsset(t.Context(), 7, "p.png", "image/png", []byte("image"))
	if err != nil {
		t.Fatal(err)
	}
	task, err := svc.CreateTask(t.Context(), 7, CreateTaskInput{AssetID: asset.ID, IdempotencyKey: "task", Operation: "DETERMINISTIC_RESIZE", Width: 20, Height: 20, Format: "png"})
	if err != nil {
		t.Fatal(err)
	}
	body := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"lingmirror_image_submit_task","arguments":{"task_id":` + jsonNumber(task.ID) + `,"idempotency_key":"exec"}}}`
	_, submitted := callMCP(t, r, body)
	toolResult := submitted["result"].(map[string]any)
	if toolResult["structuredContent"] == nil || len(toolResult["content"].([]any)) != 1 || remote.executeCalls != 1 {
		t.Fatalf("result=%v calls=%d", toolResult, remote.executeCalls)
	}

	attemptBody := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"lingmirror_image_list_attempts","arguments":{"task_id":` + jsonNumber(task.ID) + `}}}`
	_, attempts := callMCP(t, r, attemptBody)
	if attempts["result"].(map[string]any)["structuredContent"] == nil {
		t.Fatalf("result=%v", attempts)
	}
}

func TestMCPRejectsUnknownFieldsWithoutExecuting(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	r := mcpRouter(svc, int64(7))
	_, out := callMCP(t, r, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"lingmirror_image_submit_task","arguments":{"task_id":1,"idempotency_key":"x","owner_id":99}}}`)
	errObj := out["error"].(map[string]any)
	if errObj["code"] != float64(-32602) || remote.executeCalls != 0 {
		t.Fatalf("out=%v calls=%d", out, remote.executeCalls)
	}
}

func TestMCPExternalProviderTaskFailsClosed(t *testing.T) {
	t.Parallel()
	svc, remote := newTestService(t)
	task := Task{OwnerID: 7, AssetID: 1, ImageServiceJobID: "external-job", IdempotencyKey: "external", ManifestHash: strings.Repeat("b", 64), Operation: "OPENAI_GENERATE", Width: 10, Height: 10, Format: "png", Status: "queued"}
	if err := svc.db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	remote.jobs[task.ImageServiceJobID] = imageserviceJob(task)
	r := mcpRouter(svc, int64(7))
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"lingmirror_image_submit_task","arguments":{"task_id":` + jsonNumber(task.ID) + `,"idempotency_key":"exec"}}}`
	_, out := callMCP(t, r, body)
	result := out["result"].(map[string]any)
	if result["isError"] != true || remote.executeCalls != 0 || !strings.Contains(mustJSON(result), "PROVIDER_UNAVAILABLE") {
		t.Fatalf("result=%v calls=%d", result, remote.executeCalls)
	}
}

func mustJSON(v any) string     { b, _ := json.Marshal(v); return string(b) }
func jsonNumber(v int64) string { return string(bytes.TrimSpace([]byte(strconv.FormatInt(v, 10)))) }

func imageserviceJob(task Task) imageservice.Job {
	return imageservice.Job{ID: task.ImageServiceJobID, OwnerID: task.OwnerID, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, Status: task.Status}
}
