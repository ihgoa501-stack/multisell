import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: '/redesign',
    name: 'RedesignLayout',
    component: () => import('@/views-redesign/layout/RedesignLayout.vue'),
    redirect: '/redesign/dashboard',
    meta: { standalone: true },
    children: [
      {
        path: 'dashboard',
        name: 'RedesignDashboard',
        component: () => import('@/views-redesign/dashboard/Dashboard.vue'),
        meta: { menu: false, title: '工作台' },
      },
      {
        path: 'agentos',
        name: 'RedesignAgentOS',
        component: () => import('@/views-redesign/agentos/ControlCenter.vue'),
        meta: { menu: false, title: 'AgentOS 总控台' },
      },
      {
        path: 'products',
        name: 'RedesignProductList',
        component: () => import('@/views-redesign/product/ProductList.vue'),
        meta: { menu: false, title: '商品管理' },
      },
    ],
  },
]
