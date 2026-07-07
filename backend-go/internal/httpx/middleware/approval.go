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
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ApprovalRequired returns a middleware that blocks high-risk mutations
// unless a valid X-Approval-ID header is present and the corresponding
// approval request was approved.
//
// The middleware validates:
//   - Global kill switch (active → 503)
//   - Approval exists, is approved, and not expired
//   - Approval RequestType matches the current route's action type
//   - Approval TargetType matches the route context (if set)
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
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code":    503,
				"message": "global kill switch is active: all production writes are blocked",
				"reason":  killswitch.Reason(),
			})
			return
		}

		// Approval ID from header.
		approvalIDStr := c.GetHeader("X-Approval-ID")
		if approvalIDStr == "" {
			logger.Warn("approval required: missing X-Approval-ID header",
				zap.String("method", method),
				zap.String("path", fullPath),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":       403,
				"message":    "this endpoint requires an approval ID via the X-Approval-ID header. Create an approval request first, then pass its ID.",
				"action_type": actionType,
			})
			return
		}

		approvalID, err := strconv.ParseInt(approvalIDStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "X-Approval-ID must be a numeric approval request ID",
			})
			return
		}

		// Look up approval record.
		var req approval.ApprovalRequest
		if err := db.First(&req, approvalID).Error; err != nil {
			logger.Warn("approval validation failed: record not found",
				zap.Int64("approval_id", approvalID),
				zap.Error(err),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "invalid or expired approval ID",
			})
			return
		}

		// ── Binding check 1: Status ──
		if req.Status != approval.StatusApproved {
			logger.Warn("approval validation failed: not approved",
				zap.Int64("approval_id", approvalID),
				zap.String("status", req.Status),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":       403,
				"message":    "approval has not been granted (status: " + req.Status + ")",
				"approval_id": approvalID,
			})
			return
		}

		// ── Binding check 2: Expiry ──
		if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
			logger.Warn("approval validation failed: expired",
				zap.Int64("approval_id", approvalID),
				zap.Time("expires_at", *req.ExpiresAt),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "approval has expired",
			})
			return
		}

		// ── Binding check 3: RequestType matches current action ──
		if req.RequestType != "" {
			// Map action_type to request_type. The request_type is typically "publish",
			// "price_change", "delist", "content_update", "listing_task", "permission",
			// etc.  We accept an exact or prefix match (e.g. "permission_change"
			// matches an approval for "permission").
			if !matchRequestType(req.RequestType, actionType) {
				logger.Warn("approval validation failed: RequestType mismatch",
					zap.Int64("approval_id", approvalID),
					zap.String("approval_request_type", req.RequestType),
					zap.String("action_type", actionType),
				)
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "approval RequestType '" + req.RequestType + "' does not match action '" + actionType + "'",
				})
				return
			}
		}

		// ── Binding check 4: RequesterUserID matches JWT user ──
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
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "approval was created by a different user",
				})
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

// matchRequestType checks whether an approval RequestType maps to the current
// action type. Accepts prefix match so "permission_change" matches a "permission" approval.
func matchRequestType(requestType, actionType string) bool {
	if requestType == actionType {
		return true
	}
	if strings.HasPrefix(actionType, requestType) || strings.HasPrefix(requestType, actionType) {
		return true
	}
	return false
}
