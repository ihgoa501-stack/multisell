<template>
  <div>
    <h3 style="margin-bottom: 16px;">📦 运费计算器</h3>
    <n-card>
      <div style="margin-bottom: 16px;">
        <n-radio-group v-model:value="mode" name="shipping-calculate-mode">
          <n-radio-button value="manual">手动输入</n-radio-button>
          <n-radio-button value="sku">按 SKU 计算</n-radio-button>
        </n-radio-group>
      </div>
      <n-alert
        v-if="sourceProduct.id || sourceProduct.name"
        type="info"
        style="margin-bottom: 12px;"
        :show-icon="false"
      >
        来源商品：{{ sourceProduct.name || `ID ${sourceProduct.id}` }}
      </n-alert>
      <n-grid :cols="24" :x-gap="12">
        <template v-if="mode === 'manual'">
          <n-form-item-gi :span="6" label="包装长">
            <n-input-number v-model:value="form.length_cm" :min="0.01" :precision="2" style="width: 100%;">
              <template #suffix>cm</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="6" label="包装宽">
            <n-input-number v-model:value="form.width_cm" :min="0.01" :precision="2" style="width: 100%;">
              <template #suffix>cm</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="6" label="包装高">
            <n-input-number v-model:value="form.height_cm" :min="0.01" :precision="2" style="width: 100%;">
              <template #suffix>cm</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="6" label="包装重量">
            <n-input-number v-model:value="form.weight_kg" :min="0.01" :precision="3" style="width: 100%;">
              <template #suffix>kg</template>
            </n-input-number>
          </n-form-item-gi>
        </template>
        <n-form-item-gi v-if="mode === 'sku'" :span="6" label="SKU ID">
          <n-input-number v-model:value="form.sku_id" placeholder="输入SKU ID" :min="1" />
        </n-form-item-gi>
        <n-form-item-gi :span="4" label="数量">
          <n-input-number v-model:value="form.quantity" :min="1" :default-value="1" />
        </n-form-item-gi>
        <n-form-item-gi :span="5" label="目的地国家">
          <n-input
            v-model:value="form.destination_country"
            placeholder="如 US, DE"
            maxlength="10"
            @blur="normalizeCountry"
            @input="normalizeCountry"
          />
        </n-form-item-gi>
        <n-form-item-gi :span="5" label="货品类型">
          <n-select v-model:value="form.cargo_type" :options="cargoOptions" />
        </n-form-item-gi>
        <n-form-item-gi :span="4" label="邮编(可选)">
          <n-input v-model:value="form.postal_code" placeholder="可选" />
        </n-form-item-gi>
      </n-grid>
      <n-button type="primary" @click="handleCalculate" :loading="loading" :disabled="!canCalculate">
        计算运费
      </n-button>
    </n-card>

    <n-card v-if="result" style="margin-top: 16px;">
      <template #header>
        📊 计算结果 — {{ result.mode === 'manual' ? '手动包裹' : `SKU ${result.sku_id}` }} × {{ result.quantity }} → {{ result.destination_country }}
      </template>
      <n-descriptions label-placement="left" bordered style="margin-bottom: 12px;">
        <n-descriptions-item label="计算模式">{{ result.mode === 'manual' ? '手动输入' : 'SKU' }}</n-descriptions-item>
        <n-descriptions-item label="目的地国家">{{ result.destination_country }}</n-descriptions-item>
        <n-descriptions-item label="数量">{{ result.quantity }}</n-descriptions-item>
        <n-descriptions-item label="包装来源">{{ packageSourceLabel(result.package?.source) }}</n-descriptions-item>
        <n-descriptions-item label="包装尺寸">{{ result.package?.length_cm }}×{{ result.package?.width_cm }}×{{ result.package?.height_cm }} cm</n-descriptions-item>
        <n-descriptions-item label="包装重量">{{ result.package?.weight_kg }} kg</n-descriptions-item>
      </n-descriptions>

      <n-data-table
        v-if="result.results && result.results.length > 0"
        :columns="tableColumns"
        :data="result.results"
        :bordered="true"
        :single-line="false"
        size="small"
      />
      <n-empty v-else description="没有匹配的物流渠道，请检查目的地国家、货品类型或报价规则。" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import { shippingApi } from '@/api/modules/shipping'

const message = useMessage()
const route = useRoute()
const loading = ref(false)
const result = ref<any>(null)
const mode = ref<'manual' | 'sku'>('manual')
const sourceProduct = ref<{ id?: string; name?: string }>({})

const form = reactive({
  sku_id: null as number | null,
  quantity: 1,
  destination_country: '',
  postal_code: '',
  cargo_type: 'normal',
  length_cm: null as number | null,
  width_cm: null as number | null,
  height_cm: null as number | null,
  weight_kg: null as number | null,
})

const cargoOptions = [
  { label: '普通', value: 'normal' },
  { label: '带电池', value: 'battery' },
  { label: '液体', value: 'liquid' },
  { label: '敏感品', value: 'sensitive' },
]

const tableColumns = [
  { title: '供应商', key: 'provider_name', width: 120 },
  { title: '渠道', key: 'channel_name', width: 120 },
  { title: '币种', key: 'currency', width: 60 },
  { title: '实际重(kg)', key: 'actual_weight_kg', width: 100 },
  { title: '体积重(kg)', key: 'volumetric_weight_kg', width: 100 },
  { title: '计费重(kg)', key: 'chargeable_weight_kg', width: 100 },
  { title: '基础费', key: 'base_shipping_fee', width: 80 },
  { title: '附加费', key: 'surcharge_fee', width: 80 },
  { title: '燃油费', key: 'fuel_surcharge_fee', width: 80 },
  { title: '总运费', key: 'total_shipping_fee', width: 90, sortable: true },
  {
    title: '时效(天)', key: 'estimated_delivery_min', width: 80,
    render: (row: any) => row.estimated_delivery_min != null ? `${row.estimated_delivery_min}-${row.estimated_delivery_max}` : '-',
  },
  { title: '说明', key: 'calculation_detail', ellipsis: { tooltip: true } },
]

const canCalculate = computed(() => {
  if (!form.destination_country || !form.quantity) return false
  if (mode.value === 'sku') return !!form.sku_id
  return !!form.length_cm && !!form.width_cm && !!form.height_cm && !!form.weight_kg
})

function packageSourceLabel(source?: string) {
  const map: Record<string, string> = {
    manual: '手动输入',
    sku: 'SKU包装',
    product: '商品包装',
  }
  return map[source || ''] || source || '-'
}

function normalizeCountry() {
  form.destination_country = (form.destination_country || '').trim().toUpperCase()
}

function numberFromQuery(value: unknown): number | null {
  const raw = Array.isArray(value) ? value[0] : value
  if (raw == null || raw === '') return null
  const parsed = Number(raw)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null
}

onMounted(() => {
  const q = route.query
  if (q.sku_id) {
    mode.value = 'sku'
    const sid = numberFromQuery(q.sku_id)
    if (sid) form.sku_id = sid
  } else if (q.length_cm || q.width_cm || q.height_cm || q.weight_kg) {
    mode.value = 'manual'
  }
  if (q.country && typeof q.country === 'string') form.destination_country = q.country.toUpperCase()
  if (q.cargo_type && typeof q.cargo_type === 'string') form.cargo_type = q.cargo_type
  if (q.quantity) {
    const qty = numberFromQuery(q.quantity)
    if (qty) form.quantity = qty
  }
  if (q.length_cm) {
    const val = numberFromQuery(q.length_cm)
    if (val) form.length_cm = val
  }
  if (q.width_cm) {
    const val = numberFromQuery(q.width_cm)
    if (val) form.width_cm = val
  }
  if (q.height_cm) {
    const val = numberFromQuery(q.height_cm)
    if (val) form.height_cm = val
  }
  if (q.weight_kg) {
    const val = numberFromQuery(q.weight_kg)
    if (val) form.weight_kg = val
  }
  if (q.source_product_id || q.source_product_name) {
    sourceProduct.value = {
      id: typeof q.source_product_id === 'string' ? q.source_product_id : undefined,
      name: typeof q.source_product_name === 'string' ? q.source_product_name : undefined,
    }
  }
})

async function handleCalculate() {
  normalizeCountry()
  if (!canCalculate.value) {
    message.warning('请填写所有必填字段')
    return
  }
  loading.value = true
  try {
    const payload = mode.value === 'manual'
      ? {
          mode: 'manual',
          quantity: form.quantity,
          destination_country: form.destination_country,
          postal_code: form.postal_code || undefined,
          cargo_type: form.cargo_type,
          package: {
            length_cm: form.length_cm,
            width_cm: form.width_cm,
            height_cm: form.height_cm,
            weight_kg: form.weight_kg,
          },
        }
      : {
          mode: 'sku',
          sku_id: form.sku_id,
          quantity: form.quantity,
          destination_country: form.destination_country,
          postal_code: form.postal_code || undefined,
          cargo_type: form.cargo_type,
        }
    const res: any = await shippingApi.calculate(payload)
    if (res.code === 200) {
      result.value = res.data
    } else {
      message.error(res.message || '计算失败')
    }
  } catch (e: any) {
    message.error(e.message || '请求失败')
  } finally {
    loading.value = false
  }
}
</script>
