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
		// Roles
		rbac.GET("/roles", h.ListRoles)
		rbac.POST("/roles", h.CreateRole)
		rbac.GET("/roles/:id", h.GetRole)
		rbac.PUT("/roles/:id", h.UpdateRole)
		rbac.DELETE("/roles/:id", h.DeleteRole)
		rbac.GET("/roles/:id/permissions", h.GetRolePermissions)
		rbac.POST("/roles/:id/permissions", h.AssignRolePermissions)

		// Permissions
		rbac.GET("/permissions", h.ListPermissions)
		rbac.POST("/permissions", h.CreatePermission)
		rbac.GET("/permissions/:id", h.GetPermission)
		rbac.PUT("/permissions/:id", h.UpdatePermission)
		rbac.DELETE("/permissions/:id", h.DeletePermission)

		// User-Role assignment (placeholder auth to keep routes functional)
		rbac.GET("/users/:id/roles", h.GetUserRoles)
		rbac.POST("/users/:id/roles", h.AssignUserRoles)
	}
}
