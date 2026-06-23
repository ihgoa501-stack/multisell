import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: '/redesign/login',
    name: 'RedesignLogin',
    component: () => import('@/views-redesign/Login.vue'),
    meta: { standalone: true, noAuth: true, title: '登录' },
  },
  {
    path: '/redesign',
    name: 'RedesignLayout',
    component: () => import('@/views-redesign/layout/RedesignLayout.vue'),
    redirect: '/redesign/dashboard',
    meta: { standalone: true },
    children: [
      // ── 指挥中心 ──
      { path: 'dashboard', name: 'RDashboard', component: () => import('@/views-redesign/dashboard/Dashboard.vue'), meta: { menu: false, title: '工作台' } },
      { path: 'agentos', name: 'RAgentOS', component: () => import('@/views-redesign/agentos/ControlCenter.vue'), meta: { menu: false, title: 'AgentOS 总控台' } },
      { path: 'agentos/work-items', name: 'RWorkItems', component: () => import('@/views-redesign/agentos/WorkItems.vue'), meta: { menu: false, title: '任务队列' } },
      { path: 'agentos/squads', name: 'RSquads', component: () => import('@/views-redesign/agentos/Squads.vue'), meta: { menu: false, title: 'Agent 团队' } },
      { path: 'agentos/autonomy', name: 'RAutonomy', component: () => import('@/views-redesign/agentos/AutonomyManagement.vue'), meta: { menu: false, title: '自治管理' } },
      { path: 'agentos/agents/:agentId', name: 'RAgentOSAgentDetail', component: () => import('@/views-redesign/agentos/AgentDetail.vue'), meta: { menu: false, title: 'Agent 详情' } },

      // ── Agent 系统 ──
      { path: 'agents', name: 'RAgentList', component: () => import('@/views-redesign/agent/AgentList.vue'), meta: { menu: false, title: 'Agent 列表' } },
      { path: 'agents/dashboard', name: 'RAgentDashboard', component: () => import('@/views-redesign/agent/Dashboard.vue'), meta: { menu: false, title: 'Agent 看板' } },
      { path: 'agents/actions', name: 'RAgentActions', component: () => import('@/views-redesign/agent/AgentActions.vue'), meta: { menu: false, title: 'Agent 操作' } },
      { path: 'agents/rules', name: 'RAgentRules', component: () => import('@/views-redesign/agent/AgentRules.vue'), meta: { menu: false, title: 'Agent 规则' } },
      { path: 'agents/entropy', name: 'REntropyCockpit', component: () => import('@/views-redesign/agent/EntropyCockpit.vue'), meta: { menu: false, title: '熵控制台' } },
      { path: 'agents/llm-settings', name: 'RLlmSettings', component: () => import('@/views-redesign/agent/LlmSettings.vue'), meta: { menu: false, title: 'LLM 设置' } },
      { path: 'agents/:agentId', name: 'RAgentDetail', component: () => import('@/views-redesign/agent/AgentDetail.vue'), meta: { menu: false, title: 'Agent 详情' } },

      // ── 商品中心 ──
      { path: 'products', name: 'RProductList', component: () => import('@/views-redesign/product/ProductList.vue'), meta: { menu: false, title: '商品管理' } },
      { path: 'products/create', name: 'RProductCreate', component: () => import('@/views-redesign/product/ProductForm.vue'), meta: { menu: false, title: '新增商品' } },
      { path: 'products/:id/edit', name: 'RProductEdit', component: () => import('@/views-redesign/product/ProductForm.vue'), meta: { menu: false, title: '编辑商品' } },
      { path: 'products/:id', name: 'RProductDetail', component: () => import('@/views-redesign/product/ProductDetail.vue'), meta: { menu: false, title: '商品详情' } },
      { path: 'products/:id/skus', name: 'RSkuManage', component: () => import('@/views-redesign/sku/SkuManage.vue'), meta: { menu: false, title: 'SKU 管理' } },
      { path: 'products/:id/prices', name: 'RPriceManage', component: () => import('@/views-redesign/price/PriceManage.vue'), meta: { menu: false, title: '价格管理' } },
      { path: 'products/:id/inventory', name: 'RInventoryManage', component: () => import('@/views-redesign/inventory/InventoryManage.vue'), meta: { menu: false, title: '库存管理' } },
      { path: 'categories', name: 'RCategoryList', component: () => import('@/views-redesign/category/CategoryList.vue'), meta: { menu: false, title: '分类管理' } },
      { path: 'brands', name: 'RBrandList', component: () => import('@/views-redesign/brand/BrandList.vue'), meta: { menu: false, title: '品牌管理' } },
      { path: 'inventory/alerts', name: 'RInventoryAlerts', component: () => import('@/views-redesign/inventory/InventoryAlerts.vue'), meta: { menu: false, title: '库存预警' } },

      // ── 供应链 ──
      { path: 'suppliers', name: 'RSupplierList', component: () => import('@/views-redesign/supplier/SupplierList.vue'), meta: { menu: false, title: '供应商管理' } },
      { path: '1688-sourcing', name: 'RSourcing1688', component: () => import('@/views-redesign/sourcing1688/Sourcing1688List.vue'), meta: { menu: false, title: '1688 选品' } },
      { path: 'shipping/manage', name: 'RShippingManage', component: () => import('@/views-redesign/shipping/ShippingManage.vue'), meta: { menu: false, title: '物流管理' } },
      { path: 'shipping/calculator', name: 'RShippingCalc', component: () => import('@/views-redesign/shipping/ShippingCalculator.vue'), meta: { menu: false, title: '运费计算' } },
      { path: 'shipping/bill-reconciliation', name: 'RShippingRecon', component: () => import('@/views-redesign/shipping/ShippingBillReconciliation.vue'), meta: { menu: false, title: '运费对账' } },
      { path: 'allocation', name: 'RAllocation', component: () => import('@/views-redesign/allocation/AllocationManage.vue'), meta: { menu: false, title: '分摊管理' } },

      // ── 交易平台 ──
      { path: 'platforms', name: 'RPlatformList', component: () => import('@/views-redesign/platform/PlatformList.vue'), meta: { menu: false, title: '平台管理' } },
      { path: 'platform-integrations', name: 'RPlatformIntegrations', component: () => import('@/views-redesign/platform/PlatformIntegrationConsole.vue'), meta: { menu: false, title: '平台集成' } },
      { path: 'listings', name: 'RListingManage', component: () => import('@/views-redesign/listing/ListingManage.vue'), meta: { menu: false, title: '上架管理' } },
      { path: 'listing-tasks', name: 'RListingTasks', component: () => import('@/views-redesign/listing/ListingTaskQueue.vue'), meta: { menu: false, title: '上架任务' } },
      { path: 'listings/ai-workbench', name: 'RAiListing', component: () => import('@/views-redesign/listing_task/AiListingWorkbench.vue'), meta: { menu: false, title: 'AI 上架台' } },
      { path: 'orders', name: 'ROrderList', component: () => import('@/views-redesign/order/OrderList.vue'), meta: { menu: false, title: '订单管理' } },
      { path: 'orders/:id', name: 'ROrderDetail', component: () => import('@/views-redesign/order/OrderDetail.vue'), meta: { menu: false, title: '订单详情' } },
      { path: 'order-import', name: 'ROrderImport', component: () => import('@/views-redesign/order_import/OrderImport.vue'), meta: { menu: false, title: '订单导入' } },

      // ── 财务 ──
      { path: 'settlements', name: 'RSettlementList', component: () => import('@/views-redesign/settlement/SettlementList.vue'), meta: { menu: false, title: '结算列表' } },
      { path: 'settlements/:id', name: 'RSettlementDetail', component: () => import('@/views-redesign/settlement/SettlementDetail.vue'), meta: { menu: false, title: '结算详情' } },
      { path: 'finance', name: 'RFinance', component: () => import('@/views-redesign/finance/FinanceManage.vue'), meta: { menu: false, title: '财务管理' } },
      { path: 'decisions/prelisting', name: 'RDecisionWB', component: () => import('@/views-redesign/decision/DecisionWorkbench.vue'), meta: { menu: false, title: '决策工作台' } },
      { path: 'decisions/prelisting/batch', name: 'RBatchDecision', component: () => import('@/views-redesign/decision/BatchPreListingDecision.vue'), meta: { menu: false, title: '批量决策' } },

      // ── 系统 ──
      { path: 'notifications', name: 'RNotification', component: () => import('@/views-redesign/notification/NotificationCenter.vue'), meta: { menu: false, title: '通知中心' } },
      { path: 'users', name: 'RUserMgmt', component: () => import('@/views-redesign/rbac/UserManagement.vue'), meta: { menu: false, title: '用户管理' } },
      { path: 'roles', name: 'RRoleMgmt', component: () => import('@/views-redesign/rbac/RoleManagement.vue'), meta: { menu: false, title: '角色管理' } },
      { path: 'operation-logs', name: 'ROperationLog', component: () => import('@/views-redesign/operation_log/OperationLog.vue'), meta: { menu: false, title: '操作日志' } },
      { path: 'reports', name: 'RReport', component: () => import('@/views-redesign/report/index.vue'), meta: { menu: false, title: '报表' } },
      { path: 'exceptions', name: 'RExceptions', component: () => import('@/views-redesign/exceptions/ExceptionWorkbench.vue'), meta: { menu: false, title: '异常工作台' } },
      { path: 'image-gen', name: 'RImageGen', component: () => import('@/views-redesign/image_gen/ImageGenWorkbench.vue'), meta: { menu: false, title: '图片生成' } },
      { path: 'image-gen-canvas', name: 'RImageGenCanvas', component: () => import('@/views-redesign/image_gen_canvas/CanvasEditor.vue'), meta: { menu: false, title: '图片编辑器' } },
    ],
  },
]
