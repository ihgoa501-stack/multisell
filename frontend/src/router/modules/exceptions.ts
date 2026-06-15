import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'exceptions',
    name: 'ExceptionWorkbench',
    component: () => import('@/views/exceptions/ExceptionWorkbench.vue'),
    meta: {
      title: '异常工作台',
      icon: 'warning',
      menu: true,
      perm: 'exception:view',
    },
  },
]
