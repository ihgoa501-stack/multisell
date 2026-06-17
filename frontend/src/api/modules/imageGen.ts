import http from '@/api/http'

export interface GenerateImageRequest {
  product_id: number
  prompt: string
  negative_prompt?: string
  style?: string
  size?: string
  count?: number
}

export interface GenerateImageResponse {
  job_id: number
  images: string[]
  status: string
  error?: string
}

export interface BatchGenerateRequest {
  product_ids: number[]
  prompt: string
  negative_prompt?: string
  style?: string
  size?: string
  count?: number
}

export interface BatchGenerateItem {
  product_id: number
  product_name?: string
  job_id: number
  status: string
  images: string[]
  error?: string
}

export interface BatchGenerateResponse {
  batch_id: string
  total: number
  success: number
  failed: number
  results: BatchGenerateItem[]
}

export interface SaveImageRequest {
  product_id: number
  image_url: string
  set_as_main?: boolean
}

export interface RemoveBgResponse {
  url: string
}

export interface GenHistoryItem {
  id: number
  product_id: number
  prompt: string
  style: string
  status: string
  image_urls: string[]
  created_at: string
  product_name?: string
}

export interface GenHistoryResponse {
  items: GenHistoryItem[]
  total: number
}

// ====== Prompt 模板 ======

export interface PromptTemplateItem {
  id: number
  name: string
  description?: string
  prompt: string
  negative_prompt?: string
  style: string
  size: string
  platform_code?: string
  is_shared: boolean
  usage_count: number
  created_by?: number
  created_at: string
  updated_at: string
}

export interface PromptTemplateCreateRequest {
  name: string
  description?: string
  prompt: string
  negative_prompt?: string
  style?: string
  size?: string
  platform_code?: string
  is_shared?: boolean
}

export interface PromptTemplateUpdateRequest {
  name?: string
  description?: string
  prompt?: string
  negative_prompt?: string
  style?: string
  size?: string
  platform_code?: string
  is_shared?: boolean
}

export interface TemplateListResponse {
  items: PromptTemplateItem[]
  total: number
}

export const imageGenApi = {
  generate(data: GenerateImageRequest) {
    return http.post<GenerateImageResponse>('/image-gen/generate', data)
  },

  batchGenerate(data: BatchGenerateRequest) {
    return http.post<BatchGenerateResponse>('/image-gen/batch-generate', data)
  },

  save(data: SaveImageRequest) {
    return http.post('/image-gen/save', data)
  },

  removeBg(imageUrl: string) {
    return http.post<RemoveBgResponse>('/image-gen/remove-bg', { image_url: imageUrl })
  },

  history(params: { product_id?: number; page?: number; page_size?: number }) {
    return http.get<GenHistoryResponse>('/image-gen/history', { params })
  },

  // === Prompt 模板 ===
  listTemplates(params?: { platform_code?: string; page?: number; page_size?: number }) {
    return http.get<TemplateListResponse>('/image-gen/templates', { params })
  },

  createTemplate(data: PromptTemplateCreateRequest) {
    return http.post('/image-gen/templates', data)
  },

  updateTemplate(id: number, data: PromptTemplateUpdateRequest) {
    return http.put(`/image-gen/templates/${id}`, data)
  },

  deleteTemplate(id: number) {
    return http.delete(`/image-gen/templates/${id}`)
  },
}
