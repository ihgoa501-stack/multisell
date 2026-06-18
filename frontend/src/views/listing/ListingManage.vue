<template>
  <div>
    <n-page-header subtitle="商品在各平台的发布状态">
      <template #title>📤 发布管理</template>
    </n-page-header>

    <!-- 全局发布状态 -->
    <n-card title="全局发布概览" style="margin-top: 12px;" :bordered="false">
      <n-data-table :columns="columns" :data="data" :loading="loading" />
    </n-card>

    <!-- 商品详情发布状态 -->
    <n-card v-if="productId" title="单个商品发布状态" style="margin-top: 12px;" :bordered="false">
      <n-form inline>
        <n-form-item label="商品ID">
          <n-input-number v-model:value="productId" :min="1" style="width: 120px;" />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="fetchProductListings">查询</n-button>
        </n-form-item>
      </n-form>
      <n-data-table v-if="productListings.length" :columns="detailColumns" :data="productListings" :bordered="true" />
      <n-empty v-else description="暂无发布记录" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NTag, NSpace, useMessage } from 'naive-ui'
import http from '@/api/http'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const data = ref<any[]>([])
const productId = ref<number>(1)
const productListings = ref<any[]>([])

const columns = [
  { title: '商品', key: 'product_name', ellipsis: { tooltip: true } },
  { title: '平台', key: 'platform_name', width: 100, render: (row: any) => h(NTag, { size: 'small', color: { color: row.platform_code === 'ozon' ? '#005bff' : '#ee4d2d' } }, { default: () => row.platform_name }) },
  { title: '平台商品ID', key: 'platform_product_id', ellipsis: { tooltip: true } },
  { title: '状态', key: 'status', width: 90, render: (row: any) => {
    const statusMap: Record<string, any> = {
      'synced': { type: 'success', text: '已发布' },
      'pending': { type: 'warning', text: '发布中' },
      'failed': { type: 'error', text: '失败' },
      'draft': { type: 'default', text: '草稿' },
    }
    const s = statusMap[row.status] || { type: 'default', text: row.status }
    return h(NTag, { type: s.type, size: 'small' }, { default: () => s.text })
  }},
  { title: '链接', key: 'platform_url', ellipsis: { tooltip: true } },
  { title: '上次同步', key: 'last_sync_at', width: 170 },
  { title: '操作', width: 160, render: (row: any) => h(NSpace, null, { default: () => [
    h(NButton, { size: 'small', type: 'warning', onClick: () => handleRepublish(row.product_id, row.platform_id) }, { default: () => '重新发布' }),
    h(NButton, { size: 'small', onClick: () => router.push(`/products/${row.product_id}`) }, { default: () => '商品' }),
  ]})},
]

const detailColumns = [
  { title: '平台', key: 'platform_name' },
  { title: '平台商品ID', key: 'platform_product_id' },
  { title: '状态', key: 'status', render: (row: any) => h(NTag, { type: row.status === 'synced' ? 'success' : 'warning', size: 'small' }, { default: () => row.status }) },
  { title: '链接', key: 'platform_url', ellipsis: { tooltip: true } },
  { title: '同步时间', key: 'last_sync_at' },
]

async function fetchData() {
  loading.value = true
  try {
    const res: any = await http.get('/listings')
    data.value = res.data || []
  } catch (e: any) { message.error(e.message) }
  finally { loading.value = false }
}

async function fetchProductListings() {
  try {
    const res: any = await http.get(`/products/${productId.value}/listings`)
    productListings.value = res.data || []
  } catch (e: any) { message.error(e.message) }
}

async function handleRepublish(pid: number, platId: number) {
  try {
    await http.post(`/products/${pid}/publish/${platId}`)
    message.success('重新发布成功')
    fetchData()
  } catch (e: any) { message.error(e.message) }
}

onMounted(fetchData)
</script>

<style scoped>
/* 页面容器 */
:deep(.n-page-header) {
  padding-bottom: 16px;
  border-bottom: 1px solid var(--color-neutral-200, #e5e5e5);
  margin-bottom: 16px;
}

/* 卡片样式 */
:deep(.n-card) {
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.n-card:hover) {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

/* 卡片标题 */
:deep(.n-card-header__main) {
  font-weight: 600;
  font-size: 15px;
  color: var(--color-neutral-900, #171717);
}

/* 表格样式 */
:deep(.n-data-table) {
  border-radius: 6px;
}

:deep(.n-data-table-thead) {
  background: var(--color-neutral-50, #f9fafb);
}

:deep(.n-data-table-tr:hover) {
  background: var(--color-neutral-50, #f9fafb);
}

/* 标签样式 */
:deep(.n-tag--small) {
  font-weight: 500;
  border-radius: 4px;
}

/* 表单样式 */
:deep(.n-form--inline) {
  align-items: flex-end;
}

/* 输入框 */
:deep(.n-input-number) {
  border-radius: 4px;
}

/* 按钮样式 */
:deep(.n-button--small-type) {
  font-weight: 500;
  border-radius: 4px;
}

/* 空状态 */
:deep(.n-empty) {
  padding: 40px 0;
}

/* 全局发布概览卡片 */
:deep(.n-card:first-of-type) {
  background: linear-gradient(135deg, #f9fafb 0%, #f0f9ff 100%);
  border: 1px solid var(--color-neutral-200, #e5e5e5);
}

/* 响应式调整 */
@media (max-width: 768px) {
  :deep(.n-form--inline) {
    flex-direction: column;
    align-items: stretch;
  }

  :deep(.n-data-table) {
    font-size: 13px;
  }
}
</style>
