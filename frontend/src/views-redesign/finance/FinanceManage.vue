<template>
  <div>
    <!-- Page Header -->
    <div class="page-header">
      <div>
        <h2 style="margin: 0;">财务管理</h2>
        <span style="color: var(--ant-color-text-secondary, rgba(0,0,0,0.45)); font-size: 14px;">财务管理与利润分析</span>
      </div>
      <a-button @click="handleGenerateMock">初始化财务数据</a-button>
    </div>

    <a-tabs v-model:activeKey="activeTab" style="margin-top: 12px;">
      <!-- 利润汇总 -->
      <a-tab-pane key="summary" tab="利润汇总">
        <a-card :bordered="false" v-if="summary">
          <a-row :gutter="12">
            <a-col :span="6">
              <a-statistic title="订单数" :value="summary.order_count ?? 0" />
            </a-col>
            <a-col :span="6">
              <a-statistic title="总收入" :value="summary.total_revenue ?? 0" :precision="2" prefix="&yen;" />
            </a-col>
            <a-col :span="6">
              <a-statistic title="总利润" :precision="2" prefix="&yen;">
                <template #formatter>
                  <span :style="{ color: (summary.total_profit ?? 0) >= 0 ? 'var(--ant-color-success, #52c41a)' : 'var(--ant-color-error, #ff4d4f)', fontWeight: 700 }">
                    {{ (summary.total_profit ?? 0).toFixed(2) }}
                  </span>
                </template>
              </a-statistic>
            </a-col>
            <a-col :span="6">
              <a-statistic title="利润率">
                <template #formatter>
                  <span :style="{ color: (summary.profit_margin ?? 0) >= 0 ? 'var(--ant-color-success, #52c41a)' : 'var(--ant-color-error, #ff4d4f)' }">
                    {{ (summary.profit_margin ?? 0).toFixed(2) }}%
                  </span>
                </template>
              </a-statistic>
            </a-col>
          </a-row>
          <a-descriptions style="margin-top: 16px;" :column="2">
            <a-descriptions-item label="商品成本">&yen;{{ (summary.total_product_cost ?? 0).toFixed(2) }}</a-descriptions-item>
            <a-descriptions-item label="运费">&yen;{{ (summary.total_shipping_fee ?? 0).toFixed(2) }}</a-descriptions-item>
            <a-descriptions-item label="平台费">&yen;{{ (summary.total_platform_fee ?? 0).toFixed(2) }}</a-descriptions-item>
            <a-descriptions-item label="支付手续费">&yen;{{ (summary.total_payment_fee ?? 0).toFixed(2) }}</a-descriptions-item>
          </a-descriptions>
        </a-card>
        <a-empty v-else description="暂无数据，请先生成模拟数据或创建订单" />
      </a-tab-pane>

      <!-- 账户管理 -->
      <a-tab-pane key="accounts" tab="账户管理">
        <a-space style="margin-bottom: 12px;">
          <a-button type="primary" @click="showAccountModal = true">＋ 新建账户</a-button>
        </a-space>
        <a-table
          :columns="acctColumns"
          :data-source="accounts"
          :loading="loading"
          :pagination="false"
          row-key="name"
        />
      </a-tab-pane>

      <!-- 流水记录 -->
      <a-tab-pane key="transactions" tab="财务流水">
        <a-table
          :columns="txnColumns"
          :data-source="transactions"
          :loading="loading"
          :pagination="{ current: txnPagination.page, pageSize: txnPagination.pageSize, total: txnPagination.itemCount }"
          @change="onTxnTableChange"
          row-key="id"
        />
      </a-tab-pane>
    </a-tabs>

    <!-- 账户弹窗 -->
    <a-modal v-model:open="showAccountModal" title="新建财务账户" :footer="null" style="width: 400px;">
      <a-form :model="acctForm" layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="acctForm.name" />
        </a-form-item>
        <a-form-item label="类型">
          <a-select v-model:value="acctForm.account_type" :options="[
            { label: '平台收款', value: 'platform' }, { label: '支付账户', value: 'payment' },
            { label: '银行账户', value: 'bank' }, { label: '现金', value: 'cash' },
          ]" />
        </a-form-item>
        <a-form-item label="币种">
          <a-input v-model:value="acctForm.currency" style="width: 100px;" />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="showAccountModal = false">取消</a-button>
          <a-button type="primary" @click="handleCreateAccount">创建</a-button>
        </a-space>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { financeApi } from '@/api/modules/finance'

const loading = ref(false)
const activeTab = ref('summary')
const summary = ref<any>(null)
const accounts = ref<any[]>([])
const transactions = ref<any[]>([])
const showAccountModal = ref(false)
const acctForm = ref({ name: '', account_type: 'platform', currency: 'CNY' })

const txnPagination = ref({ page: 1, pageSize: 20, itemCount: 0 })

function onTxnTableChange(pagination: any) {
  txnPagination.value.page = pagination.current
  fetchTransactions()
}

const acctColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  {
    title: '类型', dataIndex: 'account_type', key: 'account_type',
    customRender: ({ record }: any) =>
      ({ platform: '平台收款', payment: '支付', bank: '银行', cash: '现金' })[record.account_type] ?? record.account_type,
  },
  {
    title: '平台', dataIndex: 'platform_name', key: 'platform_name',
    customRender: ({ text }: any) => text || '-',
  },
  { title: '币种', dataIndex: 'currency', key: 'currency', width: 70 },
  {
    title: '余额', dataIndex: 'balance', key: 'balance',
    customRender: ({ text }: any) => `¥${(text ?? 0).toFixed(2)}`,
  },
]

const txnColumns = [
  {
    title: '类型', dataIndex: 'transaction_type', key: 'transaction_type',
    customRender: ({ record }: any) =>
      ({ revenue: '收入', cost: '成本', fee: '费用', refund: '退款', transfer: '转账' })[record.transaction_type] ?? record.transaction_type,
  },
  {
    title: '金额', dataIndex: 'amount', key: 'amount',
    customRender: ({ text }: any) => {
      const amt = text ?? 0
      return h('span', { style: `color: ${amt >= 0 ? 'var(--ant-color-success, #52c41a)' : 'var(--ant-color-error, #ff4d4f)'}` }, `¥${amt.toFixed(2)}`)
    },
  },
  { title: '账户', dataIndex: 'account_name', key: 'account_name' },
  {
    title: '描述', dataIndex: 'description', key: 'description',
    customRender: ({ text }: any) => text || '-',
  },
  {
    title: '时间', dataIndex: 'created_at', key: 'created_at',
    customRender: ({ text }: any) => text ? text.slice(0, 19).replace('T', ' ') : '-',
  },
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

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
</style>
