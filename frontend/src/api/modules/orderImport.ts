/**
 * 订单导入 API 模块
 */
import http from '@/api/http'

export const orderImportApi = {
  /** 导入订单数据 */
  import(data: {
    source_type: string
    platform_id?: number
    orders: Array<{
      platform_order_no: string
      order_date?: string
      recipient_name?: string
      recipient_phone?: string
      shipping_address?: string
      items: Array<{ sku_code?: string; product_name?: string; quantity: number; unit_price: number }>
      total_amount?: number
      shipping_fee?: number
      platform_fee?: number
    }>
  }) {
    return http.post('/order-import', data)
  },

  /** 上传CSV导入 */
  importCsv(sourceType: string, file: File) {
    const form = new FormData()
    form.append('file', file)
    return http.post(`/order-import/csv?source_type=${sourceType}`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  /** 导入记录列表 */
  list(params?: { source_type?: string; page?: number; page_size?: number }) {
    return http.get('/order-imports', { params })
  },

  /** 生成模拟订单 */
  generateMock(params: { platform_id: number; count?: number }) {
    return http.post('/order-import/mock', null, { params })
  },
}
