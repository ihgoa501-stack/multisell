import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'agents',
    name: 'AgentList',
    component: () => import('@/views/agent/AgentList.vue'),
    meta: { title: 'AI 助手', icon: 'cube', menu: true, perm: 'agent:view' },
  },
  {
    path: 'agents/actions',
    name: 'AgentActions',
    component: () => import('@/views/agent/AgentActions.vue'),
    meta: { title: '待执行操作', icon: 'checkmark-circle', menu: true, perm: 'agent:view' },
  },
  {
    path: 'agents/dashboard',
    name: 'AgentDashboard',
    component: () => import('@/views/agent/Dashboard.vue'),
    meta: { title: '运营驾驶舱', icon: 'analytics', menu: true, perm: 'agent:view' },
  },
  {
    path: 'agents/llm-settings',
    name: 'LlmSettings',
    component: () => import('@/views/agent/LlmSettings.vue'),
    meta: { title: 'LLM 配置', icon: 'settings', menu: true, perm: 'agent:view' },
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
  {
    path: 'agents/entropy',
    name: 'EntropyCockpit',
    component: () => import('@/views/agent/EntropyCockpit.vue'),
    meta: { title: '熵管理', icon: 'analytics', menu: true, perm: 'agent:view' },
  },
]
