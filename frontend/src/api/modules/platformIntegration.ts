import http from '@/api/http'

export function listAdapters() {
  return http.get('/platform-integrations/adapters')
}

export function listAccounts(params?: { adapter_code?: string; status?: string }) {
  return http.get('/platform-integrations/accounts', { params })
}

export function getAccount(id: number) {
  return http.get(`/platform-integrations/accounts/${id}`)
}

export function createAccount(data: {
  platform_id: number
  adapter_code: string
  account_name: string
  credentials?: Record<string, string>
}) {
  return http.post('/platform-integrations/accounts', data)
}

export function updateAccount(
  id: number,
  data: { account_name?: string; status?: string; credentials?: Record<string, string> },
) {
  return http.put(`/platform-integrations/accounts/${id}`, data)
}

export function testAccount(id: number) {
  return http.post(`/platform-integrations/accounts/${id}/test`)
}

export function listCategoryMappings(params?: { platform_id?: number; adapter_code?: string }) {
  return http.get('/platform-integrations/category-mappings', { params })
}

export function createCategoryMapping(data: {
  platform_id: number
  adapter_code: string
  local_category_id: number
  platform_category_id: string
  platform_category_name?: string
  platform_category_path?: string
}) {
  return http.post('/platform-integrations/category-mappings', data)
}

export function listAttributeMappings(params?: { platform_id?: number; adapter_code?: string }) {
  return http.get('/platform-integrations/attribute-mappings', { params })
}

export function createAttributeMapping(data: {
  platform_id: number
  adapter_code: string
  local_attribute: string
  platform_attribute: string
  default_value?: string
}) {
  return http.post('/platform-integrations/attribute-mappings', data)
}
