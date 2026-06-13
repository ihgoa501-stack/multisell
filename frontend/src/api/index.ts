import http from './http'

// ========== 商品 API ==========
export const productApi = {
  list(params?: any) {
    return http.get('/products', { params })
  },
  getById(id: number) {
    return http.get(`/products/${id}`)
  },
  create(data: any) {
    return http.post('/products', data)
  },
  update(id: number, data: any) {
    return http.put(`/products/${id}`, data)
  },
  delete(id: number) {
    return http.delete(`/products/${id}`)
  },
}

// ========== 分类 API ==========
export const categoryApi = {
  getTree() {
    return http.get('/categories/tree')
  },
  create(data: any) {
    return http.post('/categories', data)
  },
  update(id: number, data: any) {
    return http.put(`/categories/${id}`, data)
  },
  delete(id: number) {
    return http.delete(`/categories/${id}`)
  },
}

// ========== SKU API ==========
export const skuApi = {
  getSpecs(productId: number) {
    return http.get(`/products/${productId}/specs`)
  },
  defineSpecs(productId: number, data: any) {
    return http.post(`/products/${productId}/specs`, data)
  },
  generateSkus(productId: number) {
    return http.post(`/products/${productId}/skus/generate`)
  },
  getSkus(productId: number) {
    return http.get(`/products/${productId}/skus`)
  },
  updateSku(skuId: number, data: any) {
    return http.put(`/skus/${skuId}`, data)
  },
  getSku(skuId: number) {
    return http.get(`/skus/${skuId}`)
  },
}

// ========== 价格 API ==========
export const priceApi = {
  setPrice(data: any) {
    return http.post('/prices', data)
  },
  batchSetPrice(data: any) {
    return http.post('/prices/batch', data)
  },
  getPricesBySku(skuId: number) {
    return http.get(`/skus/${skuId}/prices`)
  },
  getCurrentPrice(skuId: number) {
    return http.get(`/skus/${skuId}/current-price`)
  },
  getPriceHistory(skuId: number) {
    return http.get(`/skus/${skuId}/price-history`)
  },
}

// ========== 库存 API ==========
export const inventoryApi = {
  update(skuId: number, data: any) {
    return http.put(`/inventory/${skuId}`, data)
  },
  get(skuId: number) {
    return http.get(`/inventory/${skuId}`)
  },
  check(data: any) {
    return http.post('/inventory/check', data)
  },
  getLogs(skuId: number) {
    return http.get(`/inventory/${skuId}/logs`)
  },
}

// ========== 供应商 API ==========
export const supplierApi = {
  list(params?: any) {
    return http.get('/suppliers', { params })
  },
  getById(id: number) {
    return http.get(`/suppliers/${id}`)
  },
  create(data: any) {
    return http.post('/suppliers', data)
  },
  update(id: number, data: any) {
    return http.put(`/suppliers/${id}`, data)
  },
  delete(id: number) {
    return http.delete(`/suppliers/${id}`)
  },
  bindProduct(data: any) {
    return http.post('/product-supplier', data)
  },
  getProductSuppliers(productId: number) {
    return http.get(`/products/${productId}/suppliers`)
  },
  unbindProduct(productId: number, supplierId: number) {
    return http.delete('/product-supplier', { params: { product_id: productId, supplier_id: supplierId } })
  },
}

// ========== 文件上传 ==========
export const uploadApi = {
  upload(file: File) {
    const formData = new FormData()
    formData.append('file', file)
    return http.post('/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
}

// ========== 品牌 API ==========
export const brandApi = {
  list(params?: any) {
    return http.get('/brands', { params })
  },
  getAll() {
    return http.get('/brands/all')
  },
  getById(id: number) {
    return http.get(`/brands/${id}`)
  },
  create(data: any) {
    return http.post('/brands', data)
  },
  update(id: number, data: any) {
    return http.put(`/brands/${id}`, data)
  },
  delete(id: number) {
    return http.delete(`/brands/${id}`)
  },
}

// ========== 操作日志 API ==========
export const operationLogApi = {
  list(params?: any) {
    return http.get('/operation-logs', { params })
  },
  getModules() {
    return http.get('/operation-logs/modules')
  },
}

// ========== 平台管理 API ==========
export const platformApi = {
  list() {
    return http.get('/platforms')
  },
  getById(id: number) {
    return http.get(`/platforms/${id}`)
  },
  create(data: any) {
    return http.post('/platforms', data)
  },
  update(id: number, data: any) {
    return http.put(`/platforms/${id}`, data)
  },
  delete(id: number) {
    return http.delete(`/platforms/${id}`)
  },
}

// ========== 发布管理 API ==========
export const listingApi = {
  publish(productId: number, platformId: number) {
    return http.post(`/products/${productId}/publish/${platformId}`)
  },
  getListings(productId: number) {
    return http.get(`/products/${productId}/listings`)
  },
  getAllListings() {
    return http.get('/listings')
  },
}
