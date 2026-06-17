<template>
  <n-form>
    <n-form-item label="SKU 编码">
      <n-select v-model:value="form.sku_code" :options="skuOptions" filterable tag placeholder="搜索或输入 SKU" @update:value="onSkuChange" />
    </n-form-item>
    <template v-if="!autoMode">
      <n-form-item label="可售库存">
        <n-input-number v-model:value="form.sellable_stock" :min="0" style="width:100%" />
      </n-form-item>
      <n-form-item label="锁定库存">
        <n-input-number v-model:value="form.locked_stock" :min="0" style="width:100%" />
      </n-form-item>
      <n-form-item label="在途库存">
        <n-input-number v-model:value="form.in_transit_stock" :min="0" style="width:100%" />
      </n-form-item>
      <n-form-item label="近 7 天销量">
        <n-input-number v-model:value="form.sales_7d" :min="0" style="width:100%" />
      </n-form-item>
      <n-form-item label="采购提前期（天）">
        <n-input-number v-model:value="form.lead_time_days" :min="1" style="width:100%" />
      </n-form-item>
      <n-form-item label="MOQ（最小起订量）">
        <n-input-number v-model:value="form.moq" :min="0" style="width:100%" />
      </n-form-item>
    </template>
    <n-space>
      <n-button type="primary" @click="run" :loading="loading">执行库存检查</n-button>
      <n-button @click="run(true)" :loading="loading">模拟</n-button>
      <n-checkbox v-model:value="autoMode">自动读取数据库</n-checkbox>
    </n-space>
  </n-form>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { useMessage } from 'naive-ui'

const props = defineProps<{ result: any }>()
const emit = defineEmits(['decision'])
const message = useMessage()
const loading = ref(false)
const autoMode = ref(true)
const skuOptions = ref<any[]>([])

const form = reactive({
  sku_code: '',
  sellable_stock: 0,
  locked_stock: 0,
  in_transit_stock: 0,
  sales_7d: 0,
  sales_14d: 0,
  sales_30d: 0,
  lead_time_days: 20,
  moq: 100,
  safety_stock_days: 14,
})

async function onSkuChange(val: string) {
  if (!val) return
  // Auto-fill with demo data
  form.sellable_stock = 100
  form.sales_7d = 21
}

async function run(dryRun = false) {
  loading.value = true
  try {
    const ctx: any = { ...form }
    if (autoMode.value && form.sku_code) {
      // 只传 sku_code，后端自动补齐
      const res: any = await agentApi.decide('A5', { decision_point: 'stock_alert', context: { sku_code: form.sku_code }, dry_run: dryRun })
      emit('decision', res?.data)
    } else {
      const res: any = await agentApi.decide('A5', { decision_point: 'stock_alert', context: ctx, dry_run: dryRun })
      emit('decision', res?.data)
    }
  } catch (e: any) {
    message.error(e?.response?.data?.message || '执行失败')
  }
  loading.value = false
}
</script>
