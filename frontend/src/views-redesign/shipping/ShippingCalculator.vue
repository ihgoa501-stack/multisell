<template>
  <div>
    <h3 style="margin-bottom: 16px;">运费计算器</h3>
    <a-card>
      <div style="margin-bottom: 16px;">
        <a-radio-group v-model:value="mode" button-style="solid">
          <a-radio-button value="manual">手动输入</a-radio-button>
          <a-radio-button value="sku">按 SKU 计算</a-radio-button>
        </a-radio-group>
      </div>
      <a-alert
        v-if="sourceProduct.id || sourceProduct.name"
        type="info"
        style="margin-bottom: 12px;"
        :show-icon="false"
        :message="`来源商品：${sourceProduct.name || `ID ${sourceProduct.id}`}`"
      />
      <a-row :gutter="12">
        <template v-if="mode === 'manual'">
          <a-col :span="6">
            <a-form-item label="包装长" style="margin-bottom: 12px;">
              <a-input-number v-model:value="form.length_cm" :min="0.01" :precision="2" style="width: 100%;" addon-after="cm" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="包装宽" style="margin-bottom: 12px;">
              <a-input-number v-model:value="form.width_cm" :min="0.01" :precision="2" style="width: 100%;" addon-after="cm" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="包装高" style="margin-bottom: 12px;">
              <a-input-number v-model:value="form.height_cm" :min="0.01" :precision="2" style="width: 100%;" addon-after="cm" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="包装重量" style="margin-bottom: 12px;">
              <a-input-number v-model:value="form.weight_kg" :min="0.01" :precision="3" style="width: 100%;" addon-after="kg" />
            </a-form-item>
          </a-col>
        </template>
        <a-col v-if="mode === 'sku'" :span="6">
          <a-form-item label="SKU ID" style="margin-bottom: 12px;">
            <a-input-number v-model:value="form.sku_id" placeholder="输入SKU ID" :min="1" style="width: 100%;" />
          </a-form-item>
        </a-col>
        <a-col :span="4">
          <a-form-item label="数量" style="margin-bottom: 12px;">
            <a-input-number v-model:value="form.quantity" :min="1" style="width: 100%;" />
          </a-form-item>
        </a-col>
        <a-col :span="5">
          <a-form-item label="目的地国家" style="margin-bottom: 12px;">
            <a-input
              v-model:value="form.destination_country"
              placeholder="如 US, DE"
              :maxlength="10"
              @blur="normalizeCountry"
              @change="normalizeCountry"
            />
          </a-form-item>
        </a-col>
        <a-col :span="5">
          <a-form-item label="货品类型" style="margin-bottom: 12px;">
            <a-select v-model:value="form.cargo_type" :options="cargoOptions" />
          </a-form-item>
        </a-col>
        <a-col :span="4">
          <a-form-item label="邮编(可选)" style="margin-bottom: 12px;">
            <a-input v-model:value="form.postal_code" placeholder="可选" />
          </a-form-item>
        </a-col>
      </a-row>
      <a-button type="primary" @click="handleCalculate" :loading="loading" :disabled="!canCalculate">
        计算运费
      </a-button>
    </a-card>

    <a-card v-if="result" style="margin-top: 16px;">
      <template #title>
        计算结果 — {{ result.mode === 'manual' ? '手动包裹' : `SKU ${result.sku_id}` }} x {{ result.quantity }} -> {{ result.destination_country }}
      </template>
      <a-descriptions bordered size="small" style="margin-bottom: 12px;">
        <a-descriptions-item label="计算模式">{{ result.mode === 'manual' ? '手动输入' : 'SKU' }}</a-descriptions-item>
        <a-descriptions-item label="目的地国家">{{ result.destination_country }}</a-descriptions-item>
        <a-descriptions-item label="数量">{{ result.quantity }}</a-descriptions-item>
        <a-descriptions-item label="包装来源">{{ packageSourceLabel(result.package?.source) }}</a-descriptions-item>
        <a-descriptions-item label="包装尺寸">{{ result.package?.length_cm }}x{{ result.package?.width_cm }}x{{ result.package?.height_cm }} cm</a-descriptions-item>
        <a-descriptions-item label="包装重量">{{ result.package?.weight_kg }} kg</a-descriptions-item>
      </a-descriptions>

      <a-table
        v-if="result.results && result.results.length > 0"
        :columns="tableColumns"
        :data-source="result.results"
        :pagination="false"
        size="small"
        row-key="channel_name"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'estimated_delivery_min'">
            {{ record.estimated_delivery_min != null ? `${record.estimated_delivery_min}-${record.estimated_delivery_max}` : '-' }}
          </template>
        </template>
      </a-table>
      <a-empty v-else description="没有匹配的物流渠道，请检查目的地国家、货品类型或报价规则。" />
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { shippingApi } from '@/api/modules/shipping'

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
  { title: '供应商', dataIndex: 'provider_name', key: 'provider_name', width: 120 },
  { title: '渠道', dataIndex: 'channel_name', key: 'channel_name', width: 120 },
  { title: '币种', dataIndex: 'currency', key: 'currency', width: 60 },
  { title: '实际重(kg)', dataIndex: 'actual_weight_kg', key: 'actual_weight_kg', width: 100 },
  { title: '体积重(kg)', dataIndex: 'volumetric_weight_kg', key: 'volumetric_weight_kg', width: 100 },
  { title: '计费重(kg)', dataIndex: 'chargeable_weight_kg', key: 'chargeable_weight_kg', width: 100 },
  { title: '基础费', dataIndex: 'base_shipping_fee', key: 'base_shipping_fee', width: 80 },
  { title: '附加费', dataIndex: 'surcharge_fee', key: 'surcharge_fee', width: 80 },
  { title: '燃油费', dataIndex: 'fuel_surcharge_fee', key: 'fuel_surcharge_fee', width: 80 },
  { title: '总运费', dataIndex: 'total_shipping_fee', key: 'total_shipping_fee', width: 90, sorter: (a: any, b: any) => a.total_shipping_fee - b.total_shipping_fee },
  { title: '时效(天)', dataIndex: 'estimated_delivery_min', key: 'estimated_delivery_min', width: 80 },
  { title: '说明', dataIndex: 'calculation_detail', key: 'calculation_detail', ellipsis: true },
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
