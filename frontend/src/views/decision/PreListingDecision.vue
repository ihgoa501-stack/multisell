<template>
  <div>
    <h3 style="margin-bottom: 16px;">📊 上架前经营决策</h3>

    <!-- 步骤条 -->
    <n-steps :current="step" :status="stepStatus">
      <n-step title="选择商品" description="搜索SKU查看成本与物流数据" />
      <n-step title="销售条件" description="设置售价、目的国、平台费率" />
      <n-step title="决策结果" description="利润分析及推荐建议" />
    </n-steps>

    <!-- ========== Step 1: 选择 SKU ========== -->
    <div v-show="step === 0" style="margin-top: 24px;">
      <n-card title="选择 SKU">
        <n-form label-placement="top">
          <n-form-item label="SKU ID">
            <n-input-number
              v-model:value="form.sku_id"
              placeholder="输入 SKU ID"
              :min="1"
              style="width: 240px;"
            />
            <n-button
              style="margin-left: 12px;"
              :loading="loadingSku"
              @click="loadSkuInfo"
            >
              查询
            </n-button>
          </n-form-item>
        </n-form>

        <!-- SKU 信息卡片 -->
        <n-card v-if="skuInfo" title="商品信息" size="small" style="margin-top: 12px;">
          <n-descriptions :column="3" bordered size="small">
            <n-descriptions-item label="SKU编码">{{ skuInfo.code || '-' }}</n-descriptions-item>
            <n-descriptions-item label="规格">{{ skuInfo.spec_desc || '-' }}</n-descriptions-item>
            <n-descriptions-item label="成本价">{{ skuInfo.cost_price ?? '-' }} 元</n-descriptions-item>
            <n-descriptions-item label="尺寸">
              {{ skuInfo.sku_length_cm ?? '-' }} × {{ skuInfo.sku_width_cm ?? '-' }} × {{ skuInfo.sku_height_cm ?? '-' }} cm
            </n-descriptions-item>
            <n-descriptions-item label="重量">{{ skuInfo.sku_weight_kg ?? '-' }} kg</n-descriptions-item>
            <n-descriptions-item label="库存">{{ skuInfo.stock }}</n-descriptions-item>
          </n-descriptions>
          <template #footer>
            <n-tag v-if="hasLogisticsData" type="success">物流数据完整</n-tag>
            <n-tag v-else type="warning">物流数据不完整，无法计算运费</n-tag>
          </template>
        </n-card>
      </n-card>
      <div style="margin-top: 16px; text-align: right;">
        <n-button type="primary" :disabled="!skuInfo" @click="step = 1">
          下一步：销售条件
        </n-button>
      </div>
    </div>

    <!-- ========== Step 2: 销售条件 ========== -->
    <div v-show="step === 1" style="margin-top: 24px;">
      <n-card title="配置销售条件">
        <n-grid :cols="24" :x-gap="16" :y-gap="12">
          <n-form-item-gi :span="8" label="目标售价">
            <n-input-number v-model:value="form.target_sale_price" :min="0.01" :precision="2" style="width: 100%;">
              <template #suffix>元</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="目的国">
            <n-select
              v-model:value="form.destination_country"
              :options="countryOptions"
              filterable
              style="width: 100%;"
            />
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="货品类型">
            <n-select v-model:value="form.cargo_type" :options="cargoTypeOptions" style="width: 100%;" />
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="对比平台">
            <n-select
              v-model:value="comparePlatformIds"
              :options="platformOptions"
              multiple
              filterable
              placeholder="可选，多选对比"
              style="width: 100%;"
            />
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="支付费率">
            <n-input-number v-model:value="form.payment_fee_pct" :min="0" :max="100" :precision="1" style="width: 100%;">
              <template #suffix>%</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="其他费用">
            <n-input-number v-model:value="form.other_fee" :min="0" :precision="2" style="width: 100%;">
              <template #suffix>元</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="最低利润率">
            <n-input-number v-model:value="form.minimum_margin_pct" :min="0" :max="100" :precision="1" style="width: 100%;">
              <template #suffix>%</template>
            </n-input-number>
          </n-form-item-gi>
        </n-grid>
      </n-card>
      <div style="margin-top: 16px; display: flex; justify-content: space-between;">
        <n-button @click="step = 0">上一步</n-button>
        <n-button type="primary" :loading="loading" @click="handleCalculate">
          计算决策
        </n-button>
      </div>
    </div>

    <!-- ========== Step 3: 决策结果 ========== -->
    <div v-show="step === 2" style="margin-top: 24px;">
      <!-- 单平台结果 -->
      <n-card v-if="result" title="利润分析" style="margin-bottom: 16px;">
        <n-alert
          :type="result.recommendation === 'approve' ? 'success' : result.recommendation === 'reject' ? 'error' : 'warning'"
          :show-icon="false"
          style="margin-bottom: 16px;"
        >
          <template #header>
            <span style="font-size: 16px; font-weight: bold;">
              {{ result.recommendation === 'approve' ? '✅ 建议上架' : result.recommendation === 'reject' ? '❌ 不建议上架' : '⚠️ 数据不足' }}
            </span>
          </template>
          <div v-for="(reason, idx) in result.blocking_reasons" :key="'br-' + idx" style="color: #d03050;">
            ⛔ {{ reason }}
          </div>
          <div v-for="(warn, idx) in result.warnings" :key="'w-' + idx" style="color: #f0a020;">
            ⚠️ {{ warn }}
          </div>
        </n-alert>

        <!-- 成本构成表 -->
        <n-table :single-line="false" size="small">
          <thead>
            <tr>
              <th>项目</th>
              <th style="text-align: right;">金额（元）</th>
              <th style="text-align: right;">占比</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in costBreakdown" :key="item.label">
              <td>{{ item.label }}</td>
              <td style="text-align: right;">{{ item.value.toFixed(2) }}</td>
              <td style="text-align: right;">{{ item.pct }}%</td>
            </tr>
            <tr style="font-weight: bold; background: #f5f5f5;">
              <td>总成本</td>
              <td style="text-align: right;">{{ totalCost.toFixed(2) }}</td>
              <td style="text-align: right;">100%</td>
            </tr>
            <tr :style="{ color: result.profit_amount >= 0 ? '#18a058' : '#d03050', fontWeight: 'bold' }">
              <td>利润</td>
              <td style="text-align: right;">{{ result.profit_amount.toFixed(2) }}</td>
              <td style="text-align: right;">{{ result.profit_margin.toFixed(1) }}%</td>
            </tr>
          </tbody>
        </n-table>

        <!-- 瀑布图（简易 CSS 版） -->
        <div style="margin-top: 16px;">
          <div style="font-size: 13px; color: #666; margin-bottom: 8px;">成本瀑布图</div>
          <div
            v-for="bar in waterfallBars"
            :key="bar.label"
            style="display: flex; align-items: center; margin-bottom: 4px; gap: 8px;"
          >
            <span style="width: 80px; font-size: 12px; text-align: right; flex-shrink: 0;">{{ bar.label }}</span>
            <div
              :style="{
                height: '20px',
                width: bar.pct + '%',
                background: bar.color,
                borderRadius: '3px',
                minWidth: bar.pct > 0 ? '4px' : '0',
              }"
            />
            <span style="font-size: 12px;">{{ bar.value.toFixed(2) }}</span>
          </div>
        </div>
      </n-card>

      <!-- 多平台对比 -->
      <n-card v-if="compareResult && compareResult.results.length > 0" title="多平台对比" style="margin-bottom: 16px;">
        <n-table :single-line="false" size="small">
          <thead>
            <tr>
              <th>平台</th>
              <th style="text-align: right;">运费</th>
              <th style="text-align: right;">平台费</th>
              <th style="text-align: right;">总成本</th>
              <th style="text-align: right;">利润</th>
              <th style="text-align: right;">利润率</th>
              <th>建议</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in sortedCompareResults" :key="item.platform_id">
              <td>{{ item.platform_name }}</td>
              <td style="text-align: right;">{{ item.shipping_fee.toFixed(2) }}</td>
              <td style="text-align: right;">{{ item.platform_fee.toFixed(2) }}</td>
              <td style="text-align: right;">{{ item.total_cost.toFixed(2) }}</td>
              <td
                style="text-align: right; font-weight: bold;"
                :style="{ color: item.profit_amount >= 0 ? '#18a058' : '#d03050' }"
              >
                {{ item.profit_amount.toFixed(2) }}
              </td>
              <td style="text-align: right;">{{ item.profit_margin.toFixed(1) }}%</td>
              <td>
                <n-tag
                  :type="item.recommendation === 'approve' ? 'success' : item.recommendation === 'reject' ? 'error' : 'warning'"
                  size="small"
                >
                  {{ item.recommendation === 'approve' ? '上架' : item.recommendation === 'reject' ? '不推荐' : '缺数据' }}
                </n-tag>
              </td>
            </tr>
          </tbody>
        </n-table>
      </n-card>

      <div style="margin-top: 16px; display: flex; justify-content: space-between;">
        <n-button @click="step = 1">重新配置</n-button>
        <n-button @click="resetAll">重新开始</n-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import {
  calculatePreListingDecision,
  comparePreListingDecision,
  type PreListingDecisionRequest,
  type PreListingDecisionResponse,
  type CompareDecisionResponse,
  type CompareItem,
} from '@/api/modules/decision'
import { skuApi } from '@/api'
import { platformApi } from '@/api'

const message = useMessage()

// 步骤状态
const step = ref(0)
const stepStatus = ref<'process' | 'finish' | 'error'>('process')
const loading = ref(false)
const loadingSku = ref(false)

// 表单
const form = reactive<PreListingDecisionRequest>({
  sku_id: null as unknown as number,
  destination_country: '',
  target_sale_price: null as unknown as number,
  platform_fee_pct: 10,
  payment_fee_pct: 3,
  other_fee: 0,
  minimum_margin_pct: 20,
  cargo_type: 'normal',
})

const comparePlatformIds = ref<number[]>([])

// SKU 信息
const skuInfo = ref<any>(null)

// 结果
const result = ref<PreListingDecisionResponse | null>(null)
const compareResult = ref<CompareDecisionResponse | null>(null)

// 平台列表
const platformOptions = ref<{ label: string; value: number }[]>([])

// 选项
const cargoTypeOptions = [
  { label: '普通', value: 'normal' },
  { label: '带电', value: 'battery' },
  { label: '液体', value: 'liquid' },
  { label: '敏感', value: 'sensitive' },
]

const countryOptions = [
  { label: '俄罗斯 (RU)', value: 'RU' },
  { label: '哈萨克斯坦 (KZ)', value: 'KZ' },
  { label: '白俄罗斯 (BY)', value: 'BY' },
  { label: '中国 (CN)', value: 'CN' },
  { label: '美国 (US)', value: 'US' },
  { label: '德国 (DE)', value: 'DE' },
  { label: '法国 (FR)', value: 'FR' },
  { label: '英国 (GB)', value: 'GB' },
  { label: '日本 (JP)', value: 'JP' },
  { label: '韩国 (KR)', value: 'KR' },
  { label: '印度尼西亚 (ID)', value: 'ID' },
  { label: '泰国 (TH)', value: 'TH' },
  { label: '越南 (VN)', value: 'VN' },
  { label: '菲律宾 (PH)', value: 'PH' },
  { label: '马来西亚 (MY)', value: 'MY' },
  { label: '新加坡 (SG)', value: 'SG' },
  { label: '巴西 (BR)', value: 'BR' },
  { label: '墨西哥 (MX)', value: 'MX' },
]

// 计算属性
const hasLogisticsData = computed(() => {
  const s = skuInfo.value
  return s
    && Number(s.sku_length_cm) > 0
    && Number(s.sku_width_cm) > 0
    && Number(s.sku_height_cm) > 0
    && Number(s.sku_weight_kg) > 0
})

const totalCost = computed(() => {
  if (!result.value) return 0
  const r = result.value
  return r.product_cost + r.shipping_fee + r.platform_fee + r.payment_fee + r.other_fee
})

const costBreakdown = computed(() => {
  if (!result.value) return []
  const r = result.value
  const total = totalCost.value || 1
  return [
    { label: '商品成本', value: r.product_cost, pct: ((r.product_cost / total) * 100).toFixed(1) },
    { label: '运费', value: r.shipping_fee, pct: ((r.shipping_fee / total) * 100).toFixed(1) },
    { label: '平台费', value: r.platform_fee, pct: ((r.platform_fee / total) * 100).toFixed(1) },
    { label: '支付费', value: r.payment_fee, pct: ((r.payment_fee / total) * 100).toFixed(1) },
    { label: '其他费用', value: r.other_fee, pct: ((r.other_fee / total) * 100).toFixed(1) },
  ]
})

const waterfallBars = computed(() => {
  if (!result.value) return []
  const r = result.value
  const maxVal = Math.max(r.target_sale_price, 1)
  return [
    { label: '售价', value: r.target_sale_price, pct: (r.target_sale_price / maxVal) * 100, color: '#18a058' },
    { label: '- 商品成本', value: -r.product_cost, pct: (r.product_cost / maxVal) * 100, color: '#d03050' },
    { label: '- 运费', value: -r.shipping_fee, pct: (r.shipping_fee / maxVal) * 100, color: '#d03050' },
    { label: '- 平台费', value: -r.platform_fee, pct: (r.platform_fee / maxVal) * 100, color: '#d03050' },
    { label: '- 支付费', value: -r.payment_fee, pct: (r.payment_fee / maxVal) * 100, color: '#d03050' },
    { label: '- 其他', value: -r.other_fee, pct: (r.other_fee / maxVal) * 100, color: '#d03050' },
    { label: '= 利润', value: r.profit_amount, pct: (Math.abs(r.profit_amount) / maxVal) * 100, color: r.profit_amount >= 0 ? '#18a058' : '#d03050' },
  ]
})

const sortedCompareResults = computed(() => {
  if (!compareResult.value) return []
  return [...compareResult.value.results].sort((a, b) => b.profit_margin - a.profit_margin)
})

// 方法
async function loadSkuInfo() {
  if (!form.sku_id) {
    message.warning('请输入 SKU ID')
    return
  }
  loadingSku.value = true
  try {
    const resp = await skuApi.getSku(form.sku_id)
    skuInfo.value = resp.data as any
    // 预填目标售价
    if (skuInfo.value?.cost_price) {
      form.target_sale_price = Number((skuInfo.value.cost_price * 2.5).toFixed(2))
    }
  } catch (err: any) {
    message.error('查询SKU失败：' + (err?.response?.data?.message || err?.message || '未知错误'))
    skuInfo.value = null
  } finally {
    loadingSku.value = false
  }
}

async function handleCalculate() {
  // 验证
  if (!form.destination_country) {
    message.warning('请选择目的国')
    return
  }
  if (!form.target_sale_price || form.target_sale_price <= 0) {
    message.warning('请输入目标售价')
    return
  }

  loading.value = true
  result.value = null
  compareResult.value = null

  try {
    // 如果有多个对比平台，走对比 API
    if (comparePlatformIds.value.length > 0) {
      const resp = await comparePreListingDecision({
        sku_id: form.sku_id,
        destination_country: form.destination_country,
        target_sale_price: form.target_sale_price,
        platform_ids: comparePlatformIds.value,
        payment_fee_pct: form.payment_fee_pct,
        other_fee: form.other_fee,
        minimum_margin_pct: form.minimum_margin_pct,
        cargo_type: form.cargo_type,
      })
      compareResult.value = resp.data as unknown as CompareDecisionResponse
      // 用第一个平台作为主结果展示
      if (compareResult.value.results.length > 0) {
        const first = compareResult.value.results[0]
        result.value = {
          sku_id: form.sku_id,
          destination_country: form.destination_country,
          target_sale_price: form.target_sale_price,
          product_cost: first.product_cost,
          shipping_fee: first.shipping_fee,
          platform_fee: first.platform_fee,
          payment_fee: first.payment_fee,
          other_fee: form.other_fee,
          profit_amount: first.profit_amount,
          profit_margin: first.profit_margin,
          recommendation: first.recommendation,
          blocking_reasons: first.blocking_reasons,
          warnings: first.warnings,
        } as PreListingDecisionResponse
      }
    } else {
      const resp = await calculatePreListingDecision({ ...form })
      result.value = resp.data as unknown as PreListingDecisionResponse
    }
    step.value = 2
  } catch (err: any) {
    message.error(err?.response?.data?.detail || err?.response?.data?.message || err?.message || '请求失败')
  } finally {
    loading.value = false
  }
}

function resetAll() {
  step.value = 0
  result.value = null
  compareResult.value = null
  skuInfo.value = null
  form.sku_id = null as unknown as number
  form.destination_country = ''
  form.target_sale_price = null as unknown as number
  form.platform_fee_pct = 10
  form.payment_fee_pct = 3
  form.other_fee = 0
  form.minimum_margin_pct = 20
  form.cargo_type = 'normal'
  comparePlatformIds.value = []
}

// 加载平台列表
onMounted(async () => {
  try {
    const resp = await platformApi.list()
    const platforms = (resp.data as any)?.items || resp.data || []
    platformOptions.value = (Array.isArray(platforms) ? platforms : []).map((p: any) => ({
      label: p.name || p.platform_name,
      value: p.id,
    }))
  } catch {
    // 非关键，静默
  }
})
</script>
