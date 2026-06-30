package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/rbac"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RequirePermission returns a Gin middleware that checks the authenticated user
// has at least one of the specified permission codes. Permissions are loaded
// lazily into gin context on the first check and cached for the request cycle.
func RequirePermission(db *gorm.DB, codes ...string) gin.HandlerFunc {
	svc := rbac.NewService(db, zap.NewNop())

	return func(c *gin.Context) {
		// Read user_id set by Auth middleware
		uid, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "not authenticated",
			})
			return
		}

		var userID int64
		switch v := uid.(type) {
		case float64:
			userID = int64(v)
		case int64:
			userID = v
		case int:
			userID = int64(v)
		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid user identity",
			})
			return
		}

		// Check if permissions already cached in context (lazy load)
		permIface, exists := c.Get("_rbac_perms")
		var permCodes []string
		if exists {
			permCodes = permIface.([]string)
		} else {
			var err error
			permCodes, err = svc.GetUserPermissions(userID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "failed to load permissions",
				})
				return
			}
			c.Set("_rbac_perms", permCodes)
		}

		// Build set for O(1) lookup
		set := make(map[string]struct{}, len(permCodes))
		for _, code := range permCodes {
			set[code] = struct{}{}
		}

		// Check if user has any of the required permission codes
		for _, required := range codes {
			if _, ok := set[required]; ok {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "insufficient permissions",
		})
	}
}
