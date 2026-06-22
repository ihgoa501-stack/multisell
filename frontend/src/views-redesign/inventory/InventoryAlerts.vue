<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 class="page-header-title">库存预警</h2>
        <span class="page-header-subtitle">库存低于安全库存的SKU</span>
      </div>
    </div>

    <a-card style="margin-top: 12px;" :bordered="false">
      <a-empty v-if="!data.length && !loading" description="暂无库存预警 — 所有SKU库存充足" />
      <template v-else>
        <a-alert
          v-if="data.length > 0"
          type="warning"
          :message="`${data.length} 个SKU库存不足`"
          style="margin-bottom: 12px;"
          closable
        />
        <a-table
          :columns="columns"
          :data-source="data"
          :loading="loading"
          :pagination="false"
          row-key="id"
          size="middle"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'quantity'">
              <span style="color: var(--ant-color-error); font-weight: bold;">{{ record.quantity }}</span>
            </template>
            <template v-else-if="column.dataIndex === 'action'">
              <a-space>
                <a-button size="small" @click="router.push(`/products/${record.product_id}/inventory`)">去补货</a-button>
                <a-button size="small" @click="router.push(`/products/${record.product_id}`)">商品详情</a-button>
              </a-space>
            </template>
          </template>
        </a-table>
      </template>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import http from '@/api/http'

const router = useRouter()
const loading = ref(false)
const data = ref<any[]>([])

const columns = [
  { title: '商品', dataIndex: 'product_name', key: 'product_name', ellipsis: true },
  { title: 'SKU编码', dataIndex: 'sku_code', key: 'sku_code' },
  { title: '规格', dataIndex: 'spec_desc', key: 'spec_desc', ellipsis: true },
  { title: '当前库存', dataIndex: 'quantity', key: 'quantity', width: 100 },
  { title: '安全库存', dataIndex: 'safety_stock', key: 'safety_stock', width: 100 },
  { title: '仓库', dataIndex: 'warehouse', key: 'warehouse', width: 120 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 200 },
]

onMounted(async () => {
  loading.value = true
  try {
    const res: any = await http.get('/inventory/alerts')
    data.value = res.data || []
  } catch (e: any) {
    message.error('加载预警信息失败')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.page-header {
  padding: 12px 0;
}
.page-header-content {
  display: flex;
  align-items: baseline;
  gap: 12px;
}
.page-header-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}
.page-header-subtitle {
  color: var(--ant-color-text-secondary);
  font-size: 14px;
}
</style>
