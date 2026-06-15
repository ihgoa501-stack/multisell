import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'listing-tasks',
    name: 'ListingTaskQueue',
    component: () => import('@/views/listing/ListingTaskQueue.vue'),
    meta: {
      title: '上架任务',
      icon: 'list',
      menu: true,
      perm: 'listing:view',
    },
  },
]
