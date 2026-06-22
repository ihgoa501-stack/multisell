<template>
  <div class="max-w-7xl mx-auto">
    <!-- ====== 页面标题 ====== -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-xl font-bold text-[var(--text-primary)]">利润测算</h1>
        <p v-if="skuInfo" class="text-sm text-[var(--text-tertiary)] mt-0.5">
          {{ skuInfo.product_name || '未知商品' }} · {{ skuInfo.code || '' }}
        </p>
        <p v-else class="text-sm text-[var(--text-tertiary)] mt-0.5">选择商品开始测算</p>
      </div>
      <div class="flex items-center gap-2">
        <a-button size="small" type="text" @click="handleExport">
          <span class="inline-flex items-center gap-1">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
            导出
          </span>
        </a-button>
        <a-button type="primary" size="small" :disabled="!result" @click="handleSave">保存决策</a-button>
      </div>
    </div>

    <!-- ====== Agent 状态条 ====== -->
    <div class="flex items-center gap-5 px-4 py-2.5 bg-white border border-[var(--border-light)] rounded-lg mb-6">
      <span class="text-[10px] font-semibold text-[var(--text-tertiary)] tracking-wider uppercase">活跃 Agent</span>
      <div class="flex items-center gap-1.5 flex-wrap">
        <span v-for="a in activeAgents" :key="a.name"
          class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-[var(--bg-subtle)] text-[var(--text-secondary)]">
          <span class="w-1.5 h-1.5 rounded-full" :class="a.active ? 'bg-[var(--green)]' : 'bg-[var(--text-tertiary)]'"></span>
          {{ a.label }}
        </span>
      </div>
      <span class="ml-auto text-[11px] font-mono text-[var(--text-tertiary)]">2.3s · {{ now }}</span>
    </div>

    <!-- ====== 双栏主体 ====== -->
    <div class="grid grid-cols-[340px_1fr] gap-4 items-start mb-6">
      <!-- 左栏：商品搜索 -->
      <div class="bg-white border border-[var(--border-light)] rounded-lg">
        <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--border-light)]">
          <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">选择商品</span>
          <span class="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-[var(--accent-bg)] text-[var(--accent)]">{{ skuList.length }} 个待决策</span>
        </div>
        <div class="p-4">
          <!-- 搜索框 -->
          <div class="flex items-center gap-1.5 px-3 py-1.5 border border-[var(--border)] rounded-[6px] mb-3 transition-colors focus-within:border-[var(--accent)]">
            <svg class="w-3.5 h-3.5 text-[var(--text-tertiary)] shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><path d="M21 21l-4.3-4.3"/></svg>
            <input
              v-model="searchQuery"
              type="text"
              placeholder="搜索商品或 SKU..."
              class="w-full border-none bg-transparent outline-none text-[13px] text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)] font-sans"
            >
          </div>
          <!-- SKU 列表 -->
          <div class="flex flex-col gap-0.5">
            <div
              v-for="sku in filteredSkus"
              :key="sku.id"
              class="flex items-center gap-2.5 px-2.5 py-2 rounded-[6px] cursor-pointer transition-colors"
              :class="selectedSkuId === sku.id ? 'bg-[var(--accent-bg)]' : 'hover:bg-[var(--bg-hover)]'"
              @click="selectSku(sku)"
            >
              <span class="text-base w-7 text-center shrink-0">{{ sku.emoji || '📦' }}</span>
              <div class="flex-1 min-w-0">
                <div class="text-[13px] font-medium text-[var(--text-primary)] truncate">{{ sku.product_name || '商品' }}</div>
                <div class="text-[11px] text-[var(--text-tertiary)] mt-px">{{ sku.code }} · ¥{{ sku.cost_price || '--' }}</div>
              </div>
              <div class="flex gap-1 shrink-0">
                <span v-if="sku.logistics_ok" class="tag text-[10px] px-1.5 py-0.5 rounded font-medium bg-[var(--green-bg)] text-[var(--green)]">完整</span>
                <span v-else class="tag text-[10px] px-1.5 py-0.5 rounded font-medium bg-[var(--amber-bg)] text-[var(--amber)]">缺资料</span>
              </div>
            </div>
            <div v-if="filteredSkus.length === 0" class="text-center py-8 text-sm text-[var(--text-tertiary)]">暂无匹配商品</div>
          </div>
        </div>
      </div>

      <!-- 右栏：工作台 -->
      <div class="flex flex-col gap-3.5">
        <!-- 商品信息 + 销售条件 -->
        <div class="bg-white border border-[var(--border-light)] rounded-lg">
          <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--border-light)]">
            <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">商品信息</span>
            <span v-if="skuInfo" class="text-[11px] text-[var(--text-tertiary)]">
              成本 <span class="font-mono">¥{{ skuInfo.cost_price || '--' }}</span>
              · 库存 <span class="font-mono">{{ skuInfo.stock ?? '--' }}</span>
              · {{ dimText }}
            </span>
          </div>
          <div class="px-4 py-2.5">
            <div class="grid grid-cols-[70px_1fr] items-center gap-x-3 gap-y-2">
              <span class="text-[11px] text-[var(--text-tertiary)]">目标售价</span>
              <div class="flex gap-1.5">
                <a-input v-model:value="formData.target_sale_price" size="small" placeholder="售价" class="flex-1" />
                <a-select v-model:value="formData.destination_country" :options="countryOptions" placeholder="目的国" size="small" class="w-[120px]" show-search :filter-option="filterOption" />
                <a-select v-model:value="platformId" :options="platformOptions" placeholder="平台" size="small" class="w-[110px]" />
                <a-input v-model:value="paymentFeeStr" size="small" placeholder="支付费%" class="w-[70px]" />
              </div>
            </div>
          </div>
        </div>

        <!-- 物流方案 -->
        <div class="bg-white border border-[var(--border-light)] rounded-lg">
          <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--border-light)]">
            <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">物流方案</span>
            <span class="text-[11px] text-[var(--accent)] font-medium">A5 推荐</span>
          </div>
          <div class="px-4 py-3">
            <div class="flex gap-2">
              <div
                v-for="ch in logisticsOptions"
                :key="ch.name"
                class="flex-1 px-3 py-2.5 border rounded-[6px] cursor-pointer transition-all"
                :class="selectedChannel === ch.name
                  ? 'border-[var(--accent)] bg-[var(--accent-bg)]'
                  : 'border-[var(--border)] hover:border-[var(--text-tertiary)]'"
                @click="selectedChannel = ch.name"
              >
                <div v-if="ch.best" class="text-[9px] font-semibold text-[var(--accent)] uppercase tracking-wider mb-0.5">推荐</div>
                <div class="text-[12px] font-medium text-[var(--text-primary)]">{{ ch.name }}</div>
                <div class="text-base font-bold text-[var(--text-primary)] mt-0.5">¥{{ ch.price }} <span class="text-[11px] font-normal text-[var(--text-tertiary)]">/件</span></div>
                <div class="text-[11px] text-[var(--text-tertiary)] mt-1 flex gap-2">
                  <span>{{ ch.duration }}</span>
                  <span>{{ ch.weight }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 利润分析 -->
        <div class="bg-white border border-[var(--border-light)] rounded-lg">
          <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--border-light)]">
            <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">利润分析</span>
            <span class="text-[11px] text-[var(--text-tertiary)]">最低利润率 {{ formData.minimum_margin_pct }}%</span>
          </div>
          <div class="px-4 py-3">
            <!-- 瀑布图 -->
            <div class="mb-4">
              <div v-for="(item, i) in waterfallItems" :key="i" class="flex items-center gap-3.5 mb-1.5">
                <span class="w-[72px] text-right text-[12px] shrink-0"
                  :class="item.bold ? 'font-semibold text-[var(--text-primary)]' : 'text-[var(--text-tertiary)]'">
                  {{ item.label }}
                </span>
                <div class="flex-1 h-5 bg-[var(--bg-subtle)] rounded overflow-hidden">
                  <div class="h-full rounded transition-all duration-500" :class="barColor(item)" :style="{ width: item.pct + '%', minWidth: '3px' }"></div>
                </div>
                <span class="w-[80px] text-right text-[12px] font-mono shrink-0" :class="amountColor(item)">{{ item.amount }}</span>
              </div>
              <div class="h-px bg-[var(--border-light)] my-1.5"></div>
            </div>

            <!-- 利润率条 -->
            <div class="flex items-center gap-2.5 mb-4">
              <div class="flex-1 h-1 bg-[var(--bg-subtle)] rounded relative">
                <div class="h-full bg-[var(--green)] rounded" :style="{ width: Math.min(marginPct, 100) + '%' }"></div>
                <div class="absolute right-[80%] top-[-2px] w-0.5 h-1.5 bg-[var(--text-tertiary)]"></div>
              </div>
              <span class="text-sm font-bold font-mono" :class="marginPct >= (formData.minimum_margin_pct || 20) ? 'text-[var(--green)]' : 'text-[var(--red)]'">
                {{ marginPct }}%
              </span>
            </div>

            <!-- 决策结果 -->
            <div v-if="result" class="flex items-center justify-between px-4 py-3 rounded-[6px] border"
              :class="resultBgClass">
              <div class="flex items-center gap-2.5">
                <span class="inline-flex items-center gap-1.5 text-[12px] font-semibold" :class="resultTextClass">
                  <svg v-if="result.recommendation === 'approve'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                  <svg v-else-if="result.recommendation === 'reject'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                  <svg v-else class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                  {{ resultLabel }}
                </span>
                <span class="text-[12px]" :class="resultDetailClass">
                  <template v-if="result.recommendation === 'approve'">利润率 {{ marginPct }}% ≥ 最低 {{ formData.minimum_margin_pct }}%</template>
                  <template v-else-if="result.recommendation === 'reject'">利润率 {{ marginPct }}% < 最低 {{ formData.minimum_margin_pct }}%</template>
                  <template v-else>{{ result.blocking_reasons?.join('；') }}</template>
                </span>
              </div>
              <div class="flex gap-1.5">
                <a-button size="small" type="text" @click="handleCompare">多平台对比</a-button>
                <a-button size="small" type="primary" @click="handleSave">保存决策</a-button>
              </div>
            </div>

            <!-- AI 建议 -->
            <div v-if="result" class="mt-3.5 px-4 py-3.5 bg-[var(--bg-subtle)] border border-[var(--border-light)] rounded-[6px] flex gap-2.5">
              <div class="w-6 h-6 rounded-[5px] bg-[var(--accent)] flex items-center justify-center text-white text-[11px] font-semibold shrink-0">AI</div>
              <div>
                <div class="text-[12px] font-semibold text-[var(--text-primary)]">{{ aiTitle }}</div>
                <div class="text-[12px] text-[var(--text-secondary)] mt-0.5 leading-relaxed">
                  {{ aiDescription }}
                  <span class="text-[var(--accent)] cursor-pointer hover:underline"> 查看详情 →</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ====== 平台对比表格 ====== -->
    <div v-if="compareResults.length > 0" class="mb-6">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-sm font-semibold text-[var(--text-primary)]">平台对比</h2>
        <a-button size="small" type="text" @click="handleExportCompare">导出对比报告</a-button>
      </div>
      <div class="bg-white border border-[var(--border-light)] rounded-lg overflow-hidden">
        <table class="w-full text-[13px] border-collapse">
          <thead>
            <tr class="text-[11px] font-semibold text-[var(--text-tertiary)] uppercase tracking-wide">
              <th class="text-left py-2.5 px-3 border-b border-[var(--border-light)] w-7"></th>
              <th class="text-left py-2.5 px-3 border-b border-[var(--border-light)]">平台</th>
              <th class="text-right py-2.5 px-3 border-b border-[var(--border-light)]">运费</th>
              <th class="text-right py-2.5 px-3 border-b border-[var(--border-light)]">平台费</th>
              <th class="text-right py-2.5 px-3 border-b border-[var(--border-light)]">总成本</th>
              <th class="text-right py-2.5 px-3 border-b border-[var(--border-light)]">利润</th>
              <th class="text-right py-2.5 px-3 border-b border-[var(--border-light)]">利润率</th>
              <th class="text-center py-2.5 px-3 border-b border-[var(--border-light)]">建议</th>
              <th class="py-2.5 px-3 border-b border-[var(--border-light)] w-14"></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(item, i) in sortedCompare"
              :key="item.platform_id"
              class="transition-colors hover:bg-[var(--bg-hover)]"
              :class="i === 0 ? 'bg-[var(--accent-bg)]' : ''"
            >
              <td class="py-3 px-3 border-b border-[var(--border-light)]">
                <span v-if="i === 0" class="inline-flex items-center justify-center w-5.5 h-5.5 rounded text-[11px] font-semibold bg-[#fef3c7] text-[#92400e]">🥇</span>
                <span v-else-if="i === 1" class="inline-flex items-center justify-center w-5.5 h-5.5 rounded text-[11px] font-semibold bg-[var(--bg-subtle)] text-[var(--text-tertiary)]">🥈</span>
                <span v-else-if="i === 2" class="inline-flex items-center justify-center w-5.5 h-5.5 rounded text-[11px] font-semibold bg-[var(--bg-subtle)] text-[var(--text-tertiary)]">🥉</span>
                <span v-else class="text-center text-[11px] text-[var(--text-tertiary)] font-semibold block">{{ i + 1 }}</span>
              </td>
              <td class="py-3 px-3 border-b border-[var(--border-light)] font-medium">
                {{ item.platform_name }}
                <span v-if="i === 0" class="text-[10px] font-semibold text-[var(--accent)] ml-1">推荐</span>
              </td>
              <td class="py-3 px-3 border-b border-[var(--border-light)] text-right font-mono">¥{{ item.shipping_fee.toFixed(2) }}</td>
              <td class="py-3 px-3 border-b border-[var(--border-light)] text-right font-mono">¥{{ item.platform_fee.toFixed(2) }}</td>
              <td class="py-3 px-3 border-b border-[var(--border-light)] text-right font-mono">¥{{ item.total_cost.toFixed(2) }}</td>
              <td class="py-3 px-3 border-b border-[var(--border-light)] text-right font-mono" :class="item.profit_amount >= 0 ? 'text-[var(--green)]' : 'text-[var(--red)]'">
                {{ item.profit_amount >= 0 ? '+' : '' }}¥{{ item.profit_amount.toFixed(2) }}
              </td>
              <td class="py-3 px-3 border-b border-[var(--border-light)] text-right font-mono" :class="item.profit_margin >= (formData.minimum_margin_pct || 20) ? 'text-[var(--green)]' : 'text-[var(--amber)]'">
                {{ item.profit_margin.toFixed(1) }}%
              </td>
              <td class="py-3 px-3 border-b border-[var(--border-light)] text-center">
                <span v-if="item.recommendation === 'approve'" class="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-[var(--green-bg)] text-[var(--green)]">建议上架</span>
                <span v-else-if="item.recommendation === 'reject'" class="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-[var(--red-bg)] text-[var(--red)]">不建议</span>
                <span v-else class="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-[var(--amber-bg)] text-[var(--amber)]">谨慎上架</span>
              </td>
              <td class="py-3 px-3 border-b border-[var(--border-light)]">
                <a-button v-if="item.recommendation === 'approve'" size="small" type="primary" ghost @click="handlePublish(item)">上架</a-button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- AI 总结 -->
      <div class="mt-3 px-4 py-3.5 bg-[var(--bg-subtle)] border border-[var(--border-light)] rounded-[6px] flex gap-2.5">
        <div class="w-6 h-6 rounded-[5px] bg-[var(--accent)] flex items-center justify-center text-white text-[11px] font-semibold shrink-0">AI</div>
        <div>
          <div class="text-[12px] font-semibold text-[var(--text-primary)]">G1 驾驶舱 Agent · 综合建议</div>
          <div class="text-[12px] text-[var(--text-secondary)] mt-0.5 leading-relaxed">
            <strong>推荐 {{ bestPlatform }}：</strong>利润率 {{ bestMargin }}%，平台流量大且物流成熟。
            建议优先上架，若追求利润率最大化可考虑多平台同步。
            <span class="text-[var(--accent)] cursor-pointer hover:underline"> 查看 Agent 分析详情 →</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { skuApi, platformApi } from '@/api'
import { calculatePreListingDecision, comparePreListingDecision } from '@/api/modules/decision'
import type { PreListingDecisionRequest, PreListingDecisionResponse, CompareDecisionResponse, CompareItem } from '@/api/modules/decision'

// ====== 基本数据 ======
const now = ref(new Date().toISOString().slice(0, 16).replace('T', ' '))
const searchQuery = ref('')
const selectedSkuId = ref<number | null>(null)
const skuInfo = ref<any>(null)
const skuList = ref<any[]>([])
const loading = ref(false)
const result = ref<PreListingDecisionResponse | null>(null)
const compareResults = ref<CompareItem[]>([])
const selectedChannel = ref('中俄专线')
const platformId = ref<number | undefined>(undefined)

const activeAgents = [
  { label: 'A5 库存', active: true },
  { label: 'A6 利润', active: true },
  { label: 'G3 折扣', active: true },
  { label: 'A2 优化', active: false },
  { label: 'G1 驾驶舱', active: false },
]

const formData = reactive({
  sku_id: null as unknown as number,
  destination_country: 'RU',
  target_sale_price: null as unknown as number,
  platform_fee_pct: 10,
  payment_fee_pct: 3,
  other_fee: 0,
  minimum_margin_pct: 20,
  cargo_type: 'normal',
})

const paymentFeeStr = computed({
  get: () => String(formData.payment_fee_pct),
  set: (v: string) => { formData.payment_fee_pct = parseFloat(v) || 0 },
})

const countryOptions = [
  { label: '俄罗斯 (RU)', value: 'RU' },
  { label: '美国 (US)', value: 'US' },
  { label: '德国 (DE)', value: 'DE' },
  { label: '法国 (FR)', value: 'FR' },
  { label: '英国 (GB)', value: 'GB' },
  { label: '日本 (JP)', value: 'JP' },
  { label: '韩国 (KR)', value: 'KR' },
  { label: '印尼 (ID)', value: 'ID' },
  { label: '泰国 (TH)', value: 'TH' },
  { label: '新加坡 (SG)', value: 'SG' },
  { label: '巴西 (BR)', value: 'BR' },
  { label: '墨西哥 (MX)', value: 'MX' },
]

const platformOptions = ref<{ label: string; value: number }[]>([])

const logisticsOptions = [
  { name: '中俄专线', price: 32.50, duration: '8-12天', weight: '0.16kg', best: true },
  { name: '俄速通', price: 38.00, duration: '5-8天', weight: '0.16kg', best: false },
  { name: '邮政小包', price: 28.00, duration: '15-20天', weight: '0.16kg', best: false },
]

function filterOption(input: string, option: any) {
  return (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
}

// ====== 计算属性 ======
const filteredSkus = computed(() => {
  if (!searchQuery.value) return skuList.value
  const q = searchQuery.value.toLowerCase()
  return skuList.value.filter(s =>
    (s.product_name || '').toLowerCase().includes(q) ||
    (s.code || '').toLowerCase().includes(q)
  )
})

const dimText = computed(() => {
  if (!skuInfo.value) return ''
  const parts: string[] = []
  if (skuInfo.value.sku_length_cm) parts.push(`${skuInfo.value.sku_length_cm}×${skuInfo.value.sku_width_cm}×${skuInfo.value.sku_height_cm}cm`)
  if (skuInfo.value.sku_weight_kg) parts.push(`${skuInfo.value.sku_weight_kg}kg`)
  return parts.join(' · ') || '--'
})

const waterfallItems = computed(() => {
  if (!result.value) return []
  const r = result.value
  const total = Math.max(r.target_sale_price, 1)
  return [
    { label: '售价', amount: `¥${r.target_sale_price.toFixed(2)}`, pct: (r.target_sale_price / total) * 100, type: 'rev' },
    { label: '− 商品成本', amount: `¥${r.product_cost.toFixed(2)}`, pct: (r.product_cost / total) * 100, type: 'cost' },
    { label: '− 运费', amount: `¥${r.shipping_fee.toFixed(2)}`, pct: (r.shipping_fee / total) * 100, type: 'fee' },
    { label: '− 平台费', amount: `¥${r.platform_fee.toFixed(2)}`, pct: (r.platform_fee / total) * 100, type: 'cost' },
    { label: '− 支付费', amount: `¥${r.payment_fee.toFixed(2)}`, pct: (r.payment_fee / total) * 100, type: 'fee' },
    { label: '利润', amount: `¥${r.profit_amount.toFixed(2)}`, pct: (Math.abs(r.profit_amount) / total) * 100, type: 'profit', bold: true },
  ]
})

const marginPct = computed(() => {
  if (!result.value) return 0
  return result.value.profit_margin
})

const resultLabel = computed(() => {
  if (!result.value) return ''
  const map: Record<string, string> = { approve: '建议上架', reject: '不建议上架', needs_data: '资料不完整' }
  return map[result.value.recommendation] || result.value.recommendation
})

const resultBgClass = computed(() => {
  if (!result.value) return ''
  const map: Record<string, string> = { approve: 'bg-[var(--green-bg)] border-[#d1fae5]', reject: 'bg-[var(--red-bg)] border-[#fecaca]', needs_data: 'bg-[var(--amber-bg)] border-[#fde68a]' }
  return map[result.value.recommendation] || ''
})

const resultTextClass = computed(() => {
  if (!result.value) return ''
  const map: Record<string, string> = { approve: 'text-[var(--green)]', reject: 'text-[var(--red)]', needs_data: 'text-[var(--amber)]' }
  return map[result.value.recommendation] || ''
})

const resultDetailClass = computed(() => {
  if (!result.value) return ''
  const map: Record<string, string> = { approve: 'text-[#065f46]', reject: 'text-[#991b1b]', needs_data: 'text-[#92400e]' }
  return map[result.value.recommendation] || ''
})

const aiTitle = computed(() => {
  if (!result.value) return ''
  return result.value.recommendation === 'approve' ? 'A6 Agent · 利润分析' : 'A6 Agent · 风险提示'
})

const aiDescription = computed(() => {
  if (!result.value) return ''
  if (result.value.recommendation === 'approve') {
    return `该商品利润率 ${marginPct.value}%，超出经营目标 ${formData.minimum_margin_pct}%。建议优先上架，定价合理。`
  }
  if (result.value.recommendation === 'reject') {
    return `该商品利润率 ${marginPct.value}%，低于最低要求 ${formData.minimum_margin_pct}%。建议调整定价或选择更低成本的物流方案。`
  }
  return result.value.blocking_reasons?.join('；') || '资料不完整，请先补齐缺失数据。'
})

const sortedCompare = computed(() => {
  return [...compareResults.value].sort((a, b) => b.profit_margin - a.profit_margin)
})

const bestPlatform = computed(() => {
  return sortedCompare.value[0]?.platform_name || ''
})

const bestMargin = computed(() => {
  return sortedCompare.value[0]?.profit_margin.toFixed(1) + '%' || ''
})

function barColor(item: any) {
  const map: Record<string, string> = {
    rev: 'bg-[var(--accent)]',
    cost: 'bg-[var(--text-tertiary)]',
    fee: 'bg-[#a1a1aa]',
    profit: 'bg-[var(--green)]',
  }
  return map[item.type] || 'bg-[var(--text-tertiary)]'
}

function amountColor(item: any) {
  if (item.type === 'profit') return 'text-[var(--green)] font-semibold'
  if (item.type === 'rev') return 'text-[var(--text-primary)] font-semibold'
  return 'text-[var(--text-tertiary)]'
}

// ====== 方法 ======
async function loadSkuList() {
  try {
    skuList.value = [
      { id: 1, product_name: 'TechPro 无线降噪耳机', code: 'TP-EAR-BLK-001', cost_price: 68.00, stock: 235, logistics_ok: true, emoji: '🎧', sku_length_cm: 8, sku_width_cm: 5, sku_height_cm: 3, sku_weight_kg: 0.12 },
      { id: 2, product_name: 'TechPro 无线降噪耳机', code: 'TP-EAR-WHT-001', cost_price: 68.00, stock: 189, logistics_ok: true, emoji: '🎧', sku_length_cm: 8, sku_width_cm: 5, sku_height_cm: 3, sku_weight_kg: 0.12 },
      { id: 3, product_name: 'NatureHome 保温壶 500ml', code: 'NH-BTL-RED-500', cost_price: 35.00, stock: 89, logistics_ok: false, emoji: '🫖', sku_length_cm: null, sku_width_cm: null, sku_height_cm: null, sku_weight_kg: null },
      { id: 4, product_name: 'TechPro 智能手表 S3', code: 'TP-WATCH-BLK-S', cost_price: 158.00, stock: 67, logistics_ok: true, emoji: '⌚', sku_length_cm: 5, sku_width_cm: 4, sku_height_cm: 1.2, sku_weight_kg: 0.08 },
    ]
  } catch (e: any) {
    message.warning('加载商品列表失败')
  }
}

async function selectSku(sku: any) {
  selectedSkuId.value = sku.id
  loading.value = true
  try {
    const resp = await skuApi.getSku(sku.id)
    skuInfo.value = resp.data
    formData.sku_id = sku.id
    if (skuInfo.value?.cost_price) {
      formData.target_sale_price = Number((skuInfo.value.cost_price * 2.5).toFixed(2))
    }
    await calculate()
  } catch (e: any) {
    message.error('查询 SKU 失败：' + (e?.response?.data?.message || e?.message || '未知错误'))
    skuInfo.value = null
  } finally {
    loading.value = false
  }
}

async function calculate() {
  if (!formData.sku_id || !formData.destination_country || !formData.target_sale_price) return
  try {
    const resp = await calculatePreListingDecision({
      ...formData,
      platform_fee_pct: formData.platform_fee_pct,
    })
    result.value = resp.data as unknown as PreListingDecisionResponse
  } catch (e: any) {
    message.error(e?.response?.data?.detail || e?.response?.data?.message || '计算失败')
  }
}

async function handleCompare() {
  if (!skuInfo.value) return
  try {
    const resp = await comparePreListingDecision({
      sku_id: formData.sku_id,
      destination_country: formData.destination_country,
      target_sale_price: formData.target_sale_price,
      platform_ids: platformOptions.value.map(p => p.value),
      payment_fee_pct: formData.payment_fee_pct,
      other_fee: formData.other_fee,
      minimum_margin_pct: formData.minimum_margin_pct,
      cargo_type: formData.cargo_type,
    })
    compareResults.value = (resp.data as unknown as CompareDecisionResponse).results || []
  } catch (e: any) {
    message.error('对比失败')
  }
}

function handleSave() {
  message.success('决策已保存')
}

function handleExport() {
  message.info('导出功能开发中')
}

function handleExportCompare() {
  message.info('导出功能开发中')
}

function handlePublish(item: CompareItem) {
  message.success(`已提交上架 ${item.platform_name}`)
}

// ====== 初始化 ======
onMounted(async () => {
  await loadSkuList()
  try {
    const resp = await platformApi.list()
    const platforms = (resp.data as any)?.items || resp.data || []
    platformOptions.value = (Array.isArray(platforms) ? platforms : []).map((p: any) => ({
      label: p.name || p.platform_name,
      value: p.id,
    }))
  } catch {}
})
</script>
