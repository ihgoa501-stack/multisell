import http from '@/api/http'

export interface CanvasLayer {
  id: string
  type: 'image' | 'text' | 'mask'
  fabric_json: Record<string, any>
}

export interface CanvasSaveRequest {
  product_id: number
  name?: string
  layers: CanvasLayer[]
}

export interface CanvasItem {
  id: number
  product_id: number
  name: string
  layers: Record<string, any>[]
  thumbnail?: string
  created_by: number
  created_at: string
  updated_at: string
}

export interface CanvasListResponse {
  items: CanvasItem[]
  total: number
}

export interface InpaintRequest {
  image_url: string
  mask_base64: string
  prompt: string
  negative_prompt?: string
}

export interface OutpaintRequest {
  image_url: string
  direction: string
  prompt: string
  expand_ratio?: number
}

export interface VideoGenRequest {
  prompt: string
  image_url?: string
}

export interface SlideshowRequest {
  image_urls: string[]
  duration_per_frame?: number
  transition?: string
  resolution?: string
}

export const imageGenCanvasApi = {
  saveCanvas(data: CanvasSaveRequest) {
    return http.post('/image-gen/canvas', data)
  },

  loadCanvas(canvasId: number) {
    return http.get<CanvasItem>(`/image-gen/canvas/${canvasId}`)
  },

  listCanvases(productId: number, params?: { page?: number; page_size?: number }) {
    return http.get<CanvasListResponse>('/image-gen/canvas', {
      params: { product_id: productId, ...params },
    })
  },

  deleteCanvas(canvasId: number) {
    return http.delete(`/image-gen/canvas/${canvasId}`)
  },

  inpaint(data: InpaintRequest) {
    return http.post<{ image_url: string }>('/image-gen/inpaint', data)
  },

  outpaint(data: OutpaintRequest) {
    return http.post<{ image_url: string }>('/image-gen/outpaint', data)
  },

  generateVideo(data: VideoGenRequest) {
    return http.post<{ job_id: string; status: string }>('/image-gen/video', data)
  },

  getVideoStatus(jobId: string) {
    return http.get<{ job_id: string; status: string; video_url?: string }>(
      `/image-gen/video/status/${jobId}`
    )
  },

  createSlideshow(data: SlideshowRequest) {
    return http.post<{ video_url: string; duration: number }>('/image-gen/video/slideshow', data)
  },
}
