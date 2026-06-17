/**
 * 结算管理 API 模块
 *
 * 后端接口：
 *   POST   /api/settlements                    — 导入结算单
 *   GET    /api/settlements                    — 结算单列表
 *   GET    /api/settlements/{id}               — 结算单详情
 *   PUT    /api/settlements/{id}               — 更新结算单
 *   DELETE /api/settlements/{id}               — 删除结算单
 *   POST   /api/settlements/{id}/items         — 添加结算明细
 *   GET    /api/settlements/{id}/items         — 结算明细列表
 *   POST   /api/settlements/{id}/reconcile     — 执行对账
 *   PUT    /api/settlements/items/{id}/reconciliation — 更新对账状态
 *   POST   /api/settlements/mock               — 生成模拟数据
 */

import http from '@/api/http'

export const settlementApi = {
  /** 导入结算单 */
  import(data: {
    platform_id: number
    settlement_no: string
    period_start?: string
    period_end?: string
    currency?: string
    total_revenue?: number
    total_fee?: number
    total_refund?: number
    total_net?: number
    raw_data?: Record<string, any>
  }) {
    return http.post('/settlements', data)
  },

  /** 结算单列表 */
  list(params?: {
    platform_id?: number
    status?: string
    keyword?: string
    page?: number
    page_size?: number
  }) {
    return http.get('/settlements', { params })
  },

  /** 结算单详情 */
  getById(id: number) {
    return http.get(`/settlements/${id}`)
  },

  /** 更新结算单 */
  update(id: number, data: {
    status?: string
    total_revenue?: number
    total_fee?: number
    total_refund?: number
    total_net?: number
  }) {
    return http.put(`/settlements/${id}`, data)
  },

  /** 删除结算单 */
  delete(id: number) {
    return http.delete(`/settlements/${id}`)
  },

  /** 添加结算明细 */
  addItem(settlementId: number, data: {
    transaction_type: string
    transaction_id?: string
    order_no?: string
    order_id?: number
    sku_id?: number
    amount?: number
    fee?: number
    net?: number
    quantity?: number
    occurred_at?: string
  }) {
    return http.post(`/settlements/${settlementId}/items`, data)
  },

  /** 结算明细列表 */
  listItems(settlementId: number, params?: {
    reconciliation_status?: string
    transaction_type?: string
    page?: number
    page_size?: number
  }) {
    return http.get(`/settlements/${settlementId}/items`, { params })
  },

  /** 执行对账 */
  reconcile(settlementId: number, data?: {
    auto_match?: boolean
    strategy?: string
  }) {
    return http.post(`/settlements/${settlementId}/reconcile`, data ?? {})
  },

  /** 更新明细对账状态 */
  updateReconciliation(itemId: number, data: {
    status: string
    note?: string
  }) {
    return http.put(`/settlements/items/${itemId}/reconciliation`, data)
  },

  /** 生成模拟结算数据 */
  generateMock(params: {
    platform_id: number
    count?: number
  }) {
    return http.post('/settlements/mock', null, { params })
  },
}
