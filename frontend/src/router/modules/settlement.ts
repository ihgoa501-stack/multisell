import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'settlements',
    name: 'SettlementList',
    component: () => import('@/views/settlement/SettlementList.vue'),
    meta: {
      title: '结算管理',
      icon: 'cash',
      menu: true,
      perm: 'settlement:view',
    },
  },
  {
    path: 'settlements/:id',
    name: 'SettlementDetail',
    component: () => import('@/views/settlement/SettlementDetail.vue'),
    meta: {
      title: '结算详情',
      menu: false,
      perm: 'settlement:view',
    },
  },
]
