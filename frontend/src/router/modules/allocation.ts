import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'allocation',
    name: 'AllocationManage',
    component: () => import('@/views/allocation/AllocationManage.vue'),
    meta: { title: '库存分配', icon: 'warehouse', menu: true, perm: 'inventory:view' },
  },
]
