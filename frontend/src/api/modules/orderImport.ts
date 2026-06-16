import http from '@/api/http'

export function importOrders(file: File, adapterCode: string = 'csv_order') {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('adapter_code', adapterCode)
  return http.post('/order-imports/csv', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export function listOrderImports(adapterCode?: string) {
  const params: Record<string, string> = {}
  if (adapterCode) params.adapter_code = adapterCode
  return http.get('/order-imports', { params })
}

export function getOrderImport(batchId: number) {
  return http.get(`/order-imports/${batchId}`)
}

export function listOrderImportItems(batchId: number) {
  return http.get(`/order-imports/${batchId}/items`)
}

export function processOrderImportChain(batchId: number) {
  return http.post(`/order-imports/${batchId}/process-chain`)
}

export function getOrderImportChainSummary(batchId: number) {
  return http.get(`/order-imports/${batchId}/chain-summary`)
}
