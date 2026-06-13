/**
 * 路由模块示例 — 新建模块时复制这个文件按格式写
 *
 * 规则：
 * 1. 导出名为 `routes` 的 RouteRecordRaw[] 数组
 * 2. 所有路由 path 前面不加 "/"，是 Layout.vue 的子路由（跟 /products 一样）
 * 3. 想要在侧边菜单显示，设置 meta.menu = true + meta.icon （图标key见 Layout.vue 的 iconMap）
 * 4. 不改 router/index.ts、不改 Layout.vue
 */

import type { RouteRecordRaw } from 'vue-router'

/**
 * @example 订单管理模块
 */
export const routes: RouteRecordRaw[] = [
  {
    path: 'orders',
    name: 'OrderList',
    component: () => import('@/views/order/OrderList.vue'),
    meta: { title: '订单管理', icon: 'cart', menu: true },
  },
  {
    path: 'orders/:id',
    name: 'OrderDetail',
    component: () => import('@/views/order/OrderDetail.vue'),
    meta: { title: '订单详情', menu: false },
  },
]
