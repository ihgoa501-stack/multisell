import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'platform-integrations',
    name: 'PlatformIntegrationConsole',
    component: () => import('@/views/platform/PlatformIntegrationConsole.vue'),
    meta: {
      title: '平台集成',
      icon: 'globe',
      menu: true,
      perm: 'platform_integration:view',
    },
  },
]
