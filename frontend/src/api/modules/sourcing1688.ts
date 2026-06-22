import api from '@/api'
import type { AxiosRequestConfig } from 'axios'

/** 采集 payload */
export interface CollectPayload {
  url: string
  title?: string
  price?: number
  moq?: number
  supplier?: string
  shop_url?: string
  shop_location?: string
  images?: string[]
  attributes?: any[]
  skuVariants?: any[]
  description?: string
  length_cm?: number
  width_cm?: number
  height_cm?: number
  weight_g?: number
}

/** 候选商品 VO */
export interface Sourcing1688ProductVO {
  id: number
  source_url: string
  title?: string
  price?: number
  moq?: number
  supplier_name?: string
  shop_url?: string
  shop_location?: string
  images?: string[]
  attributes?: any[]
  sku_variants?: any[]
  description?: string
  package_length_cm?: number
  package_width_cm?: number
  package_height_cm?: number
  package_weight_kg?: number
  status: string
  product_id?: number
  supplier_id?: number
  collected_by?: string
  imported_by?: string
  imported_at?: string
  created_at?: string
  updated_at?: string
}

/** 导入 payload */
export interface ImportPayload {
  category_id?: number
  brand_id?: number
  cargo_type?: string
  unit?: string
}

/** 分页结果 */
interface PageResult<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

// ===== API 调用 =====

export function collectProduct(data: CollectPayload, config?: AxiosRequestConfig) {
  return api.post<{ data: Sourcing1688ProductVO }>('/1688-collect/products', data, config)
}

export function listProducts(params: {
  status?: string
  keyword?: string
  page?: number
  page_size?: number
}, config?: AxiosRequestConfig) {
  return api.get<PageResult<Sourcing1688ProductVO>>('/1688-collect/products', { params, ...config })
}

export function getProduct(id: number, config?: AxiosRequestConfig) {
  return api.get<{ data: Sourcing1688ProductVO }>(`/1688-collect/products/${id}`, config)
}

export function importProduct(id: number, data: ImportPayload, config?: AxiosRequestConfig) {
  return api.post<{ data: Sourcing1688ProductVO }>(`/1688-collect/products/${id}/import`, data, config)
}

export function rejectProduct(id: number, config?: AxiosRequestConfig) {
  return api.post<{ data: Sourcing1688ProductVO }>(`/1688-collect/products/${id}/reject`, {}, config)
}
