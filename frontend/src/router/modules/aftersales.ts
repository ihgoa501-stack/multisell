import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'aftersales',
    name: 'ReturnList',
    component: () => import('@/views/aftersales/ReturnList.vue'),
    meta: {
      title: '退货管理',
      icon: 'return-up-back',
      menu: true,
      perm: 'aftersales:view',
    },
  },
]
