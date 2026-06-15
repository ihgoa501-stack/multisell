import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'shipping/manage',
    name: 'ShippingManage',
    component: () => import('@/views/shipping/ShippingManage.vue'),
    meta: { title: '物流管理', icon: 'cube', menu: true, perm: 'shipping:view' },
  },
  {
    path: 'shipping/calculator',
    name: 'ShippingCalculator',
    component: () => import('@/views/shipping/ShippingCalculator.vue'),
    meta: { title: '运费计算器', icon: 'trend', menu: true, perm: 'shipping:calculate' },
  },
]
