package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Audit returns a middleware that records operation logs for mutating requests
// (POST/PUT/PATCH/DELETE) and GET requests to sensitive paths. It runs after
// the handler completes and does not block the response — logging happens on a
// background goroutine so request latency is unaffected.
//
// Reads: request method, path, :id param, operator from JWT context (user_id
// or username), client IP, request body (truncated to 2KB, never secrets).
// Writes: a single row into operation_log per audited request.
//
// GET/HEAD/OPTIONS and /api/health are skipped unless the path matches
// sensitiveReadPaths.
func Audit(db *gorm.DB, logger *zap.Logger) gin.HandlerFunc {
	svc := operationlog.NewService(db, logger)
	return func(c *gin.Context) {
		method := c.Request.Method
		path := c.Request.URL.Path

		// Sensitive read paths that trigger audit even for GET requests.
		sensitiveReadPaths := []string{
			"/api/v1/finance",
			"/api/v1/orders",
			"/api/v1/settlement",
			"/api/v1/user",
			"/api/v1/rbac",
		}

		// Skip health checks unconditionally.
		if strings.HasSuffix(path, "/health") || strings.HasSuffix(path, "/healthz") {
			c.Next()
			return
		}

		// Determine if this request should be audited.
		isMutation := method == http.MethodPost || method == http.MethodPut ||
			method == http.MethodPatch || method == http.MethodDelete
		isSensitiveRead := method == http.MethodGet &&
			isSensitivePath(path, sensitiveReadPaths)

		if !isMutation && !isSensitiveRead {
			c.Next()
			return
		}

		// Snapshot the request body (capped) so handlers can still read it.
		var bodySnippet string
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, 2048))
			if err == nil {
				bodySnippet = string(bodyBytes)
				// Restore the body for downstream handlers.
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				bodySnippet = sanitizeBody(bodySnippet)
			}
		}

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		// Operator: prefer username claim, fall back to user_id, then "anonymous".
		operator := "anonymous"
		if v, ok := c.Get("username"); ok {
			if s, ok := v.(string); ok && s != "" {
				operator = s
			}
		}
		if operator == "anonymous" {
			if v, ok := c.Get("user_id"); ok {
				switch x := v.(type) {
				case string:
					if x != "" {
						operator = x
					}
				case int64:
					operator = "user:" + strconv.FormatInt(x, 10)
				}
			}
		}

		// Extract user_id as int64 for structured audit.
		var userID int64
		if v, ok := c.Get("user_id"); ok {
			switch x := v.(type) {
			case int64:
				userID = x
			case float64:
				userID = int64(x)
			}
		}

		status := c.Writer.Status()
		result := "success"
		if status >= 400 {
			result = "failure"
		}

		entry := &operationlog.OperationLog{
			Module:     moduleFromPath(path),
			Action:     actionFromMethod(method, c.FullPath()),
			ResourceID: resourceIDFromCtx(c),
			Content:    composeAuditContent(c, bodySnippet, status),
			Operator:   operator,
			UserID:     userID,
			Result:     result,
			IP:         c.ClientIP(),
			Duration:   int(latency.Milliseconds()),
		}

		// Fire-and-forget. Use a separate goroutine with a context timeout so a
		// slow DB never blocks the request goroutine.
		go func(e *operationlog.OperationLog) {
			_, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := svc.Create(e); err != nil {
				logger.Warn("audit log write failed", zap.Error(err), zap.String("path", path))
			}
		}(entry)
	}
}

// moduleFromPath extracts the module name from a request path.
// "/api/v1/order/123" → "order"; "/api/v1/ai/actions/5/approve" → "ai".
func moduleFromPath(path string) string {
	// Strip /api and /api/v1 prefixes.
	trimmed := strings.TrimPrefix(path, "/api/v1")
	trimmed = strings.TrimPrefix(trimmed, "/api")
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return "root"
	}
	parts := strings.SplitN(trimmed, "/", 2)
	return parts[0]
}

// actionFromMethod composes an action label from method + route template.
// e.g. POST /order → "create_order"; PUT /order/:id → "update_order";
// POST /ai/actions/:id/approve → "approve_ai_actions".
func actionFromMethod(method, routeTemplate string) string {
	verb := map[string]string{
		http.MethodPost:   "create",
		http.MethodPut:    "update",
		http.MethodPatch:  "patch",
		http.MethodDelete: "delete",
	}[method]
	if verb == "" {
		verb = strings.ToLower(method)
	}
	route := routeTemplate
	if route == "" {
		route = "unknown"
	}
	// Strip param segments to a stable resource name.
	route = strings.TrimPrefix(route, "/api/v1/")
	route = strings.TrimPrefix(route, "/api/")
	// Replace :id and :xxx with a placeholder marker for readability.
	route = strings.ReplaceAll(route, ":", "")
	// Use the last meaningful segment as the resource.
	segments := strings.Split(route, "/")
	name := segments[0]
	if len(segments) > 1 && segments[len(segments)-1] != "" {
		last := segments[len(segments)-1]
		// Heuristic: if last segment looks like an action (not a param), append it.
		if !strings.ContainsAny(last, "0123456789") {
			name = last + "_" + name
		}
	}
	return verb + "_" + name
}

// resourceIDFromCtx extracts the primary resource id from path params.
// Prefers :id, then :trace_id, then :product_id / :order_id / :sku_id.
func resourceIDFromCtx(c *gin.Context) string {
	for _, key := range []string{"id", "trace_id", "action_id", "product_id", "order_id", "sku_id", "settlement_id"} {
		if v := c.Param(key); v != "" {
			return v
		}
	}
	return ""
}

// composeAuditContent builds a compact JSON snapshot of the audit context.
func composeAuditContent(c *gin.Context, bodySnippet string, status int) string {
	content := map[string]interface{}{
		"path":   c.Request.URL.Path,
		"query":  c.Request.URL.RawQuery,
		"status": status,
	}
	if bodySnippet != "" {
		// Try to pretty-validate JSON; otherwise store raw.
		var parsed interface{}
		if err := json.Unmarshal([]byte(bodySnippet), &parsed); err == nil {
			content["body"] = parsed
		} else {
			// Truncate non-JSON bodies to 256 chars to avoid log bloat.
			if len(bodySnippet) > 256 {
				bodySnippet = bodySnippet[:256] + "..."
			}
			content["body_raw"] = bodySnippet
		}
	}
	if errMsg, exists := c.Get("error"); exists {
		content["error"] = errMsg
	}
	b, err := json.Marshal(content)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// isSensitivePath checks if the given path matches any sensitive read path prefix.
func isSensitivePath(path string, sensitivePaths []string) bool {
	for _, p := range sensitivePaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// sensitiveFields are redacted from audit logs.
var sensitiveFields = []string{"password", "secret", "token", "api_key", "api_secret", "credential", "access_key", "access_secret", "refresh_token", "jwt", "authorization"}

// sanitizeBody redacts sensitive fields from JSON content for audit logging.
func sanitizeBody(body string) string {
	if body == "" {
		return body
	}
	// Handle both JSON objects and arrays at the top level.
	var raw interface{}
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return body
	}
	redactAny(raw)
	b, _ := json.Marshal(raw)
	return string(b)
}

func redactNested(data map[string]interface{}) {
	for k, v := range data {
		redactKey(k, data)
		redactAny(v)
	}
}

func redactAny(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		redactNested(val)
	case []interface{}:
		for _, item := range val {
			if m, ok := item.(map[string]interface{}); ok {
				redactNested(m)
			}
		}
	}

}
// redactKey redacts data[k] if the key matches any sensitive field pattern.
func redactKey(k string, data map[string]interface{}) {
	lower := strings.ToLower(k)
	for _, sf := range sensitiveFields {
		if strings.Contains(lower, sf) {
			data[k] = "***REDACTED***"
			break
		}
	}
}
