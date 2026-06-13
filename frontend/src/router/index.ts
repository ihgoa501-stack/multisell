import { createRouter, createWebHistory } from 'vue-router'
import Layout from '@/components/Layout.vue'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录', noAuth: true },
  },
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/dashboard/Dashboard.vue'), meta: { title: '首页' } },
      { path: 'products', name: 'ProductList', component: () => import('@/views/product/ProductList.vue'), meta: { title: '商品列表' } },
      { path: 'products/create', name: 'ProductCreate', component: () => import('@/views/product/ProductForm.vue'), meta: { title: '新增商品' } },
      { path: 'products/:id/edit', name: 'ProductEdit', component: () => import('@/views/product/ProductForm.vue'), meta: { title: '编辑商品' } },
      { path: 'products/:id', name: 'ProductDetail', component: () => import('@/views/product/ProductDetail.vue'), meta: { title: '商品详情' } },
      { path: 'categories', name: 'CategoryList', component: () => import('@/views/category/CategoryList.vue'), meta: { title: '分类管理' } },
      { path: 'products/:id/skus', name: 'SkuManage', component: () => import('@/views/sku/SkuManage.vue'), meta: { title: 'SKU管理' } },
      { path: 'products/:id/prices', name: 'PriceManage', component: () => import('@/views/price/PriceManage.vue'), meta: { title: '价格管理' } },
      { path: 'products/:id/inventory', name: 'InventoryManage', component: () => import('@/views/inventory/InventoryManage.vue'), meta: { title: '库存管理' } },
      { path: 'suppliers', name: 'SupplierList', component: () => import('@/views/supplier/SupplierList.vue'), meta: { title: '供应商管理' } },
      { path: 'brands', name: 'BrandList', component: () => import('@/views/brand/BrandList.vue'), meta: { title: '品牌管理' } },
      { path: 'operation-logs', name: 'OperationLog', component: () => import('@/views/operation_log/OperationLog.vue'), meta: { title: '操作日志' } },
      { path: 'platforms', name: 'PlatformList', component: () => import('@/views/platform/PlatformList.vue'), meta: { title: '平台管理' } },
      { path: 'listings', name: 'ListingManage', component: () => import('@/views/listing/ListingManage.vue'), meta: { title: '发布管理' } },
      { path: 'inventory/alerts', name: 'InventoryAlerts', component: () => import('@/views/inventory/InventoryAlerts.vue'), meta: { title: '库存预警' } },
    ],
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
