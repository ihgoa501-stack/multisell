import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'image-gen',
    name: 'ImageGen',
    component: () => import('@/views/image_gen/ImageGenWorkbench.vue'),
    meta: {
      title: 'AI 生图',
      icon: 'image',
      menu: true,
      perm: 'image_gen:generate',
    },
  },
]
