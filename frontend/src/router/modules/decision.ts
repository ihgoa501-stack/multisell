import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'decisions/prelisting',
    name: 'PreListingDecision',
    component: () => import('@/views/decision/PreListingDecision.vue'),
    meta: {
      title: '上架决策',
      icon: 'analytics',
      menu: true,
      perm: 'decision:calculate',
    },
  },
]
