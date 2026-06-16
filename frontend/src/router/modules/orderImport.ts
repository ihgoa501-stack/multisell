import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'order-imports',
    name: 'OrderImport',
    component: () => import('@/views/order/OrderImport.vue'),
    meta: {
      title: '订单导入',
      icon: 'upload',
      menu: true,
      perm: 'order_import:view',
    },
  },
]
