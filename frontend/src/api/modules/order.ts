/**
 * 订单管理 API 模块
 *
 * 后端接口：
 *   GET    /api/orders?status=xxx&page=1&page_size=20
 *   GET    /api/orders/{id}
 *   POST   /api/orders/{id}/shipping-quote
 *   PUT    /api/orders/{id}/profit-inputs
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
  /** 计算并绑定订单运费快照 */
  bindShippingQuote(id: number | string, data: {
    sku_id: number
    quantity: number
    destination_country: string
    postal_code?: string
    cargo_type?: string
    channel_id?: number | null
  }) {
    return http.post(`/orders/${id}/shipping-quote`, data)
  },
  /** 更新订单利润输入 */
  updateProfitInputs(id: number | string, data: {
    platform_fee?: number
    payment_fee?: number
    other_fee?: number
    product_cost?: number
  }) {
    return http.put(`/orders/${id}/profit-inputs`, data)
  },
}
