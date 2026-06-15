import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'settlement/import',
    name: 'SettlementImport',
    component: () => import('@/views/settlement/SettlementImport.vue'),
    meta: { title: '平台结算', icon: 'money', menu: true, perm: 'settlement:view' },
  },
]
