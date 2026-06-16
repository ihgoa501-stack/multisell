import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'allocations',
    name: 'CostAllocation',
    component: () => import('@/views/allocation/CostAllocation.vue'),
    meta: {
      title: '费用分摊',
      icon: 'trend',
      menu: true,
      perm: 'allocation:view',
    },
  },
]
