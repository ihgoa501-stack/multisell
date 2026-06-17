<template>
  <n-form>
    <n-form-item label="SKU / ASIN">
      <n-input v-model:value="form.sku_code" placeholder="SKU001" />
    </n-form-item>
    <template v-if="!autoMode">
      <n-form-item label="售价">
        <n-input-number v-model:value="form.selling_price" :min="0" style="width:100%" />
      </n-form-item>
      <n-form-item label="成本价">
        <n-input-number v-model:value="form.cost_price" :min="0" style="width:100%" />
      </n-form-item>
    </template>
    <n-form-item label="平台">
      <n-select v-model:value="form.platform" :options="[{label:'Amazon',value:'amazon'},{label:'Shopify',value:'shopify'},{label:'Ozon',value:'ozon'},{label:'Wildberries',value:'wb'}]" />
    </n-form-item>
    <n-form-item label="最低毛利率阈值 %">
      <n-input-number v-model:value="form.min_margin_threshold" :min="0" style="width:100%" />
    </n-form-item>
    <n-form-item label="折扣列表">
      <n-space vertical style="width:100%">
        <n-space v-for="(d, i) in form.active_discounts" :key="i">
          <n-select v-model:value="d.type" :options="discountTypes" style="width:120px" />
          <n-input-number v-model:value="d.value" :min="0" style="width:100px" />
          <n-button size="tiny" quaternary type="error" @click="form.active_discounts.splice(i,1)">×</n-button>
        </n-space>
        <n-button size="small" @click="form.active_discounts.push({type:'coupon',value:10})">＋ 添加折扣</n-button>
      </n-space>
    </n-form-item>
    <n-space>
      <n-button type="primary" @click="run" :loading="loading">检查折扣风险</n-button>
      <n-checkbox v-model:value="autoMode">自动读取数据库</n-checkbox>
    </n-space>
  </n-form>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { useMessage } from 'naive-ui'

const emit = defineEmits(['decision'])
const message = useMessage()
const loading = ref(false)
const autoMode = ref(false)

const discountTypes = [
  { label: '优惠券', value: 'coupon' },
  { label: '促销', value: 'promotion' },
  { label: '会员折扣', value: 'member_discount' },
  { label: '固定金额', value: 'fixed' },
]

const form = reactive({
  sku_code: '',
  selling_price: 100,
  cost_price: 60,
  platform: 'amazon',
  min_margin_threshold: 10,
  active_discounts: [] as {type:string;value:number}[],
})

async function run() {
  loading.value = true
  try {
    const ctx: any = { ...form }
    const res: any = await agentApi.decide('G3', { decision_point: 'discount_check', context: ctx })
    emit('decision', res?.data)
  } catch (e: any) {
    message.error(e?.response?.data?.message || '执行失败')
  }
  loading.value = false
}
</script>
