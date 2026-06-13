/**
 * Excel 导入导出 API
 *
 * 自动合并到 apiModules（详见 @/api/index.ts）。
 * 使用：import { apiModules } from '@/api' → apiModules.excelApi.xxx()
 */

import http from '@/api/http'

export const excelApi = {
  /** 导出商品列表为 Excel */
  exportProducts(params?: any) {
    return http.get('/products/export', { params, responseType: 'blob' })
  },
  /** 从 Excel 导入商品 */
  importProducts(file: File) {
    const formData = new FormData()
    formData.append('file', file)
    return http.post('/products/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
  /** 下载导入模板 */
  downloadTemplate() {
    return http.get('/products/export-template', { responseType: 'blob' })
  },
}
