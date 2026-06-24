package rbac

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles RBAC HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new RBAC handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context, field string) (int64, bool) {
	idStr := c.Param(field)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}

// ===================== Roles =====================

// ListRoles GET /api/v1/rbac/roles
func (h *Handler) ListRoles(c *gin.Context) {
	status := -1
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			status = v
		}
	}
	p := common.ParsePagination(c)
	roles, total, err := h.service.ListRoles(status, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, roles, total, p.Page, p.Size)
}

// GetRole GET /api/v1/rbac/roles/:id
func (h *Handler) GetRole(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	r, err := h.service.GetRole(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "角色不存在")
		return
	}
	response.Success(c, r)
}

// CreateRole POST /api/v1/rbac/roles
func (h *Handler) CreateRole(c *gin.Context) {
	var r Role
	if err := c.ShouldBindJSON(&r); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if strings.TrimSpace(r.Code) == "" || strings.TrimSpace(r.Name) == "" {
		response.Error(c, http.StatusBadRequest, "角色名称和代码不能为空")
		return
	}
	if err := h.service.CreateRole(&r); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}

// UpdateRole PUT /api/v1/rbac/roles/:id
func (h *Handler) UpdateRole(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	r, err := h.service.GetRole(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "角色不存在")
		return
	}
	if err := c.ShouldBindJSON(r); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	r.ID = id
	if err := h.service.UpdateRole(r); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}

// DeleteRole DELETE /api/v1/rbac/roles/:id
func (h *Handler) DeleteRole(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteRole(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ===================== Permissions =====================

// ListPermissions GET /api/v1/rbac/permissions
func (h *Handler) ListPermissions(c *gin.Context) {
	module := c.Query("module")
	p := common.ParsePagination(c)
	perms, total, err := h.service.ListPermissions(module, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, perms, total, p.Page, p.Size)
}

// GetPermission GET /api/v1/rbac/permissions/:id
func (h *Handler) GetPermission(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	p, err := h.service.GetPermission(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "权限不存在")
		return
	}
	response.Success(c, p)
}

// CreatePermission POST /api/v1/rbac/permissions
func (h *Handler) CreatePermission(c *gin.Context) {
	var p Permission
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if strings.TrimSpace(p.Code) == "" || strings.TrimSpace(p.Name) == "" {
		response.Error(c, http.StatusBadRequest, "权限名称和代码不能为空")
		return
	}
	if err := h.service.CreatePermission(&p); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// UpdatePermission PUT /api/v1/rbac/permissions/:id
func (h *Handler) UpdatePermission(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	p, err := h.service.GetPermission(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "权限不存在")
		return
	}
	if err := c.ShouldBindJSON(p); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	p.ID = id
	if err := h.service.UpdatePermission(p); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// DeletePermission DELETE /api/v1/rbac/permissions/:id
func (h *Handler) DeletePermission(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeletePermission(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ===================== User-Role =====================

type assignRolesRequest struct {
	RoleIDs []int64 `json:"role_ids"`
}

// AssignUserRoles POST /api/v1/rbac/users/:id/roles
func (h *Handler) AssignUserRoles(c *gin.Context) {
	userID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req assignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if err := h.service.AssignUserRoles(userID, req.RoleIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"user_id": userID, "role_ids": req.RoleIDs})
}

// GetUserRoles GET /api/v1/rbac/users/:id/roles
func (h *Handler) GetUserRoles(c *gin.Context) {
	userID, ok := parseID(c, "id")
	if !ok {
		return
	}
	roles, err := h.service.GetUserRoles(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"user_id": userID, "roles": roles})
}

// ===================== Role-Permission =====================

type assignPermissionsRequest struct {
	PermissionIDs []int64 `json:"permission_ids"`
}

// GetRolePermissions GET /api/v1/rbac/roles/:id/permissions
func (h *Handler) GetRolePermissions(c *gin.Context) {
	roleID, ok := parseID(c, "id")
	if !ok {
		return
	}
	perms, err := h.service.GetRolePermissions(roleID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"role_id": roleID, "permissions": perms})
}

// AssignRolePermissions POST /api/v1/rbac/roles/:id/permissions
func (h *Handler) AssignRolePermissions(c *gin.Context) {
	roleID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req assignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if err := h.service.AssignRolePermissions(roleID, req.PermissionIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"role_id": roleID, "permission_ids": req.PermissionIDs})
}
