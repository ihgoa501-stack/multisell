import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'notifications',
    name: 'NotificationCenter',
    component: () => import('@/views/notification/NotificationCenter.vue'),
    meta: { title: '通知中心', icon: 'notification', menu: true, perm: 'notification:view' },
  },
]
