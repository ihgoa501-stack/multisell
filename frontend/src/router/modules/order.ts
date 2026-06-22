/**
 * 订单管理路由模块
 *
 * 约定：
 * - 导出 routes: RouteRecordRaw[] 数组
 * - path 不加 "/"，是 Layout.vue 的子路由
 * - meta.menu = true + meta.icon 让侧边菜单自动出现
 */

import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'orders',
    name: 'OrderList',
    component: () => import('@/views/order/OrderList.vue'),
    meta: { title: '订单管理', icon: 'cart', menu: true, perm: 'order:view' },
  },
  {
    path: 'orders/:id',
    name: 'OrderDetail',
    component: () => import('@/views/order/OrderDetail.vue'),
    meta: { title: '订单详情', menu: false },
  },
]
