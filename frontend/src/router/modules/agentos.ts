import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'agentos',
    name: 'AgentOS',
    redirect: '/agentos/control-center',
    meta: {
      title: 'AgentOS',
      icon: 'analytics',
      menu: true,
      perm: 'agentos:view',
    },
    children: [
      {
        path: 'control-center',
        name: 'AgentOSControlCenter',
        component: () => import('@/views/agentos/ControlCenter.vue'),
        meta: {
          title: '总控台',
          icon: 'analytics',
          menu: true,
          perm: 'agentos:view',
        },
      },
      {
        path: 'work-items',
        name: 'AgentOSWorkItems',
        component: () => import('@/views/agentos/WorkItems.vue'),
        meta: {
          title: '任务中心',
          icon: 'list',
          menu: true,
          perm: 'agentos:view',
        },
      },
      {
        path: 'squads',
        name: 'AgentOSSquads',
        component: () => import('@/views/agentos/Squads.vue'),
        meta: {
          title: 'Agent 团队',
          icon: 'people',
          menu: true,
          perm: 'agentos:view',
        },
      },
    ],
  },
]
