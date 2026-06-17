import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'order-import',
    name: 'OrderImport',
    component: () => import('@/views/order_import/OrderImport.vue'),
    meta: { title: '订单导入', icon: 'download', menu: true, perm: 'order:view' },
  },
]
