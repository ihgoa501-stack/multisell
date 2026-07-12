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
	"gorm.io/gorm"
)

const mcpProtocolVersion = "2025-03-26"

type MCPHandler struct{ service *Service }

func NewMCPHandler(service *Service) *MCPHandler { return &MCPHandler{service: service} }

// RegisterMCP attaches the MCP endpoint to an already JWT-protected router group.
// The endpoint intentionally lives on LingMirror Backend; Image Service stays private.
func RegisterMCP(rg *gin.RouterGroup, service *Service) {
	rg.POST("/product-images/mcp", NewMCPHandler(service).ServeHTTP)
}

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
		h.writeError(c, nil, -32603, "MCP service unavailable")
		return
	}
	var req mcpRequest
	if err := decodeStrict(c.Request.Body, &req); err != nil || req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
		h.writeError(c, nil, -32600, "Invalid Request")
		return
	}
	id, valid := parseMCPID(req.ID)
	if !valid {
		h.writeError(c, nil, -32600, "Invalid Request")
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
		} else {
			result, callErr = h.callTool(c, owner, params.Name, params.Arguments)
		}
	default:
		callErr = &mcpError{Code: -32601, Message: "Method not found"}
	}
	if callErr != nil {
		h.writeError(c, id, callErr.Code, callErr.Message)
		return
	}
	c.JSON(http.StatusOK, mcpResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (h *MCPHandler) callTool(c *gin.Context, owner int64, name string, raw json.RawMessage) (any, *mcpError) {
	switch name {
	case "lingmirror_image_list_capabilities":
		var in struct{}
		if decodeParams(raw, &in) != nil {
			return nil, invalidToolParams()
		}
		configured := h.service.image != nil
		return toolResult(gin.H{"capabilities": []gin.H{
			{"code": "deterministic", "name": "凌镜标准处理", "configured": configured, "operations": []string{"DETERMINISTIC_RESIZE"}},
			{"code": "photoroom", "name": "Photoroom", "configured": false, "operations": []string{}},
			{"code": "adobe", "name": "Adobe Firefly", "configured": false, "operations": []string{}},
			{"code": "openai", "name": "OpenAI Images", "configured": false, "operations": []string{}},
		}}), nil
	case "lingmirror_image_get_task":
		var in struct {
			TaskID int64 `json:"task_id"`
		}
		if decodeParams(raw, &in) != nil || in.TaskID <= 0 {
			return nil, invalidToolParams()
		}
		task, err := h.service.GetTask(c.Request.Context(), owner, in.TaskID)
		if err != nil {
			return toolFailure(err), nil
		}
		return toolResult(gin.H{"task": task}), nil
	case "lingmirror_image_list_attempts":
		var in struct {
			TaskID int64 `json:"task_id"`
		}
		if decodeParams(raw, &in) != nil || in.TaskID <= 0 {
			return nil, invalidToolParams()
		}
		items, err := h.service.Attempts(c.Request.Context(), owner, in.TaskID)
		if err != nil {
			return toolFailure(err), nil
		}
		return toolResult(gin.H{"attempts": items}), nil
	case "lingmirror_image_submit_task":
		var in struct {
			TaskID         int64  `json:"task_id"`
			IdempotencyKey string `json:"idempotency_key"`
		}
		if decodeParams(raw, &in) != nil || in.TaskID <= 0 || strings.TrimSpace(in.IdempotencyKey) == "" {
			return nil, invalidToolParams()
		}
		task, err := h.service.GetTask(c.Request.Context(), owner, in.TaskID)
		if err != nil {
			return toolFailure(err), nil
		}
		if task.Operation != "DETERMINISTIC_RESIZE" {
			return toolError("PROVIDER_UNAVAILABLE", "External image providers are not available"), nil
		}
		if h.service.image == nil {
			return toolError("IMAGE_SERVICE_UNAVAILABLE", "Image service unavailable"), nil
		}
		attempt, err := h.service.Execute(c.Request.Context(), owner, in.TaskID, in.IdempotencyKey)
		if err != nil {
			return toolFailure(err), nil
		}
		return toolResult(gin.H{"attempt": attempt}), nil
	default:
		return nil, &mcpError{Code: -32602, Message: "Unknown tool"}
	}
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

func toolError(code, message string) mcpToolResult {
	v := gin.H{"error_code": code, "message": message}
	r := toolResult(v)
	r.IsError = true
	return r
}

func toolFailure(err error) mcpToolResult {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return toolError("NOT_FOUND", "Resource not found")
	}
	if errors.Is(err, ErrInvalidInput) {
		return toolError("VALIDATION_ERROR", "Request is not valid")
	}
	var conflict *ConflictError
	if errors.As(err, &conflict) {
		return toolError("IDEMPOTENCY_CONFLICT", "Idempotency key conflicts with another request")
	}
	return toolError("IMAGE_SERVICE_ERROR", "Image service unavailable")
}

func (h *MCPHandler) writeError(c *gin.Context, id any, code int, message string) {
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
	return []gin.H{
		{"name": "lingmirror_image_list_capabilities", "description": "List image processing capabilities available to the authenticated Owner.", "inputSchema": objectSchema(gin.H{}), "outputSchema": objectSchema(gin.H{"capabilities": gin.H{"type": "array", "items": gin.H{"type": "object"}}}, "capabilities")},
		{"name": "lingmirror_image_get_task", "description": "Get one Owner-scoped image task.", "inputSchema": objectSchema(taskID, "task_id"), "outputSchema": objectSchema(gin.H{"task": gin.H{"type": "object"}}, "task")},
		{"name": "lingmirror_image_list_attempts", "description": "List execution attempts for one Owner-scoped image task.", "inputSchema": objectSchema(taskID, "task_id"), "outputSchema": objectSchema(gin.H{"attempts": gin.H{"type": "array", "items": gin.H{"type": "object"}}}, "attempts")},
		{"name": "lingmirror_image_submit_task", "description": "Submit an existing deterministic image task for execution. External providers are unavailable.", "inputSchema": objectSchema(gin.H{"task_id": gin.H{"type": "integer", "minimum": 1}, "idempotency_key": gin.H{"type": "string", "minLength": 1, "maxLength": 100}}, "task_id", "idempotency_key"), "outputSchema": objectSchema(gin.H{"attempt": gin.H{"type": "object"}}, "attempt")},
	}
}
