/**
 * 订单管理 API 模块
 *
 * 后端接口：
 *   GET    /api/orders?status=xxx&page=1&page_size=20
 *   GET    /api/orders/{id}
 *   PUT    /api/orders/{id}/status   body: { status: "shipped" }
 */

import http from '@/api/http'

export const orderApi = {
  /** 订单列表（分页 + 按状态筛选） */
  list(params?: { status?: string; page?: number; page_size?: number }) {
    return http.get('/orders', { params })
  },
  /** 订单详情 */
  getById(id: number | string) {
    return http.get(`/orders/${id}`)
  },
  /** 更新订单状态（eg. 发货） */
  updateStatus(id: number | string, status: string) {
    return http.put(`/orders/${id}/status`, { status })
  },
}
