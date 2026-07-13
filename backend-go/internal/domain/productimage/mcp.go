package productimage

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"gorm.io/gorm"
)

const mcpProtocolVersion = "2025-03-26"

type MCPHandler struct{ service *Service }

func NewMCPHandler(service *Service) *MCPHandler { return &MCPHandler{service: service} }

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *mcpError `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpToolResult struct {
	Content           []mcpText `json:"content"`
	StructuredContent any       `json:"structuredContent"`
	IsError           bool      `json:"isError,omitempty"`
}

func (h *MCPHandler) ServeHTTP(c *gin.Context) {
	idClaim := common.UserIDFromCtx(c)
	if idClaim == nil || *idClaim <= 0 {
		c.JSON(http.StatusUnauthorized, mcpResponse{JSONRPC: "2.0", ID: nil, Error: &mcpError{Code: -32001, Message: "Owner authentication required"}})
		return
	}
	owner := *idClaim
	if h == nil || h.service == nil {
		h.writeError(c, nil, false, -32603, "MCP service unavailable")
		return
	}
	var req mcpRequest
	if err := decodeStrict(c.Request.Body, &req); err != nil || req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
		h.writeError(c, nil, false, -32600, "Invalid Request")
		return
	}
	notification := len(req.ID) == 0
	id, valid := parseMCPID(req.ID)
	if !valid {
		h.writeError(c, nil, notification, -32600, "Invalid Request")
		return
	}

	var result any
	var callErr *mcpError
	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string         `json:"protocolVersion"`
			Capabilities    map[string]any `json:"capabilities"`
			ClientInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"clientInfo"`
		}
		if err := decodeParams(req.Params, &params); err != nil || strings.TrimSpace(params.ProtocolVersion) == "" || strings.TrimSpace(params.ClientInfo.Name) == "" {
			callErr = &mcpError{Code: -32602, Message: "Invalid params"}
		} else {
			result = gin.H{"protocolVersion": mcpProtocolVersion, "capabilities": gin.H{"tools": gin.H{"listChanged": false}}, "serverInfo": gin.H{"name": "lingmirror-product-images", "version": "1.0.0"}}
		}
	case "tools/list":
		var params struct{}
		if err := decodeParams(req.Params, &params); err != nil {
			callErr = &mcpError{Code: -32602, Message: "Invalid params"}
		} else {
			result = gin.H{"tools": mcpTools()}
		}
	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := decodeParams(req.Params, &params); err != nil || params.Name == "" {
			callErr = &mcpError{Code: -32602, Message: "Invalid params"}
		} else if notification && isMutatingMCPTool(params.Name) {
			// A mutation without a request ID gives the caller no result to
			// reconcile. Refuse it before any approval, attempt or external state
			// can be consumed; JSON-RPC notifications still receive no body.
			c.Status(http.StatusNoContent)
			return
		} else {
			result, callErr = h.callTool(c, owner, params.Name, params.Arguments)
		}
	default:
		callErr = &mcpError{Code: -32601, Message: "Method not found"}
	}
	if callErr != nil {
		h.writeError(c, id, notification, callErr.Code, callErr.Message)
		return
	}
	if notification {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, mcpResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func isMutatingMCPTool(name string) bool {
	return name == "lingmirror_image_submit_approved_task"
}

func (h *MCPHandler) callTool(c *gin.Context, owner int64, name string, raw json.RawMessage) (any, *mcpError) {
	switch name {
	case "lingmirror_image_list_capabilities":
		var in pageInput
		if decodeParams(raw, &in) != nil || !in.valid() {
			return nil, invalidToolParams()
		}
		page, size := in.normalized()
		items, total := h.service.ListCapabilitiesContext(c.Request.Context(), page, size)
		return toolResult(pageResult("capabilities", items, page, size, total)), nil
	case "lingmirror_image_estimate":
		var in taskInput
		if decodeParams(raw, &in) != nil || in.TaskID <= 0 {
			return nil, invalidToolParams()
		}
		estimate, err := h.service.EstimateTask(c.Request.Context(), owner, in.TaskID)
		if err != nil {
			return toolFailure(err), nil
		}
		return toolResult(gin.H{"estimate": estimate}), nil
	case "lingmirror_image_get_task":
		var in taskInput
		if decodeParams(raw, &in) != nil || in.TaskID <= 0 {
			return nil, invalidToolParams()
		}
		task, err := h.service.GetTask(c.Request.Context(), owner, in.TaskID)
		if err != nil {
			return toolFailure(err), nil
		}
		return toolResult(gin.H{"task": mcpTaskFrom(task)}), nil
	case "lingmirror_image_list_outputs":
		var in taskPageInput
		if decodeParams(raw, &in) != nil || in.TaskID <= 0 || !in.pageInput.valid() {
			return nil, invalidToolParams()
		}
		page, size := in.normalized()
		items, total, err := h.service.ListOutputs(c.Request.Context(), owner, in.TaskID, page, size)
		if err != nil {
			return toolFailure(err), nil
		}
		return toolResult(pageResult("outputs", items, page, size, total)), nil
	case "lingmirror_image_reconcile_task":
		var in taskInput
		if decodeParams(raw, &in) != nil || in.TaskID <= 0 {
			return nil, invalidToolParams()
		}
		task, err := h.service.ReconcileTask(c.Request.Context(), owner, in.TaskID)
		if err != nil {
			return toolFailure(err), nil
		}
		return toolResult(gin.H{"task": mcpTaskFrom(task)}), nil
	case "lingmirror_image_submit_approved_task":
		var in struct {
			TaskID         int64  `json:"task_id"`
			IdempotencyKey string `json:"idempotency_key"`
		}
		if decodeParams(raw, &in) != nil || in.TaskID <= 0 || strings.TrimSpace(in.IdempotencyKey) == "" {
			return nil, invalidToolParams()
		}
		attempt, err := h.service.Execute(c.Request.Context(), owner, in.TaskID, in.IdempotencyKey)
		if err != nil {
			return toolFailure(err), nil
		}
		return toolResult(gin.H{"attempt": mcpAttemptFrom(attempt)}), nil
	default:
		return nil, &mcpError{Code: -32602, Message: "Unknown tool"}
	}
}

func mcpTaskFrom(task *Task) gin.H {
	return gin.H{
		"id": task.ID, "asset_id": task.AssetID, "operation": task.Operation,
		"processor": task.Processor, "version": task.Version, "width": task.Width,
		"height": task.Height, "format": task.Format, "status": task.Status,
		"error_code": task.ErrorCode, "has_output": task.OutputBlobID != "",
		"sandbox": task.Sandbox, "watermarked": task.Watermarked, "non_publishable": task.NonPublishable,
		"created_at": task.CreatedAt, "updated_at": task.UpdatedAt,
	}
}

func mcpAttemptFrom(attempt *imageservice.Attempt) gin.H {
	return gin.H{
		"id": attempt.ID, "number": attempt.Number, "status": attempt.Status,
		"error_code": attempt.ErrorCode, "created_at": attempt.CreatedAt,
		"started_at": attempt.StartedAt, "completed_at": attempt.CompletedAt,
	}
}

type pageInput struct {
	Page     int `json:"page,omitempty"`
	PageSize int `json:"page_size,omitempty"`
}

func (p pageInput) valid() bool { return p.Page >= 0 && p.PageSize >= 0 && p.PageSize <= 100 }
func (p pageInput) normalized() (int, int) {
	page, size := p.Page, p.PageSize
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	return page, size
}

type taskInput struct {
	TaskID int64 `json:"task_id"`
}
type taskPageInput struct {
	taskInput
	pageInput
}

func pageResult(key string, items any, page, size, total int) gin.H {
	return gin.H{key: items, "page": page, "page_size": size, "total": total, "has_more": page*size < total}
}

func decodeStrict(r io.Reader, dst any) error {
	d := json.NewDecoder(io.LimitReader(r, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values")
	}
	return nil
}

func decodeParams(raw json.RawMessage, dst any) error {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		raw = []byte("{}")
	}
	return decodeStrict(bytes.NewReader(raw), dst)
}

func parseMCPID(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, true
	}
	var n json.Number
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&n) == nil {
		return n, true
	}
	return nil, false
}

func invalidToolParams() *mcpError { return &mcpError{Code: -32602, Message: "Invalid tool arguments"} }

func toolResult(v any) mcpToolResult {
	b, _ := json.Marshal(v)
	return mcpToolResult{Content: []mcpText{{Type: "text", Text: string(b)}}, StructuredContent: v}
}

func toolError(code, message, nextAction string, retryable bool) mcpToolResult {
	v := gin.H{"error_code": code, "message": message, "next_action": nextAction, "retryable": retryable}
	r := toolResult(v)
	r.IsError = true
	return r
}

func toolFailure(err error) mcpToolResult {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return toolError("NOT_FOUND", "Resource not found for the authenticated Owner", "Check the LingMirror task reference; do not try another Owner's ID", false)
	}
	if errors.Is(err, ErrInvalidInput) {
		return toolError("VALIDATION_ERROR", "Request is not valid", "Correct the arguments or wait until the task has an output", false)
	}
	var conflict *ConflictError
	if errors.As(err, &conflict) {
		switch conflict.Code {
		case "APPROVAL_REQUIRED":
			return toolError(conflict.Code, "A current Owner approval is required for this paid task", "Ask the Owner to approve this exact task version in LingMirror, then retry with the same idempotency key", false)
		case "BUDGET_COST_REQUIRED":
			return toolError(conflict.Code, "The approved paid task has no valid budget cost record", "Ask the Owner to review the budget record in LingMirror", false)
		case "PROVIDER_UNAVAILABLE":
			return toolError(conflict.Code, "The task provider is not configured", "Choose an available processor or configure and verify the provider", false)
		case "RECONCILE_NOT_SUPPORTED":
			return toolError(conflict.Code, "This processor has no remote provider state to reconcile", "Read the task status; do not submit a new paid intent", false)
		case "VERSION_CONFLICT":
			return toolError(conflict.Code, "The task version no longer matches the approval", "Reload the task and request a new Owner approval for the exact version", false)
		default:
			return toolError("IDEMPOTENCY_CONFLICT", "Idempotency key conflicts with another request", "Reuse the key only with the original request or create a new explicit intent", false)
		}
	}
	return toolError("IMAGE_SERVICE_ERROR", "Image service is unavailable or returned an invalid response", "Wait and inspect the task before retrying; do not create a new paid intent", true)
}

func (h *MCPHandler) writeError(c *gin.Context, id any, notification bool, code int, message string) {
	if notification {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, mcpResponse{JSONRPC: "2.0", ID: id, Error: &mcpError{Code: code, Message: message}})
}

func objectSchema(properties gin.H, required ...string) gin.H {
	s := gin.H{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func mcpTools() []gin.H {
	taskID := gin.H{"task_id": gin.H{"type": "integer", "minimum": 1}}
	page := gin.H{"page": gin.H{"type": "integer", "minimum": 1, "default": 1}, "page_size": gin.H{"type": "integer", "minimum": 1, "maximum": 100, "default": 20}}
	taskPage := gin.H{"task_id": taskID["task_id"], "page": page["page"], "page_size": page["page_size"]}
	readonly := gin.H{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true, "openWorldHint": false}
	reconcile := gin.H{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true, "openWorldHint": true}
	submit := gin.H{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": true, "openWorldHint": true}
	pageOutput := func(key string, item gin.H) gin.H {
		return objectSchema(gin.H{key: gin.H{"type": "array", "items": item}, "page": gin.H{"type": "integer", "minimum": 1}, "page_size": gin.H{"type": "integer", "minimum": 1, "maximum": 100}, "total": gin.H{"type": "integer", "minimum": 0}, "has_more": gin.H{"type": "boolean"}}, key, "page", "page_size", "total", "has_more")
	}
	stringArray := gin.H{"type": "array", "items": gin.H{"type": "string"}}
	capabilitySchema := objectSchema(gin.H{"code": gin.H{"type": "string"}, "name": gin.H{"type": "string"}, "configured": gin.H{"type": "boolean"}, "availability": gin.H{"type": "string", "enum": []string{"available", "unavailable"}}, "operations": stringArray, "paid": gin.H{"type": "boolean"}, "reconcile_safe": gin.H{"type": "boolean"}, "reason": gin.H{"type": "string"}, "safety_level": gin.H{"type": "string"}, "provider_environment": gin.H{"type": "string"}, "watermarked": gin.H{"type": "boolean"}, "non_publishable": gin.H{"type": "boolean"}, "quota_available": gin.H{"type": "boolean"}, "quota_remaining": gin.H{"type": "integer", "minimum": 0}}, "code", "name", "configured", "availability", "operations", "paid", "reconcile_safe", "safety_level", "watermarked", "non_publishable", "quota_available")
	taskSchema := objectSchema(gin.H{"id": gin.H{"type": "integer", "minimum": 1}, "asset_id": gin.H{"type": "integer", "minimum": 1}, "operation": gin.H{"type": "string"}, "processor": gin.H{"type": "string"}, "version": gin.H{"type": "integer", "minimum": 1}, "width": gin.H{"type": "integer", "minimum": 1}, "height": gin.H{"type": "integer", "minimum": 1}, "format": gin.H{"type": "string"}, "status": gin.H{"type": "string"}, "error_code": gin.H{"type": "string"}, "has_output": gin.H{"type": "boolean"}, "sandbox": gin.H{"type": "boolean"}, "watermarked": gin.H{"type": "boolean"}, "non_publishable": gin.H{"type": "boolean"}, "created_at": gin.H{"type": "string", "format": "date-time"}, "updated_at": gin.H{"type": "string", "format": "date-time"}}, "id", "asset_id", "operation", "processor", "version", "width", "height", "format", "status", "error_code", "has_output", "sandbox", "watermarked", "non_publishable", "created_at", "updated_at")
	estimateSchema := objectSchema(gin.H{"task_id": gin.H{"type": "integer", "minimum": 1}, "processor": gin.H{"type": "string"}, "operation": gin.H{"type": "string"}, "availability": gin.H{"type": "string", "enum": []string{"available", "unavailable"}}, "amount": gin.H{"type": "string"}, "currency": gin.H{"type": "string"}, "exact": gin.H{"type": "boolean"}, "creates_approval_or_paid_intent": gin.H{"type": "boolean"}, "reason": gin.H{"type": "string"}}, "task_id", "processor", "operation", "availability", "exact", "creates_approval_or_paid_intent")
	outputSchema := objectSchema(gin.H{"ordinal": gin.H{"type": "integer", "minimum": 1}, "blob_sha256": gin.H{"type": "string", "pattern": "^[a-f0-9]{64}$"}, "media_reference": gin.H{"type": "string", "pattern": "^/api/v1/product-images/tasks/[1-9][0-9]*/output/content$"}, "status": gin.H{"type": "string"}, "sandbox": gin.H{"type": "boolean"}, "watermarked": gin.H{"type": "boolean"}, "non_publishable": gin.H{"type": "boolean"}}, "ordinal", "blob_sha256", "media_reference", "status", "sandbox", "watermarked", "non_publishable")
	attemptSchema := objectSchema(gin.H{"id": gin.H{"type": "string"}, "number": gin.H{"type": "integer", "minimum": 0}, "status": gin.H{"type": "string"}, "error_code": gin.H{"type": "string"}, "created_at": gin.H{"type": "string", "format": "date-time"}, "started_at": gin.H{"type": []string{"string", "null"}, "format": "date-time"}, "completed_at": gin.H{"type": []string{"string", "null"}, "format": "date-time"}}, "id", "number", "status", "error_code", "created_at", "started_at", "completed_at")
	return []gin.H{
		{"name": "lingmirror_image_list_capabilities", "description": "List paginated image processors and explicit availability for the authenticated Owner.", "inputSchema": objectSchema(page), "outputSchema": pageOutput("capabilities", capabilitySchema), "annotations": readonly},
		{"name": "lingmirror_image_estimate", "description": "Read a task estimate. This never creates an approval, budget reservation, paid intent, or provider call.", "inputSchema": objectSchema(taskID, "task_id"), "outputSchema": objectSchema(gin.H{"estimate": estimateSchema}, "estimate"), "annotations": readonly},
		{"name": "lingmirror_image_get_task", "description": "Get one Owner-scoped LingMirror image task.", "inputSchema": objectSchema(taskID, "task_id"), "outputSchema": objectSchema(gin.H{"task": taskSchema}, "task"), "annotations": readonly},
		{"name": "lingmirror_image_list_outputs", "description": "List paginated output metadata and LingMirror-controlled media references; never returns arbitrary URLs.", "inputSchema": objectSchema(taskPage, "task_id"), "outputSchema": pageOutput("outputs", outputSchema), "annotations": readonly},
		{"name": "lingmirror_image_reconcile_task", "description": "Reconcile existing remote provider state without creating a new paid intent. Unsupported providers fail closed.", "inputSchema": objectSchema(taskID, "task_id"), "outputSchema": objectSchema(gin.H{"task": taskSchema}, "task"), "annotations": reconcile},
		{"name": "lingmirror_image_submit_approved_task", "description": "Idempotently submit an existing task. Paid tasks require a valid backend-held Owner approval; execution tokens never enter MCP.", "inputSchema": objectSchema(gin.H{"task_id": gin.H{"type": "integer", "minimum": 1}, "idempotency_key": gin.H{"type": "string", "minLength": 1, "maxLength": 100}}, "task_id", "idempotency_key"), "outputSchema": objectSchema(gin.H{"attempt": attemptSchema}, "attempt"), "annotations": submit},
	}
}
