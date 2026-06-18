import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'agentos',
    name: 'AgentOSControlCenter',
    component: () => import('@/views/agentos/ControlCenter.vue'),
    meta: { title: 'AgentOS 总控台', icon: 'analytics', menu: true, perm: 'agent:view' },
  },
  {
    path: 'agentos/tasks',
    name: 'AgentOSTaskCenter',
    component: () => import('@/views/agentos/TaskCenter.vue'),
    meta: { title: '任务中心', icon: 'checkmark-circle', menu: true, perm: 'agent:view' },
  },
  {
    path: 'agentos/squads',
    name: 'AgentOSSquads',
    component: () => import('@/views/agentos/AgentSquads.vue'),
    meta: { title: 'Agent 小队', icon: 'cube', menu: true, perm: 'agent:view' },
  },
]
