<template>
  <div>
    <n-page-header subtitle="财务管理与利润分析">
      <template #title>💰 财务管理</template>
      <template #extra>
        <n-button @click="handleGenerateMock">🎲 初始化财务数据</n-button>
      </template>
    </n-page-header>

    <n-tabs type="line" default-value="summary" style="margin-top: 12px;">
      <!-- 利润汇总 -->
      <n-tab-pane name="summary" tab="利润汇总">
        <n-card :bordered="false" v-if="summary">
          <n-grid :cols="4" :x-gap="12">
            <n-grid-item>
              <n-statistic label="订单数" :value="summary.order_count ?? 0" />
            </n-grid-item>
            <n-grid-item>
              <n-statistic label="总收入" :value="`¥${(summary.total_revenue ?? 0).toFixed(2)}`" />
            </n-grid-item>
            <n-grid-item>
              <n-statistic label="总利润">
                <span :style="{ color: (summary.total_profit ?? 0) >= 0 ? '#18a058' : '#d03050', fontWeight: 700 }">
                  ¥{{ (summary.total_profit ?? 0).toFixed(2) }}
                </span>
              </n-statistic>
            </n-grid-item>
            <n-grid-item>
              <n-statistic label="利润率">
                <span :style="{ color: (summary.profit_margin ?? 0) >= 0 ? '#18a058' : '#d03050' }">
                  {{ (summary.profit_margin ?? 0).toFixed(2) }}%
                </span>
              </n-statistic>
            </n-grid-item>
          </n-grid>
          <n-descriptions label-placement="left" style="margin-top: 16px;" :column="2">
            <n-descriptions-item label="商品成本">¥{{ (summary.total_product_cost ?? 0).toFixed(2) }}</n-descriptions-item>
            <n-descriptions-item label="运费">¥{{ (summary.total_shipping_fee ?? 0).toFixed(2) }}</n-descriptions-item>
            <n-descriptions-item label="平台费">¥{{ (summary.total_platform_fee ?? 0).toFixed(2) }}</n-descriptions-item>
            <n-descriptions-item label="支付手续费">¥{{ (summary.total_payment_fee ?? 0).toFixed(2) }}</n-descriptions-item>
          </n-descriptions>
        </n-card>
        <n-empty v-else description="暂无数据，请先生成模拟数据或创建订单" />
      </n-tab-pane>

      <!-- 账户管理 -->
      <n-tab-pane name="accounts" tab="账户管理">
        <n-space style="margin-bottom: 12px;">
          <n-button type="primary" @click="showAccountModal = true">＋ 新建账户</n-button>
        </n-space>
        <n-data-table :columns="acctColumns" :data="accounts" :loading="loading" />
      </n-tab-pane>

      <!-- 流水记录 -->
      <n-tab-pane name="transactions" tab="财务流水">
        <n-data-table :columns="txnColumns" :data="transactions" :loading="loading" :pagination="txnPagination"
          @update:page="onTxnPageChange" />
      </n-tab-pane>
    </n-tabs>

    <!-- 账户弹窗 -->
    <n-modal v-model:show="showAccountModal" title="新建财务账户" preset="card" style="width: 400px;">
      <n-form :model="acctForm" label-placement="left" label-width="100px">
        <n-form-item label="名称" required><n-input v-model:value="acctForm.name" /></n-form-item>
        <n-form-item label="类型">
          <n-select v-model:value="acctForm.account_type" :options="[
            { label: '平台收款', value: 'platform' }, { label: '支付账户', value: 'payment' },
            { label: '银行账户', value: 'bank' }, { label: '现金', value: 'cash' },
          ]" />
        </n-form-item>
        <n-form-item label="币种"><n-input v-model:value="acctForm.currency" style="width: 100px;" /></n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAccountModal = false">取消</n-button>
          <n-button type="primary" @click="handleCreateAccount">创建</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { financeApi } from '@/api/modules/finance'

const message = useMessage()
const loading = ref(false)
const summary = ref<any>(null)
const accounts = ref<any[]>([])
const transactions = ref<any[]>([])
const showAccountModal = ref(false)
const acctForm = ref({ name: '', account_type: 'platform', currency: 'CNY' })

const txnPagination = ref({ page: 1, pageSize: 20, itemCount: 0,
  onChange: (p: number) => { txnPagination.value.page = p; fetchTransactions() } })
function onTxnPageChange(p: number) { txnPagination.value.page = p; fetchTransactions() }

const acctColumns = [
  { title: '名称', key: 'name' },
  { title: '类型', key: 'account_type', render: (r: any) => ({ platform: '平台收款', payment: '支付', bank: '银行', cash: '现金' })[r.account_type] ?? r.account_type },
  { title: '平台', key: 'platform_name', render: (r: any) => r.platform_name || '-' },
  { title: '币种', key: 'currency', width: 70 },
  { title: '余额', key: 'balance', render: (r: any) => `¥${(r.balance ?? 0).toFixed(2)}` },
]

const txnColumns = [
  { title: '类型', key: 'transaction_type', render: (r: any) => ({ revenue: '收入', cost: '成本', fee: '费用', refund: '退款', transfer: '转账' })[r.transaction_type] ?? r.transaction_type },
  { title: '金额', key: 'amount', render: (r: any) => {
    const amt = r.amount ?? 0
    return h('span', { style: `color: ${amt >= 0 ? '#18a058' : '#d03050'}` }, `¥${amt.toFixed(2)}`)
  }},
  { title: '账户', key: 'account_name' },
  { title: '描述', key: 'description', render: (r: any) => r.description || '-' },
  { title: '时间', key: 'created_at', render: (r: any) => r.created_at ? r.created_at.slice(0, 19).replace('T', ' ') : '-' },
]

async function fetchSummary() {
  try { const res = await financeApi.getProfitSummary(); summary.value = res.data?.data ?? null } catch {}
}
async function fetchAccounts() {
  try { const res = await financeApi.listAccounts(); accounts.value = res.data?.data ?? [] } catch {}
}
async function fetchTransactions() {
  loading.value = true
  try {
    const res = await financeApi.listTransactions({ page: txnPagination.value.page, page_size: txnPagination.value.pageSize })
    const body = res.data
    transactions.value = body?.data?.records ?? body?.records ?? []
    txnPagination.value.itemCount = body?.data?.total ?? body?.total ?? 0
  } catch { message.error('加载失败') }
  finally { loading.value = false }
}
async function handleCreateAccount() {
  try { await financeApi.createAccount(acctForm.value); showAccountModal.value = false; fetchAccounts(); message.success('创建成功') }
  catch { message.error('创建失败') }
}
async function handleGenerateMock() {
  try { await financeApi.generateMock(); message.success('财务数据已初始化'); fetchAccounts() }
  catch { message.error('生成失败') }
}

onMounted(async () => { await fetchSummary(); await fetchAccounts(); await fetchTransactions() })
</script>
