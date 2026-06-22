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
  {
    path: 'listing-tasks/:id',
    name: 'ListingTaskDetail',
    component: () => import('@/views/listing_task/ListingTaskDetail.vue'),
    meta: {
      title: '上架任务详情',
    },
  },
]
