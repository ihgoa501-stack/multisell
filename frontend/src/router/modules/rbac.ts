/**
 * RBAC 权限管理路由配置
 */
import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'users',
    name: 'UserManagement',
    component: () => import('@/views/rbac/UserManagement.vue'),
    meta: { title: '用户管理', icon: 'people', menu: true },
  },
  {
    path: 'roles',
    name: 'RoleManagement',
    component: () => import('@/views/rbac/RoleManagement.vue'),
    meta: { title: '角色管理', icon: 'shield', menu: true },
  },
]
