import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'image-gen-canvas',
    name: 'ImageGenCanvas',
    component: () => import('@/views/image_gen_canvas/CanvasEditor.vue'),
    meta: {
      title: '无限画布',
      icon: 'grid',
      menu: true,
      perm: 'image_gen:generate',
    },
  },
]
