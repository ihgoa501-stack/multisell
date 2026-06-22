<template>
  <div>
    <!-- Page Header -->
    <div style="margin-bottom: 16px;">
      <a-typography-title :level="4" style="margin-bottom: 4px;">发布管理</a-typography-title>
      <a-typography-text type="secondary">商品在各平台的发布状态</a-typography-text>
    </div>

    <!-- 全局发布状态 -->
    <a-card title="全局发布概览" style="margin-top: 12px;" :bordered="false">
      <a-table
        :columns="columns"
        :data-source="data"
        :loading="loading"
        :pagination="false"
        row-key="product_id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'platform_name'">
            <a-tag :color="record.platform_code === 'ozon' ? 'blue' : 'volcano'">
              {{ record.platform_name }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="listingStatusColor[record.status] || 'default'">
              {{ listingStatusText[record.status] || record.status }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button size="small" type="primary" danger @click="handleRepublish(record.product_id, record.platform_id)">
                重新发布
              </a-button>
              <a-button size="small" @click="router.push(`/products/${record.product_id}`)">
                商品
              </a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 商品详情发布状态 -->
    <a-card v-if="productId" title="单个商品发布状态" style="margin-top: 12px;" :bordered="false">
      <a-form layout="inline">
        <a-form-item label="商品ID">
          <a-input-number v-model:value="productId" :min="1" style="width: 120px;" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="fetchProductListings">查询</a-button>
        </a-form-item>
      </a-form>
      <a-table
        v-if="productListings.length"
        :columns="detailColumns"
        :data-source="productListings"
        :bordered="true"
        :pagination="false"
        row-key="platform_name"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'detail_status'">
            <a-tag :color="record.status === 'synced' ? 'green' : 'orange'">
              {{ record.status }}
            </a-tag>
          </template>
        </template>
      </a-table>
      <a-empty v-else description="暂无发布记录" />
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { useRouter } from 'vue-router'
import http from '@/api/http'

const router = useRouter()
const loading = ref(false)
const data = ref<any[]>([])
const productId = ref<number>(1)
const productListings = ref<any[]>([])

const listingStatusColor: Record<string, string> = {
  synced: 'green',
  pending: 'orange',
  failed: 'red',
  draft: 'default',
}

const listingStatusText: Record<string, string> = {
  synced: '已发布',
  pending: '发布中',
  failed: '失败',
  draft: '草稿',
}

const columns = [
  { title: '商品', dataIndex: 'product_name', key: 'product_name', ellipsis: true },
  { title: '平台', dataIndex: 'platform_name', key: 'platform_name', width: 100 },
  { title: '平台商品ID', dataIndex: 'platform_product_id', key: 'platform_product_id', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 90 },
  { title: '链接', dataIndex: 'platform_url', key: 'platform_url', ellipsis: true },
  { title: '上次同步', dataIndex: 'last_sync_at', key: 'last_sync_at', width: 170 },
  { title: '操作', key: 'actions', width: 160 },
]

const detailColumns = [
  { title: '平台', dataIndex: 'platform_name', key: 'platform_name' },
  { title: '平台商品ID', dataIndex: 'platform_product_id', key: 'platform_product_id' },
  { title: '状态', dataIndex: 'status', key: 'detail_status' },
  { title: '链接', dataIndex: 'platform_url', key: 'platform_url', ellipsis: true },
  { title: '同步时间', dataIndex: 'last_sync_at', key: 'last_sync_at' },
]

async function fetchData() {
  loading.value = true
  try {
    const res: any = await http.get('/listings')
    data.value = res.data || []
  } catch (e: any) {
    message.error(e.message)
  } finally {
    loading.value = false
  }
}

async function fetchProductListings() {
  try {
    const res: any = await http.get(`/products/${productId.value}/listings`)
    productListings.value = res.data || []
  } catch (e: any) {
    message.error(e.message)
  }
}

async function handleRepublish(pid: number, platId: number) {
  try {
    await http.post(`/products/${pid}/publish/${platId}`)
    message.success('重新发布成功')
    fetchData()
  } catch (e: any) {
    message.error(e.message)
  }
}

onMounted(fetchData)
</script>

<style scoped>
:deep(.ant-card) {
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.ant-card:hover) {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

:deep(.ant-card-head-title) {
  font-weight: 600;
  font-size: 15px;
}

:deep(.ant-table) {
  border-radius: 6px;
}

:deep(.ant-tag) {
  font-weight: 500;
  border-radius: 4px;
}

:deep(.ant-empty) {
  padding: 40px 0;
}

:deep(.ant-card:first-of-type) {
  background: linear-gradient(135deg, #f9fafb 0%, #f0f9ff 100%);
  border: 1px solid #e5e5e5;
}

@media (max-width: 768px) {
  :deep(.ant-form-inline) {
    flex-direction: column;
    align-items: stretch;
  }

  :deep(.ant-table) {
    font-size: 13px;
  }
}
</style>
