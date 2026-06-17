import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'agents',
    name: 'AgentList',
    component: () => import('@/views/agent/AgentList.vue'),
    meta: { title: 'AI 助手', icon: 'cube', menu: true, perm: 'agent:view' },
  },
  {
    path: 'agents/:agentId',
    name: 'AgentDetail',
    component: () => import('@/views/agent/AgentDetail.vue'),
    meta: { title: 'Agent 详情', menu: false },
  },
  {
    path: 'agents/rules',
    name: 'AgentRules',
    component: () => import('@/views/agent/AgentRules.vue'),
    meta: { title: '个人规则', icon: 'settings', menu: false },
  },
]
