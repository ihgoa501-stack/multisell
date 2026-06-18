/**
 * 财务管理 API 模块
 */
import http from '@/api/http'

export const financeApi = {
  // 账户
  listAccounts() { return http.get('/finance/accounts') },
  createAccount(data: any) { return http.post('/finance/accounts', data) },

  // 流水
  listTransactions(params?: {
    account_id?: number; transaction_type?: string; page?: number; page_size?: number
  }) { return http.get('/finance/transactions', { params }) },
  createTransaction(data: any) { return http.post('/finance/transactions', data) },

  // 利润汇总
  getProfitSummary(params?: {
    period_start?: string; period_end?: string; platform_id?: number
  }) { return http.get('/finance/profit-summary', { params }) },

  // 模拟数据
  generateMock() { return http.post('/finance/mock') },
}

export function rebuildOrderLedger(orderId: number) {
  return http.post(`/finance/orders/${orderId}/ledger/rebuild`)
}

export function getOrderLedger(orderId: number) {
  return http.get(`/finance/orders/${orderId}/ledger`)
}

export function getOrderProfit(orderId: number) {
  return http.get(`/finance/orders/${orderId}/profit`)
}
