package rbac

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RequirePermission returns a Gin middleware that checks whether the
// authenticated user holds the given permission code.
//
// It reads user_id from the Gin context (set by the JWT auth middleware),
// then looks up the user's aggregated permissions via GetUserPermissions.
// If the user lacks the required permission, the request is aborted with 403.
func RequirePermission(db *gorm.DB, logger *zap.Logger, code string) gin.HandlerFunc {
	svc := NewService(db, logger)
	return func(c *gin.Context) {
		raw, exists := c.Get("user_id")
		if !exists {
			response.Error(c, http.StatusUnauthorized, "未认证")
			c.Abort()
			return
		}

		// user_id is float64 from JWT MapClaims.
		userID, ok := raw.(float64)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "无效的用户标识")
			c.Abort()
			return
		}

		permissions, err := svc.GetUserPermissions(int64(userID))
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "权限检查失败")
			c.Abort()
			return
		}

		// Admin role has implicit access to everything.
		for _, p := range permissions {
			if p == "*" || p == code {
				c.Next()
				return
			}
		}

		response.Error(c, http.StatusForbidden, "无权限执行此操作")
		c.Abort()
	}
}
