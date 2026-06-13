<template>
  <div>
    <n-page-header subtitle="库存低于安全库存的SKU">
      <template #title>⚠️ 库存预警</template>
    </n-page-header>

    <n-card style="margin-top: 12px;" :bordered="false">
      <n-empty v-if="!data.length && !loading" description="暂无库存预警 — 所有SKU库存充足" />
      <template v-else>
        <n-alert v-if="data.length > 0" type="warning" :title="`${data.length} 个SKU库存不足`" style="margin-bottom: 12px;" closable />
        <n-data-table :columns="columns" :data="data" :loading="loading" />
      </template>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NTag, useMessage } from 'naive-ui'
import http from '@/api/http'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const data = ref<any[]>([])

const columns = [
  { title: '商品', key: 'product_name', ellipsis: { tooltip: true } },
  { title: 'SKU编码', key: 'sku_code' },
  { title: '规格', key: 'spec_desc', ellipsis: { tooltip: true } },
  { title: '当前库存', key: 'quantity', width: 100,
    render: (row: any) => h('span', { style: 'color:#d03050; font-weight: bold;' }, row.quantity) },
  { title: '安全库存', key: 'safety_stock', width: 100 },
  { title: '仓库', key: 'warehouse', width: 120 },
  { title: '操作', width: 200, render: (row: any) => {
    return h('span', [
      h(NButton, { size: 'small', onClick: () => router.push(`/products/${row.product_id}/inventory`) }, { default: () => '去补货' }),
      h(NButton, { size: 'small', style: 'margin-left: 8px;', onClick: () => router.push(`/products/${row.product_id}`) }, { default: () => '商品详情' }),
    ])
  }},
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
