import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import Layout from '@/components/Layout.vue'

// ========== 基础路由 ==========
const baseRoutes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录', noAuth: true, menu: false },
  },
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/Dashboard.vue'),
        meta: { title: '首页', icon: 'home', menu: true },
      },
      {
        path: 'products',
        name: 'ProductList',
        component: () => import('@/views/product/ProductList.vue'),
        meta: { title: '商品列表', icon: 'list', menu: true },
      },
      {
        path: 'products/create',
        name: 'ProductCreate',
        component: () => import('@/views/product/ProductForm.vue'),
        meta: { title: '新增商品', menu: false },
      },
      {
        path: 'products/:id/edit',
        name: 'ProductEdit',
        component: () => import('@/views/product/ProductForm.vue'),
        meta: { title: '编辑商品', menu: false },
      },
      {
        path: 'products/:id',
        name: 'ProductDetail',
        component: () => import('@/views/product/ProductDetail.vue'),
        meta: { title: '商品详情', menu: false },
      },
      {
        path: 'categories',
        name: 'CategoryList',
        component: () => import('@/views/category/CategoryList.vue'),
        meta: { title: '分类管理', icon: 'layers', menu: true },
      },
      {
        path: 'products/:id/skus',
        name: 'SkuManage',
        component: () => import('@/views/sku/SkuManage.vue'),
        meta: { title: 'SKU管理', menu: false },
      },
      {
        path: 'products/:id/prices',
        name: 'PriceManage',
        component: () => import('@/views/price/PriceManage.vue'),
        meta: { title: '价格管理', menu: false },
      },
      {
        path: 'products/:id/inventory',
        name: 'InventoryManage',
        component: () => import('@/views/inventory/InventoryManage.vue'),
        meta: { title: '库存管理', menu: false },
      },
      {
        path: 'suppliers',
        name: 'SupplierList',
        component: () => import('@/views/supplier/SupplierList.vue'),
        meta: { title: '供应商管理', icon: 'people', menu: true },
      },
      {
        path: 'brands',
        name: 'BrandList',
        component: () => import('@/views/brand/BrandList.vue'),
        meta: { title: '品牌管理', icon: 'tag', menu: true },
      },
      {
        path: 'operation-logs',
        name: 'OperationLog',
        component: () => import('@/views/operation_log/OperationLog.vue'),
        meta: { title: '操作日志', icon: 'doc-text', menu: true },
      },
      {
        path: 'platforms',
        name: 'PlatformList',
        component: () => import('@/views/platform/PlatformList.vue'),
        meta: { title: '平台管理', icon: 'globe', menu: true },
      },
      {
        path: 'listings',
        name: 'ListingManage',
        component: () => import('@/views/listing/ListingManage.vue'),
        meta: { title: '发布管理', icon: 'archive', menu: true },
      },
      {
        path: 'inventory/alerts',
        name: 'InventoryAlerts',
        component: () => import('@/views/inventory/InventoryAlerts.vue'),
        meta: { title: '库存预警', icon: 'warning', menu: true },
      },
    ],
  },
]

// ========== 合并 router/modules/*.ts 中新模块的路由 ==========
// 每个 AI 在 modules/ 下创建文件（如 modules/order.ts），导出 route 配置。
// 路由格式见 modules/order.ts.example
// ===============================================
const _routeModules = import.meta.glob('./modules/*.ts', { eager: true })
const _allChildren: RouteRecordRaw[] = [...(baseRoutes[1] as any).children]

for (const mod of Object.values(_routeModules)) {
  const _mod = mod as Record<string, any>
  // 支持 export const routes = [...]
  if (Array.isArray(_mod.routes)) {
    _allChildren.push(..._mod.routes)
  }
  // 支持 export default [...]
  if (Array.isArray(_mod.default)) {
    _allChildren.push(..._mod.default)
  }
}

// 重建完整的 routes 数组
const routes: RouteRecordRaw[] = [
  baseRoutes[0], // /login
  {
    ...baseRoutes[1],
    children: _allChildren,
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.noAuth) {
    next()
  } else if (!token) {
    next('/login')
  } else {
    next()
  }
})

export default router
