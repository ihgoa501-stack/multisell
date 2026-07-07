package middleware

import (
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
	"publish":       {"auto_publish", "listing_optimize"},
	"price_change":  {"price_update"},
	"listing_task":  {"listing_optimize"},
	"list_generation": {"listing_optimize", "auto_publish"},
	"order_update":  {"order_cancel"},
	"order_cancel":  {"order_cancel"},
	"refund":        {"refund_issue"},
	"sync_inventory": {"sync_inventory"},
	"credential":    {"credential_change"},
	"credential_change": {"credential_change"},
	"permission":    {"permission_change"},
	"permission_change": {"permission_change"},
	"finance":       {"destructive_data_change"},
	"settlement":    {"destructive_data_change"},
	"content_update": {"listing_optimize"},
	"delist":        {"listing_optimize"},
	"agent_action":  {"agent_approve"},
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

		// ── Binding check 5: TargetID / EntityID / ProductID match path ──
		paramIDStr := resolvePathID(c)
		if paramIDStr != "" {
			paramID, parseErr := strconv.ParseInt(paramIDStr, 10, 64)
			if parseErr == nil && hasApprovalIDConstraint(&req) && !matchesAnyID(&req, paramID) {
				logger.Warn("approval validation failed: entity ID mismatch",
					zap.Int64("approval_id", approvalID),
					zap.Int64("param_id", paramID),
					zap.Int64("approval_product_id", req.ProductID),
					zap.Int64("approval_target_id", req.TargetID),
					zap.Int64("approval_entity_id", req.EntityID),
				)
				response.Error(c, http.StatusForbidden,
					"approval was created for a different entity (ID mismatch)")
				return
			}
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

// resolvePathID returns the first matched numeric path parameter from the Gin
// context. Checks common ID param names in order of specificity.
func resolvePathID(c *gin.Context) string {
	for _, key := range []string{"id", "task_id", "order_id", "product_id", "productId", "sku_id", "item_id", "ref-id"} {
		if v := c.Param(key); v != "" {
			return v
		}
	}
	return ""
}

// hasApprovalIDConstraint returns true if the approval request has at least
// one specific ID field set (ProductID, TargetID, or EntityID > 0).
func hasApprovalIDConstraint(req *approval.ApprovalRequest) bool {
	return req.ProductID > 0 || req.TargetID > 0 || req.EntityID > 0
}

// matchesAnyID returns true if paramID matches any of the approval's ID fields.
func matchesAnyID(req *approval.ApprovalRequest, paramID int64) bool {
	if req.ProductID > 0 && req.ProductID == paramID {
		return true
	}
	if req.TargetID > 0 && req.TargetID == paramID {
		return true
	}
	if req.EntityID > 0 && req.EntityID == paramID {
		return true
	}
	return false
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
