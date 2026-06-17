import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'finance',
    name: 'FinanceManage',
    component: () => import('@/views/finance/FinanceManage.vue'),
    meta: { title: '财务管理', icon: 'trending-up', menu: true, perm: 'finance:view' },
  },
]
