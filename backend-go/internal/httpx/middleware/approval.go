package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
// Every mutation must have an explicit route policy. Standard authenticated
// mutations pass through to the global synchronous audit middleware; high-risk
// mutations additionally execute the approval protocol. Missing policy fails
// closed so a newly added write route cannot silently bypass classification.
func ApprovalRequired(db *gorm.DB, logger *zap.Logger) gin.HandlerFunc {
	approvalSvc := approval.NewService(db, logger, nil)
	return func(c *gin.Context) {
		method := c.Request.Method
		if method != http.MethodPost && method != http.MethodPut &&
			method != http.MethodPatch && method != http.MethodDelete {
			c.Next()
			return
		}

		fullPath := c.FullPath()
		policy, classified := routecatalog.GetMutationPolicy(method, fullPath)
		if !classified {
			policy, classified = routecatalog.ResolveMutationPolicy(method, c.Request.URL.Path)
			if classified {
				fullPath = policy.Path
			}
		}
		if !classified {
			logger.Error("mutation route has no explicit security policy",
				zap.String("method", method), zap.String("path", c.Request.URL.Path))
			abortApproval(c, http.StatusInternalServerError, "mutation route security policy is missing")
			return
		}
		if policy.Class != routecatalog.MutationHigh {
			c.Next()
			return
		}
		actionType := policy.ActionType

		idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if len(idempotencyKey) < 8 || len(idempotencyKey) > 255 || strings.ContainsAny(idempotencyKey, "\r\n\t ") {
			abortApproval(c, http.StatusBadRequest, "high-risk writes require an Idempotency-Key header of 8-255 non-whitespace characters")
			return
		}

		// Kill switch.
		if killswitch.IsActive() {
			logger.Warn("kill switch blocked request",
				zap.String("method", method),
				zap.String("path", fullPath),
				zap.String("action_type", actionType),
			)
			abortApproval(c, http.StatusServiceUnavailable,
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
			abortApproval(c, http.StatusForbidden,
				"this endpoint requires an approval ID via the X-Approval-ID header. Create an approval request first, then pass its ID.")
			return
		}

		approvalID, err := strconv.ParseInt(approvalIDStr, 10, 64)
		if err != nil {
			abortApproval(c, http.StatusBadRequest, "X-Approval-ID must be a numeric approval request ID")
			return
		}

		// Look up approval record.
		var req approval.ApprovalRequest
		if err := db.First(&req, approvalID).Error; err != nil {
			logger.Warn("approval validation failed: record not found",
				zap.Int64("approval_id", approvalID),
				zap.Error(err),
			)
			abortApproval(c, http.StatusForbidden, "invalid or expired approval ID")
			return
		}

		// ── Binding check 1: Status ──
		if req.Status != approval.StatusApproved {
			logger.Warn("approval validation failed: not approved",
				zap.Int64("approval_id", approvalID),
				zap.String("status", req.Status),
			)
			abortApproval(c, http.StatusForbidden, "approval has not been granted (status: "+req.Status+")")
			return
		}

		// ── Binding check 2: Expiry ──
		if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
			logger.Warn("approval validation failed: expired",
				zap.Int64("approval_id", approvalID),
				zap.Time("expires_at", *req.ExpiresAt),
			)
			abortApproval(c, http.StatusForbidden, "approval has expired")
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
				abortApproval(c, http.StatusForbidden,
					"approval type '"+req.RequestType+"' does not cover action '"+actionType+"'")
				return
			}
		}

		// ── Binding check 4: TargetType matches route context ──
		targetType := deriveTargetType(fullPath)
		binding, _ := routecatalog.GetBinding(method, fullPath)
		if binding.TargetType != "" {
			targetType = binding.TargetType
		}
		if req.TargetType != "" && targetType != "" && req.TargetType != targetType {
			logger.Warn("approval validation failed: TargetType mismatch",
				zap.Int64("approval_id", approvalID),
				zap.String("target_type", req.TargetType),
				zap.String("route_target", targetType),
			)
			abortApproval(c, http.StatusForbidden,
				"approval target type '"+req.TargetType+"' does not match route '"+targetType+"'")
			return
		}

		// ── Binding check 5: TargetID / EntityID / ProductID match request identity ──
		requestIDs := resolveRequestIDs(c, targetType)
		if binding.TargetIDParam != "" {
			targetID, parseErr := strconv.ParseInt(c.Param(binding.TargetIDParam), 10, 64)
			if parseErr != nil || targetID <= 0 {
				abortApproval(c, http.StatusBadRequest, "high-risk route target ID is missing or invalid")
				return
			}
			requestIDs.TargetID = targetID
			requestIDs.PrimaryID = targetID
		}
		if hasApprovalIDConstraint(&req) && !approvalMatchesRequestIDs(&req, requestIDs) {
			logger.Warn("approval validation failed: entity ID mismatch",
				zap.Int64("approval_id", approvalID),
				zap.Any("request_ids", requestIDs),
				zap.Int64("approval_product_id", req.ProductID),
				zap.Int64("approval_target_id", req.TargetID),
				zap.Int64("approval_entity_id", req.EntityID),
			)
			abortApproval(c, http.StatusForbidden,
				"approval was created for a different entity, or this request does not expose a verifiable target ID")
			return
		}
		executionTargetID := approvalExecutionTargetID(&req, requestIDs, fullPath)

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
				abortApproval(c, http.StatusForbidden, "approval was created by a different user")
				return
			}
		}

		// Atomically bind this approval to exactly one logical HTTP write.
		if err := approvalSvc.AuthorizeExecution(c.Request.Context(), approvalID, actionType, targetType, executionTargetID, idempotencyKey); err != nil {
			logger.Warn("approval execution authorization failed", zap.Int64("approval_id", approvalID), zap.String("idempotency_key", idempotencyKey), zap.Error(err))
			abortApproval(c, http.StatusConflict, "approval has already been consumed or is bound to another execution")
			return
		}
		if err := approvalSvc.ConsumeExecution(c.Request.Context(), approvalID, actionType, targetType, executionTargetID, idempotencyKey); err != nil {
			status := http.StatusConflict
			if !errors.Is(err, approval.ErrExecutionInProgress) && !errors.Is(err, approval.ErrApprovalConsumed) {
				status = http.StatusForbidden
			}
			abortApproval(c, status, "approval execution cannot be claimed")
			return
		}

		// Approval valid and consumed for this logical request.
		c.Set("approval_id", approvalID)
		c.Set("approval_request", &req)
		c.Set("idempotency_key", idempotencyKey)
		logger.Info("approval validated",
			zap.Int64("approval_id", approvalID),
			zap.String("action_type", actionType),
			zap.String("path", fullPath),
		)
		c.Next()
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 3*time.Second)
		defer cancel()
		if c.Writer.Status() >= http.StatusBadRequest {
			cause := fmt.Errorf("HTTP execution failed with status %d", c.Writer.Status())
			if err := approvalSvc.FailExecution(persistCtx, approvalID, idempotencyKey, cause); err != nil {
				logger.Error("failed to persist HTTP approval execution failure", zap.Int64("approval_id", approvalID), zap.Error(err))
			}
		} else if err := approvalSvc.CompleteExecution(persistCtx, approvalID, idempotencyKey); err != nil {
			logger.Error("failed to persist HTTP approval execution success", zap.Int64("approval_id", approvalID), zap.Error(err))
		}
	}
}

func abortApproval(c *gin.Context, status int, message string) {
	response.Error(c, status, message)
	c.Abort()
}

func approvalExecutionTargetID(req *approval.ApprovalRequest, ids requestIdentity, fullPath string) string {
	parts := make([]string, 0, 3)
	if req.ProductID > 0 {
		parts = append(parts, "product="+strconv.FormatInt(req.ProductID, 10))
	}
	if req.TargetID > 0 {
		parts = append(parts, "target="+strconv.FormatInt(req.TargetID, 10))
	}
	if req.EntityID > 0 {
		parts = append(parts, "entity="+strconv.FormatInt(req.EntityID, 10))
	}
	if len(parts) > 1 {
		return strings.Join(parts, ";")
	}
	for _, id := range []int64{req.TargetID, req.EntityID, req.ProductID, ids.TargetID, ids.EntityID, ids.ProductID, ids.PrimaryID} {
		if id > 0 {
			return strconv.FormatInt(id, 10)
		}
	}
	return fullPath
}

// requestTypeMatchesAction checks whether an approval RequestType explicitly
// maps to a routecatalog action type. Uses the approval domain's canonical table.
// Exact table lookup only — no prefix guessing.
func requestTypeMatchesAction(requestType, actionType string) bool {
	return approval.RequestTypeCoversAction(requestType, actionType)
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
