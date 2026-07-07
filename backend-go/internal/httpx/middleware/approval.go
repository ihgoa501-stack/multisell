package middleware

import (
	"net/http"
	"strconv"
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
// The middleware also checks the global kill switch before letting any
// high-risk mutation through.
//
// Routes that are not in the routecatalog are passed through unmodified.
// Only mutating methods (POST/PUT/PATCH/DELETE) are checked.
func ApprovalRequired(db *gorm.DB, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		path := c.FullPath()

		// Only check mutating methods.
		if method != http.MethodPost && method != http.MethodPut &&
			method != http.MethodPatch && method != http.MethodDelete {
			c.Next()
			return
		}

		// Skip if this route is not in the high-risk registry.
		if !routecatalog.IsHighRisk(method, path) {
			c.Next()
			return
		}

		actionType := routecatalog.GetActionType(method, path)

		// ── Kill switch check ──
		if killswitch.IsActive() {
			logger.Warn("kill switch blocked request",
				zap.String("method", method),
				zap.String("path", path),
				zap.String("action_type", actionType),
			)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code":    503,
				"message": "global kill switch is active: all production writes are blocked",
				"reason":  killswitch.Reason(),
			})
			return
		}

		// ── Approval check ──
		approvalIDStr := c.GetHeader("X-Approval-ID")
		if approvalIDStr == "" {
			logger.Warn("approval required: missing X-Approval-ID header",
				zap.String("method", method),
				zap.String("path", path),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":       403,
				"message":    "this endpoint requires an approval ID via the X-Approval-ID header",
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

		// Validate against the approval_request table.
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

		// Check expiry.
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

		// Approval valid — pass the approval context downstream.
		c.Set("approval_id", approvalID)
		c.Set("approval_request", &req)
		logger.Info("approval validated",
			zap.Int64("approval_id", approvalID),
			zap.String("action_type", actionType),
			zap.String("path", path),
		)
		c.Next()
	}
}
