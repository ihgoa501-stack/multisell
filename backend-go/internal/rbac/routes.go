package rbac

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers RBAC routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	rbac := rg.Group("/rbac")
	{
		// Roles — read
		rbac.GET("/roles", RequirePermission(db, logger, "rbac:view"), h.ListRoles)
		rbac.GET("/roles/:id", RequirePermission(db, logger, "rbac:view"), h.GetRole)
		rbac.GET("/roles/:id/permissions", RequirePermission(db, logger, "rbac:view"), h.GetRolePermissions)

		// Roles — write
		rbac.POST("/roles", RequirePermission(db, logger, "rbac:manage"), h.CreateRole)
		rbac.PUT("/roles/:id", RequirePermission(db, logger, "rbac:manage"), h.UpdateRole)
		rbac.DELETE("/roles/:id", RequirePermission(db, logger, "rbac:manage"), h.DeleteRole)
		rbac.POST("/roles/:id/permissions", RequirePermission(db, logger, "rbac:manage"), h.AssignRolePermissions)

		// Permissions — read
		rbac.GET("/permissions", RequirePermission(db, logger, "rbac:view"), h.ListPermissions)
		rbac.GET("/permissions/:id", RequirePermission(db, logger, "rbac:view"), h.GetPermission)

		// Permissions — write
		rbac.POST("/permissions", RequirePermission(db, logger, "rbac:manage"), h.CreatePermission)
		rbac.PUT("/permissions/:id", RequirePermission(db, logger, "rbac:manage"), h.UpdatePermission)
		rbac.DELETE("/permissions/:id", RequirePermission(db, logger, "rbac:manage"), h.DeletePermission)

		// Current user permissions (self-service, no permission needed — user can see their own)
		rbac.GET("/current/permissions", h.GetCurrentUserPermissions)

		// User-Role assignment
		rbac.GET("/users/:id/roles", RequirePermission(db, logger, "rbac:view"), h.GetUserRoles)
		rbac.POST("/users/:id/roles", RequirePermission(db, logger, "rbac:manage"), h.AssignUserRoles)
	}
}
