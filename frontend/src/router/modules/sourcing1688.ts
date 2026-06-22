import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: '1688-sourcing',
    name: 'Sourcing1688List',
    component: () => import('@/views/sourcing1688/Sourcing1688List.vue'),
    meta: {
      title: '1688 货源池',
      icon: 'cloud-download',
      menu: true,
      perm: 'sourcing:view',
    },
  },
]
