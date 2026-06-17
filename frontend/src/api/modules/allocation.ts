/**
 * 库存分配 API 模块
 */
import http from '@/api/http'

export const allocationApi = {
  // 仓库
  listWarehouses() { return http.get('/warehouses') },
  createWarehouse(data: any) { return http.post('/warehouses', data) },
  updateWarehouse(id: number, data: any) { return http.put(`/warehouses/${id}`, data) },
  deleteWarehouse(id: number) { return http.delete(`/warehouses/${id}`) },
  generateMock() { return http.post('/warehouses/mock') },

  // 分配规则
  listRules() { return http.get('/allocation-rules') },
  createRule(data: any) { return http.post('/allocation-rules', data) },
  deleteRule(id: number) { return http.delete(`/allocation-rules/${id}`) },

  // 库存分配
  getWarehouseInventory(skuId: number) { return http.get(`/inventory/warehouse/${skuId}`) },
  allocate(data: { sku_id: number; warehouse_id: number; quantity: number }) { return http.post('/inventory/allocate', data) },
  autoAllocate(skuId: number) { return http.post(`/inventory/auto-allocate/${skuId}`) },
}
