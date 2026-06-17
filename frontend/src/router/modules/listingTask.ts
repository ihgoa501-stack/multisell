import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'listings/ai-workbench',
    name: 'AiListingWorkbench',
    component: () => import('@/views/listing_task/AiListingWorkbench.vue'),
    meta: {
      title: 'AI多平台上架',
      icon: 'flash',
      menu: true,
      perm: 'listing:publish',
    },
  },
]
