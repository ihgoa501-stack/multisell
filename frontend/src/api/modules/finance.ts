import http from '@/api/http'

export function rebuildOrderLedger(orderId: number) {
  return http.post(`/finance/orders/${orderId}/ledger/rebuild`)
}

export function getOrderLedger(orderId: number) {
  return http.get(`/finance/orders/${orderId}/ledger`)
}

export function getOrderProfit(orderId: number) {
  return http.get(`/finance/orders/${orderId}/profit`)
}
