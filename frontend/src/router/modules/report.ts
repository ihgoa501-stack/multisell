/**
 * 数据报表路由模块
 *
 * 规则：
 * 1. 导出名为 `routes` 的 RouteRecordRaw[] 数组
 * 2. 路由 path 前面不加 "/"，是 Layout.vue 的子路由
 * 3. meta.menu = true + meta.icon = 'chart' 会在侧边菜单显示
 */
import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'reports',
    name: 'Report',
    component: () => import('@/views/report/index.vue'),
    meta: { title: '数据报表', icon: 'chart', menu: true },
  },
]
