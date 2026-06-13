/**
 * RBAC 权限管理 API 模块
 */
import http from '@/api/http'

export const rbacApi = {
  // ========== 角色 ==========
  listRoles(params?: any) {
    return http.get('/roles', { params })
  },
  getRole(id: number) {
    return http.get(`/roles/${id}`)
  },
  createRole(data: any) {
    return http.post('/roles', data)
  },
  updateRole(id: number, data: any) {
    return http.put(`/roles/${id}`, data)
  },
  deleteRole(id: number) {
    return http.delete(`/roles/${id}`)
  },
  assignRolePermissions(roleId: number, permissionIds: number[]) {
    return http.post(`/roles/${roleId}/permissions`, { role_ids: permissionIds })
  },

  // ========== 权限 ==========
  listPermissions(params?: any) {
    return http.get('/permissions', { params })
  },
  getPermission(id: number) {
    return http.get(`/permissions/${id}`)
  },
  createPermission(data: any) {
    return http.post('/permissions', data)
  },
  updatePermission(id: number, data: any) {
    return http.put(`/permissions/${id}`, data)
  },
  deletePermission(id: number) {
    return http.delete(`/permissions/${id}`)
  },

  // ========== 用户-角色 ==========
  listUsers(params?: any) {
    return http.get('/users', { params })
  },
  assignUserRoles(userId: number, roleIds: number[]) {
    return http.post(`/users/${userId}/roles`, { role_ids: roleIds })
  },
  getUserPermissions(userId: number) {
    return http.get(`/users/${userId}/permissions`)
  },
}
