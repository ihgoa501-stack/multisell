import http from '../http'

export const shippingApi = {
  // Provider
  listProviders() {
    return http.get('/shipping/providers')
  },
  createProvider(data: any) {
    return http.post('/shipping/providers', data)
  },
  updateProvider(id: number, data: any) {
    return http.put(`/shipping/providers/${id}`, data)
  },
  deleteProvider(id: number) {
    return http.delete(`/shipping/providers/${id}`)
  },

  // Channel
  listChannels(providerId?: number) {
    const params: any = {}
    if (providerId !== undefined) params.provider_id = providerId
    return http.get('/shipping/channels', { params })
  },
  createChannel(data: any) {
    return http.post('/shipping/channels', data)
  },
  updateChannel(id: number, data: any) {
    return http.put(`/shipping/channels/${id}`, data)
  },
  deleteChannel(id: number) {
    return http.delete(`/shipping/channels/${id}`)
  },

  // Zone
  listZones(channelId: number) {
    return http.get(`/shipping/channels/${channelId}/zones`)
  },
  createZone(channelId: number, data: any) {
    return http.post(`/shipping/channels/${channelId}/zones`, data)
  },
  deleteZone(zoneId: number) {
    return http.delete(`/shipping/zones/${zoneId}`)
  },

  // Rule
  listRules(channelId: number) {
    return http.get(`/shipping/channels/${channelId}/rules`)
  },
  createRule(channelId: number, data: any) {
    return http.post(`/shipping/channels/${channelId}/rules`, data)
  },
  updateRule(ruleId: number, data: any) {
    return http.put(`/shipping/rules/${ruleId}`, data)
  },
  deleteRule(ruleId: number) {
    return http.delete(`/shipping/rules/${ruleId}`)
  },
  importRules(file: File) {
    const formData = new FormData()
    formData.append('file', file)
    return http.post('/shipping/import-rules', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  // Calculate
  calculate(data: any) {
    return http.post('/shipping/calculate', data)
  },

  // ===== Bill Import & Reconciliation =====
  importBills(file: File) {
    const formData = new FormData()
    formData.append('file', file)
    return http.post('/shipping/bills/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
  listBillBatches(status?: string) {
    const params: any = {}
    if (status) params.status = status
    return http.get('/shipping/bills', { params })
  },
  getBillBatch(batchId: number) {
    return http.get(`/shipping/bills/${batchId}`)
  },
  listBillItems(batchId: number, status?: string) {
    const params: any = {}
    if (status) params.status = status
    return http.get(`/shipping/bills/${batchId}/items`, { params })
  },
  reconcileBatch(batchId: number) {
    return http.post(`/shipping/bills/${batchId}/reconcile`)
  },
  resolveBillItem(itemId: number, note: string) {
    return http.post(`/shipping/bills/items/${itemId}/resolve`, { note })
  },
  getReconciliationSummary() {
    return http.get('/shipping/reconciliation/summary')
  },
}
