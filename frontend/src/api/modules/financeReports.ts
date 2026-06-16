import http from '@/api/http'

export function getProfitSummary(params?: { date_from?: string; date_to?: string }) {
  return http.get('/finance/reports/profit-summary', { params })
}

export function getOrderProfit(params?: { date_from?: string; date_to?: string; page?: number; page_size?: number }) {
  return http.get('/finance/reports/order-profit', { params })
}

export function getCostVariance(params?: { date_from?: string; date_to?: string }) {
  return http.get('/finance/reports/cost-variance', { params })
}

export function getNegativeProfit(params?: { date_from?: string; date_to?: string }) {
  return http.get('/finance/reports/negative-profit', { params })
}

export function getCostLayerMix(params?: { date_from?: string; date_to?: string }) {
  return http.get('/finance/reports/cost-layer-mix', { params })
}
