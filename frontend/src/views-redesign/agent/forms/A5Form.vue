<template>
  <a-form>
    <a-form-item label="SKU 编码">
      <a-select v-model:value="form.sku_code" :options="skuOptions" show-search mode="tags" placeholder="搜索或输入 SKU" @change="onSkuChange" />
    </a-form-item>
    <template v-if="!autoMode">
      <a-form-item label="可售库存">
        <a-input-number v-model:value="form.sellable_stock" :min="0" style="width:100%" />
      </a-form-item>
      <a-form-item label="锁定库存">
        <a-input-number v-model:value="form.locked_stock" :min="0" style="width:100%" />
      </a-form-item>
      <a-form-item label="在途库存">
        <a-input-number v-model:value="form.in_transit_stock" :min="0" style="width:100%" />
      </a-form-item>
      <a-form-item label="近 7 天销量">
        <a-input-number v-model:value="form.sales_7d" :min="0" style="width:100%" />
      </a-form-item>
      <a-form-item label="采购提前期（天）">
        <a-input-number v-model:value="form.lead_time_days" :min="1" style="width:100%" />
      </a-form-item>
      <a-form-item label="MOQ（最小起订量）">
        <a-input-number v-model:value="form.moq" :min="0" style="width:100%" />
      </a-form-item>
    </template>
    <a-space>
      <a-button type="primary" @click="run" :loading="loading">执行库存检查</a-button>
      <a-button @click="run(true)" :loading="loading">模拟</a-button>
      <a-checkbox v-model:checked="autoMode">自动读取数据库</a-checkbox>
    </a-space>
  </a-form>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { message } from 'ant-design-vue'

const props = defineProps<{ result: any }>()
const emit = defineEmits(['decision'])
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
