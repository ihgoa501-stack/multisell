import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'finance/dashboard',
    name: 'ProfitDashboard',
    component: () => import('@/views/finance/ProfitDashboard.vue'),
    meta: {
      title: '利润看板',
      icon: 'trend',
      menu: true,
      perm: 'finance:report:view',
    },
  },
]
