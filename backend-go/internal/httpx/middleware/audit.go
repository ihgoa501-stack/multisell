package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Audit returns a middleware that records operation logs for mutating requests
// (POST/PUT/PATCH/DELETE) and GET requests to sensitive paths. It runs after
// the handler completes. The audit write is awaited with a short deadline so a
// completed request cannot leave an untracked background write behind.
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

		// Preserve the complete body for downstream handlers; cap only the audit copy.
		var bodySnippet string
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				auditBytes := bodyBytes
				if len(auditBytes) > 2048 {
					auditBytes = auditBytes[:2048]
				}
				bodySnippet = string(auditBytes)
				bodySnippet = operationlog.RedactSensitive(bodySnippet)
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
			Module:        moduleFromPath(path),
			Action:        actionFromMethod(method, c.FullPath()),
			ResourceID:    resourceIDFromCtx(c),
			Content:       composeAuditContent(c, bodySnippet, status),
			Operator:      operator,
			UserID:        userID,
			Result:        result,
			IP:            c.ClientIP(),
			Duration:      int(latency.Milliseconds()),
			CorrelationID: c.GetString("request_id"),
		}

		// Await the write so request completion has deterministic audit
		// semantics. The detached timeout context survives cancellation of the
		// incoming request while still bounding database latency.
		auditCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := svc.CreateWithContext(auditCtx, entry); err != nil {
			logger.Error("audit log write failed", zap.Error(err), zap.String("path", path))
		}
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
