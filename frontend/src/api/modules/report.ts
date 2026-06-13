/**
 * 报表 API 模块
 *
 * 使用方式：
 *   import { apiModules } from '@/api'
 *   const data = await apiModules.reportApi.getProductStats()
 */

import http from '@/api/http'

export const reportApi = {
  /** 商品分布数据（饼图） */
  getProductStats(params?: any) {
    return http.get('/reports/product-stats', { params })
  },
  /** 平台发布状态数据（柱状图） */
  getPlatformStats(params?: any) {
    return http.get('/reports/platform-stats', { params })
  },
  /** 总览数据（库存健康度等） */
  getDashboardStats(params?: any) {
    return http.get('/dashboard/stats', { params })
  },
}
