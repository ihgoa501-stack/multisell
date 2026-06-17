import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'decisions/prelisting',
    name: 'PreListingDecision',
    component: () => import('@/views/decision/DecisionWorkbench.vue'),
    meta: {
      title: '上架决策',
      icon: 'analytics',
      menu: true,
      perm: 'decision:calculate',
    },
  },
  {
    path: 'decisions/prelisting/batch',
    name: 'BatchPreListingDecision',
    component: () => import('@/views/decision/BatchPreListingDecision.vue'),
    meta: {
      title: '批量上架决策',
      icon: 'analytics',
      menu: true,
      perm: 'decision:calculate',
    },
  },
]
