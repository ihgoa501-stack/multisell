import http from '@/api/http'

export interface ListingTaskCreate {
  name: string
  product_ids: number[]
  platform_ids: number[]
}

export interface PlatformItem {
  id: number
  name: string
  code: string
  status: number
}

export function fetchPlatforms() {
  return http.get('/platforms')
}

export function createListingTask(data: ListingTaskCreate) {
  return http.post('/listing-tasks', data)
}

export function fetchListingTasks(page = 1, page_size = 20) {
  return http.get('/listing-tasks', { params: { page, page_size } })
}

export function fetchListingTaskDetail(taskId: number) {
  return http.get(`/listing-tasks/${taskId}`)
}

export function executeListingTask(taskId: number) {
  return http.post(`/listing-tasks/${taskId}/execute`)
}

export function retryListingItem(taskId: number, itemId: number) {
  return http.post(`/listing-tasks/${taskId}/items/${itemId}/retry`)
}

export function deleteListingTask(taskId: number) {
  return http.delete(`/listing-tasks/${taskId}`)
}
