/**
 * 批量导入 API 模块
 *
 * 后端接口：
 *   GET    /api/import/templates/{import_type}  — 下载导入模板
 *   POST   /api/import/preview                  — 上传并预览导入文件
 *   POST   /api/import/commit/{batch_id}        — 提交导入批次
 *   GET    /api/import/batches                  — 导入批次列表
 *   GET    /api/import/batches/{batch_id}       — 导入批次详情
 *   GET    /api/import/batches/{batch_id}/errors — 下载错误报告
 */

import http from '@/api/http'

export const importBatchApi = {
  /** 下载导入模板 */
  downloadTemplate(importType: string) {
    return http.get(`/import/templates/${importType}`, { responseType: 'blob' })
  },

  /** 上传并预览导入文件 */
  uploadPreview(file: File, importType: string) {
    const form = new FormData()
    form.append('file', file)
    return http.post(`/import/preview?type=${importType}`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  /** 提交导入批次 */
  commitBatch(batchId: number) {
    return http.post(`/import/commit/${batchId}`)
  },

  /** 导入批次列表 */
  listBatches(params?: { page?: number; page_size?: number }) {
    return http.get('/import/batches', { params })
  },

  /** 导入批次详情 */
  getBatchDetail(batchId: number) {
    return http.get(`/import/batches/${batchId}`)
  },

  /** 下载错误报告 */
  downloadErrors(batchId: number) {
    return http.get(`/import/batches/${batchId}/errors`, { responseType: 'blob' })
  },
}
