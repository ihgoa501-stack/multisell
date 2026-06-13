/**
 * API 模块示例 — 新建模块时复制这个文件按格式写
 *
 * 规则：
 * 1. export 你的 API 对象
 * 2. 前端页面通过 import { apiModules } from '@/api' 拿到
 *    const data = await apiModules.myModuleApi.list(params)
 *
 * 3. import http from '@/api/http' 发请求
 *    已有模块（productApi / categoryApi 等）不受影响，可直接 import
 */

import http from '@/api/http'

export const orderApi = {
  list(params?: any) {
    return http.get('/orders', { params })
  },
  getById(id: number) {
    return http.get(`/orders/${id}`)
  },
  create(data: any) {
    return http.post('/orders', data)
  },
  updateStatus(id: number, status: string) {
    return http.put(`/orders/${id}/status`, { status })
  },
}
