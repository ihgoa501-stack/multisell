<template>
  <div class="settlement-import">
    <n-card title="平台结算导入" :bordered="false" class="mb-4">
      <n-upload
        :default-upload="false"
        accept=".csv"
        @change="handleFileChange"
      >
        <n-upload-dragger>
          <div style="padding: 40px 0">
            <n-icon size="48" color="#18a058">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                <polyline points="17 8 12 3 7 8"/>
                <line x1="12" y1="3" x2="12" y2="15"/>
              </svg>
            </n-icon>
            <p style="margin-top: 12px; font-size: 14px; color: #666;">
              点击或拖拽 CSV 文件到此区域上传
            </p>
            <p style="font-size: 12px; color: #999; margin-top: 4px;">
              格式：<code>platform,store_name,platform_order_no,order_no,transaction_type,currency,amount,settled_at,description</code>
            </p>
          </div>
        </n-upload-dragger>
      </n-upload>
      <n-alert type="info" :show-icon="false" style="margin-top: 12px;">
        交易类型：sale / platform_fee / payment_fee / refund / adjustment / payout / tax / other
      </n-alert>
    </n-card>

    <n-card title="结算批次" :bordered="false">
      <template #header-extra>
        <n-space>
          <n-button size="small" @click="refreshBatches">刷新</n-button>
          <n-button size="small" @click="showUnmatched = true">查看未匹配 ({{ unmatchedCount }})</n-button>
        </n-space>
      </template>

      <n-data-table
        :columns="batchColumns"
        :data="batches"
        :loading="loading"
        :pagination="{ pageSize: 20 }"
        striped
      />

      <!-- 批次详情弹窗 -->
      <n-modal
        v-model:show="showDetail"
        :title="`批次详情 #${currentBatch?.id}`"
        style="width: 90%; max-width: 1200px;"
        preset="card"
        :segmented="{ content: true }"
      >
        <n-space vertical>
          <n-space>
            <n-select
              v-model:value="filterMatchStatus"
              placeholder="筛选匹配状态"
              :options="matchStatusOptions"
              clearable
              style="width: 200px"
            />
            <n-tag v-if="currentBatch" type="info">
              已匹配 {{ currentBatch.matched_count }} / 未匹配 {{ currentBatch.unmatched_count }}
            </n-tag>
          </n-space>

          <n-data-table
            :columns="itemColumns"
            :data="filteredItems"
            :loading="itemsLoading"
            :pagination="{ pageSize: 50 }"
            striped
            :row-props="itemRowProps"
          />
        </n-space>
      </n-modal>

      <!-- 未匹配列表弹窗 -->
      <n-modal
        v-model:show="showUnmatched"
        title="未匹配结算行"
        style="width: 90%; max-width: 1000px;"
        preset="card"
        :segmented="{ content: true }"
      >
        <n-data-table
          :columns="unmatchedColumns"
          :data="unmatchedItems"
          :loading="unmatchedLoading"
          :pagination="{ pageSize: 50 }"
          striped
        />
      </n-modal>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useMessage } from 'naive-ui'
import {
  importSettlement,
  getSettlementBatches,
  getSettlementBatch,
  getSettlementItems,
  getUnmatchedSettlements,
  type SettlementBatch,
  type SettlementItem,
} from '@/api/modules/settlement'

const message = useMessage()

// ── State ────────────────────────────────────────────────────────────────

const batches = ref<SettlementBatch[]>([])
const loading = ref(false)

const showDetail = ref(false)
const currentBatch = ref<SettlementBatch | null>(null)
const items = ref<SettlementItem[]>([])
const itemsLoading = ref(false)
const filterMatchStatus = ref<string | null>(null)

const showUnmatched = ref(false)
const unmatchedItems = ref<SettlementItem[]>([])
const unmatchedLoading = ref(false)

// ── Computed ─────────────────────────────────────────────────────────────

const filteredItems = computed(() => {
  if (!filterMatchStatus.value) return items.value
  return items.value.filter((it: SettlementItem) => it.match_status === filterMatchStatus.value)
})

const unmatchedCount = computed(() => {
  return batches.value.reduce((sum, b) => sum + (b.unmatched_count || 0), 0)
})

const matchStatusOptions = [
  { label: '已匹配', value: 'matched' },
  { label: '未匹配', value: 'unmatched' },
]

const txTypeMap: Record<string, { label: string; type: string }> = {
  sale: { label: '销售', type: 'success' },
  platform_fee: { label: '平台费', type: 'warning' },
  payment_fee: { label: '支付费', type: 'warning' },
  refund: { label: '退款', type: 'error' },
  adjustment: { label: '调整', type: 'info' },
  payout: { label: '付款', type: 'primary' },
  tax: { label: '税费', type: 'warning' },
  other: { label: '其他', type: 'default' },
}

// ── Table Columns ────────────────────────────────────────────────────────

const batchColumns = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '平台', key: 'platform_name', width: 100 },
  { title: '文件名', key: 'filename', ellipsis: { tooltip: true } },
  { title: '行数', key: 'row_count', width: 70 },
  { title: '已匹配', key: 'matched_count', width: 80 },
  { title: '未匹配', key: 'unmatched_count', width: 80 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建人', key: 'created_by', width: 100 },
  { title: '创建时间', key: 'created_at', width: 180,
    render: (row: any) => row.created_at ? row.created_at.slice(0, 19).replace('T', ' ') : ''
  },
  {
    title: '操作', key: 'actions', width: 80, fixed: 'right',
    render: (row: any) => h('n-button', { size: 'small', onClick: () => viewDetail(row) }, '详情')
  },
]

const itemColumns = [
  { title: '行号', key: 'row_number', width: 60 },
  { title: '类型', key: 'transaction_type', width: 100,
    render: (row: any) => {
      const meta = txTypeMap[row.transaction_type] || { label: row.transaction_type, type: 'default' }
      return h('n-tag', { size: 'small', type: meta.type }, meta.label)
    }
  },
  { title: '平台', key: 'platform', width: 80 },
  { title: '店铺', key: 'store_name', width: 100, ellipsis: { tooltip: true } },
  { title: '平台订单号', key: 'platform_order_no', width: 140, ellipsis: { tooltip: true } },
  { title: '订单号', key: 'order_no', width: 140, ellipsis: { tooltip: true } },
  { title: '币种', key: 'currency', width: 60 },
  { title: '金额', key: 'amount', width: 100,
    render: (row: any) => {
      const color = row.amount < 0 ? 'red' : row.amount > 0 ? 'green' : undefined
      return h('span', { style: color ? `color:${color}` : undefined }, row.amount?.toFixed(2))
    }
  },
  { title: '结算日期', key: 'settled_at', width: 120,
    render: (row: any) => row.settled_at ? row.settled_at.slice(0, 10) : ''
  },
  { title: '匹配状态', key: 'match_status', width: 100,
    render: (row: any) => {
      const label = row.match_status === 'matched' ? '已匹配' : '未匹配'
      const type = row.match_status === 'matched' ? 'success' : 'warning'
      return h('n-tag', { size: 'small', type }, label)
    }
  },
  { title: '说明', key: 'description', width: 150, ellipsis: { tooltip: true } },
]

const unmatchedColumns = itemColumns.slice() // same as item columns
</script>

<style scoped>
.mb-4 { margin-bottom: 16px; }
</style>
