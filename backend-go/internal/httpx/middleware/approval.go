package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/platform/killswitch"
	"github.com/lingmirror/backend-go/internal/platform/routecatalog"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// requestTypeToAction is the explicit mapping from ApprovalRequest.RequestType
// (as used in the approval UI and agents) to routecatalog action type names
// (as used by the approval middleware). This replaces prefix guessing.
//
// The approval system uses human-readable types like "publish", "price_change",
// "listing_task". The actioncatalog uses names like "auto_publish",
// "price_update", "listing_optimize". This map bridges the two naming systems.
var requestTypeToAction = map[string][]string{
	"publish":                 {"auto_publish", "listing_optimize"},
	"price_change":            {"price_update"},
	"listing_task":            {"listing_optimize"},
	"list_generation":         {"listing_optimize", "auto_publish"},
	"order_update":            {"order_cancel"},
	"order_cancel":            {"order_cancel"},
	"refund":                  {"refund_issue"},
	"sync_inventory":          {"sync_inventory"},
	"credential":              {"credential_change"},
	"credential_change":       {"credential_change"},
	"permission":              {"permission_change"},
	"permission_change":       {"permission_change"},
	"finance":                 {"destructive_data_change"},
	"settlement":              {"destructive_data_change"},
	"content_update":          {"listing_optimize"},
	"delist":                  {"listing_optimize"},
	"agent_action":            {"agent_approve"},
	"destructive_data_change": {"destructive_data_change"},
}

// ApprovalRequired returns a middleware that blocks high-risk mutations
// unless a valid X-Approval-ID header is present and the corresponding
// approval request was approved.
//
// The middleware validates:
//   - Global kill switch (active → 503)
//   - Approval exists, is approved, and not expired
//   - Approval RequestType explicitly maps to the route's action type
//   - Approval TargetType matches the route context (if set)
//   - Approval TargetID/EntityID/ProductID matches the request path ID (if set)
//   - Approval RequesterUserID matches the JWT user (if set)
//
// Routes not in the routecatalog pass through unmodified.
// Only mutating methods (POST/PUT/PATCH/DELETE) are checked.
func ApprovalRequired(db *gorm.DB, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		fullPath := c.FullPath()

		if method != http.MethodPost && method != http.MethodPut &&
			method != http.MethodPatch && method != http.MethodDelete {
			c.Next()
			return
		}

		if !routecatalog.IsHighRisk(method, fullPath) {
			c.Next()
			return
		}

		actionType := routecatalog.GetActionType(method, fullPath)

		// Kill switch.
		if killswitch.IsActive() {
			logger.Warn("kill switch blocked request",
				zap.String("method", method),
				zap.String("path", fullPath),
				zap.String("action_type", actionType),
			)
			response.Error(c, http.StatusServiceUnavailable,
				"global kill switch is active: all production writes are blocked. Reason: "+killswitch.Reason())
			return
		}

		// Approval ID from header.
		approvalIDStr := c.GetHeader("X-Approval-ID")
		if approvalIDStr == "" {
			logger.Warn("approval required: missing X-Approval-ID header",
				zap.String("method", method),
				zap.String("path", fullPath),
			)
			response.Error(c, http.StatusForbidden,
				"this endpoint requires an approval ID via the X-Approval-ID header. Create an approval request first, then pass its ID.")
			return
		}

		approvalID, err := strconv.ParseInt(approvalIDStr, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "X-Approval-ID must be a numeric approval request ID")
			return
		}

		// Look up approval record.
		var req approval.ApprovalRequest
		if err := db.First(&req, approvalID).Error; err != nil {
			logger.Warn("approval validation failed: record not found",
				zap.Int64("approval_id", approvalID),
				zap.Error(err),
			)
			response.Error(c, http.StatusForbidden, "invalid or expired approval ID")
			return
		}

		// ── Binding check 1: Status ──
		if req.Status != approval.StatusApproved {
			logger.Warn("approval validation failed: not approved",
				zap.Int64("approval_id", approvalID),
				zap.String("status", req.Status),
			)
			response.Error(c, http.StatusForbidden, "approval has not been granted (status: "+req.Status+")")
			return
		}

		// ── Binding check 2: Expiry ──
		if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
			logger.Warn("approval validation failed: expired",
				zap.Int64("approval_id", approvalID),
				zap.Time("expires_at", *req.ExpiresAt),
			)
			response.Error(c, http.StatusForbidden, "approval has expired")
			return
		}

		// ── Binding check 3: RequestType → ActionType explicit mapping ──
		if req.RequestType != "" {
			if !requestTypeMatchesAction(req.RequestType, actionType) {
				logger.Warn("approval validation failed: RequestType mismatch",
					zap.Int64("approval_id", approvalID),
					zap.String("request_type", req.RequestType),
					zap.String("action_type", actionType),
				)
				response.Error(c, http.StatusForbidden,
					"approval type '"+req.RequestType+"' does not cover action '"+actionType+"'")
				return
			}
		}

		// ── Binding check 4: TargetType matches route context ──
		targetType := deriveTargetType(fullPath)
		if req.TargetType != "" && targetType != "" && req.TargetType != targetType {
			logger.Warn("approval validation failed: TargetType mismatch",
				zap.Int64("approval_id", approvalID),
				zap.String("target_type", req.TargetType),
				zap.String("route_target", targetType),
			)
			response.Error(c, http.StatusForbidden,
				"approval target type '"+req.TargetType+"' does not match route '"+targetType+"'")
			return
		}

		// ── Binding check 5: TargetID / EntityID / ProductID match request identity ──
		requestIDs := resolveRequestIDs(c, targetType)
		if hasApprovalIDConstraint(&req) && !approvalMatchesRequestIDs(&req, requestIDs) {
			logger.Warn("approval validation failed: entity ID mismatch",
				zap.Int64("approval_id", approvalID),
				zap.Any("request_ids", requestIDs),
				zap.Int64("approval_product_id", req.ProductID),
				zap.Int64("approval_target_id", req.TargetID),
				zap.Int64("approval_entity_id", req.EntityID),
			)
			response.Error(c, http.StatusForbidden,
				"approval was created for a different entity, or this request does not expose a verifiable target ID")
			return
		}

		// ── Binding check 6: RequesterUserID matches JWT user ──
		if req.RequesterUserID != nil {
			var userID int64
			if v, ok := c.Get("user_id"); ok {
				switch x := v.(type) {
				case int64:
					userID = x
				case float64:
					userID = int64(x)
				}
			}
			if userID != 0 && *req.RequesterUserID != 0 && *req.RequesterUserID != userID {
				logger.Warn("approval validation failed: requester mismatch",
					zap.Int64("approval_id", approvalID),
					zap.Int64("approval_requester", *req.RequesterUserID),
					zap.Int64("jwt_user_id", userID),
				)
				response.Error(c, http.StatusForbidden, "approval was created by a different user")
				return
			}
		}

		// Approval valid.
		c.Set("approval_id", approvalID)
		c.Set("approval_request", &req)
		logger.Info("approval validated",
			zap.Int64("approval_id", approvalID),
			zap.String("action_type", actionType),
			zap.String("path", fullPath),
		)
		c.Next()
	}
}

// requestTypeMatchesAction checks whether an approval RequestType explicitly
// maps to a routecatalog action type. Uses the requestTypeToAction table.
// Exact table lookup only — no prefix guessing.
func requestTypeMatchesAction(requestType, actionType string) bool {
	allowed, ok := requestTypeToAction[requestType]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == actionType {
			return true
		}
	}
	return false
}

// hasApprovalIDConstraint returns true if the approval request has at least
// one specific ID field set (ProductID, TargetID, or EntityID > 0).
func hasApprovalIDConstraint(req *approval.ApprovalRequest) bool {
	return req.ProductID > 0 || req.TargetID > 0 || req.EntityID > 0
}

// requestIdentity captures IDs exposed by the current HTTP request. The fields
// keep their business meaning so approvals cannot pass just because an unrelated
// numeric ID happens to match.
type requestIdentity struct {
	PrimaryID int64
	ProductID int64
	TargetID  int64
	EntityID  int64
	TaskID    int64
	OrderID   int64
	SkuID     int64
	ItemID    int64
}

// resolveRequestIDs returns typed IDs from path params, query params, and JSON
// body. The body is restored after reading so handlers can bind it normally.
func resolveRequestIDs(c *gin.Context, targetType string) requestIdentity {
	ids := requestIdentity{}
	assignID := func(key, value string) {
		if value == "" {
			return
		}
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n <= 0 {
			return
		}
		switch normalizeIDKey(key) {
		case "id":
			ids.PrimaryID = n
			assignPrimaryAlias(&ids, targetType, n)
		case "product_id":
			ids.ProductID = n
		case "target_id":
			ids.TargetID = n
		case "entity_id":
			ids.EntityID = n
		case "task_id":
			ids.TaskID = n
			if ids.EntityID == 0 {
				ids.EntityID = n
			}
		case "order_id":
			ids.OrderID = n
			if ids.EntityID == 0 {
				ids.EntityID = n
			}
		case "sku_id":
			ids.SkuID = n
		case "item_id":
			ids.ItemID = n
		}
	}

	for _, key := range []string{"id", "task_id", "order_id", "product_id", "productId", "sku_id", "item_id", "target_id", "entity_id"} {
		assignID(key, c.Param(key))
		assignID(key, c.Query(key))
	}

	if c.Request != nil && c.Request.Body != nil {
		body, err := io.ReadAll(c.Request.Body)
		if err == nil {
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			if len(bytes.TrimSpace(body)) > 0 && bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) {
				var payload map[string]interface{}
				if json.Unmarshal(body, &payload) == nil {
					for _, key := range []string{"id", "product_id", "productId", "target_id", "targetId", "entity_id", "entityId", "task_id", "taskId", "order_id", "orderId", "sku_id", "skuId", "item_id", "itemId"} {
						assignJSONID(assignID, key, payload[key])
					}
				}
			}
		}
	}

	return ids
}

func assignJSONID(assign func(string, string), key string, value interface{}) {
	switch v := value.(type) {
	case float64:
		assign(key, strconv.FormatInt(int64(v), 10))
	case string:
		assign(key, v)
	case json.Number:
		assign(key, string(v))
	}
}

func normalizeIDKey(key string) string {
	key = strings.ReplaceAll(key, "-", "_")
	var b strings.Builder
	for i, r := range key {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func assignPrimaryAlias(ids *requestIdentity, targetType string, id int64) {
	switch targetType {
	case "product-master", "product":
		ids.ProductID = id
	case "listing-task", "listing":
		ids.EntityID = id
	case "order", "aftersale", "settlement", "platform-integration", "finance", "rbac", "sku":
		ids.TargetID = id
	}
}

func approvalMatchesRequestIDs(req *approval.ApprovalRequest, ids requestIdentity) bool {
	matched := false
	if req.ProductID > 0 && ids.ProductID > 0 {
		if req.ProductID != ids.ProductID {
			return false
		}
		matched = true
	}
	if req.TargetID > 0 && ids.TargetID > 0 {
		if req.TargetID != ids.TargetID {
			return false
		}
		matched = true
	}
	if req.EntityID > 0 && ids.EntityID > 0 {
		if req.EntityID != ids.EntityID {
			return false
		}
		matched = true
	}
	if req.TargetID > 0 && ids.TargetID == 0 && ids.PrimaryID > 0 {
		if req.TargetID != ids.PrimaryID {
			return false
		}
		matched = true
	}
	if req.EntityID > 0 && ids.EntityID == 0 && ids.PrimaryID > 0 {
		if req.EntityID != ids.PrimaryID {
			return false
		}
		matched = true
	}
	return matched
}

// deriveTargetType extracts a target type from a Gin route path.
// "/api/v1/prices/:id" → "price"
// "/api/v1/rbac/roles" → "rbac"
// "/api/v1/inventory/:id" → "inventory"
func deriveTargetType(fullPath string) string {
	trimmed := strings.TrimPrefix(fullPath, "/api/v1/")
	if trimmed == "" {
		return ""
	}
	// Take the first path segment.
	idx := strings.Index(trimmed, "/")
	if idx > 0 {
		trimmed = trimmed[:idx]
	}
	// Remove trailing 's' for singular normalization (prices → price, listings → listing)
	if strings.HasSuffix(trimmed, "s") && len(trimmed) > 3 {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return trimmed
}
