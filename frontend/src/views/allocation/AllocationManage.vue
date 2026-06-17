<template>
  <div>
    <n-page-header subtitle="多仓库库存管理与分配">
      <template #title>🏭 库存分配</template>
      <template #extra>
        <n-button @click="handleGenerateMock">🎲 初始化模拟数据</n-button>
      </template>
    </n-page-header>

    <n-tabs type="line" default-value="warehouses" style="margin-top: 12px;">
      <!-- 仓库管理 -->
      <n-tab-pane name="warehouses" tab="仓库管理">
        <n-space style="margin-bottom: 12px;">
          <n-button type="primary" @click="showWarehouseModal = true">＋ 新建仓库</n-button>
        </n-space>
        <n-data-table :columns="whColumns" :data="warehouses" :loading="loading" />
      </n-tab-pane>

      <!-- 分配规则 -->
      <n-tab-pane name="rules" tab="分配规则">
        <n-space style="margin-bottom: 12px;">
          <n-button type="primary" @click="showRuleModal = true">＋ 新建规则</n-button>
        </n-space>
        <n-data-table :columns="ruleColumns" :data="rules" :loading="loading" />
      </n-tab-pane>

      <!-- 库存分布 -->
      <n-tab-pane name="inventory" tab="库存分布">
        <n-space style="margin-bottom: 12px;">
          <n-input v-model:value="querySkuId" placeholder="SKU ID" type="number" style="width: 120px;" />
          <n-button @click="fetchInventory">查询</n-button>
          <n-button v-if="inventoryData.length" @click="handleAutoAllocate">自动分配</n-button>
        </n-space>
        <n-data-table v-if="inventoryData.length" :columns="invColumns" :data="inventoryData" />
        <n-empty v-else description="输入SKU ID查询库存分布" />
      </n-tab-pane>
    </n-tabs>

    <!-- 仓库弹窗 -->
    <n-modal v-model:show="showWarehouseModal" title="新建仓库" preset="card" style="width: 450px;">
      <n-form :model="whForm" label-placement="left" label-width="100px">
        <n-form-item label="名称" required><n-input v-model:value="whForm.name" /></n-form-item>
        <n-form-item label="编码"><n-input v-model:value="whForm.code" /></n-form-item>
        <n-form-item label="地址"><n-input v-model:value="whForm.address" /></n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showWarehouseModal = false">取消</n-button>
          <n-button type="primary" @click="handleCreateWarehouse">创建</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 规则弹窗 -->
    <n-modal v-model:show="showRuleModal" title="新建分配规则" preset="card" style="width: 450px;">
      <n-form :model="ruleForm" label-placement="left" label-width="110px">
        <n-form-item label="规则名称" required><n-input v-model:value="ruleForm.name" /></n-form-item>
        <n-form-item label="仓库" required>
          <n-select v-model:value="ruleForm.warehouse_id" :options="whOptions" />
        </n-form-item>
        <n-form-item label="规则类型">
          <n-select v-model:value="ruleForm.rule_type" :options="[
            { label: '百分比分配', value: 'percentage' },
            { label: '固定数量', value: 'fixed' },
            { label: '优先分配', value: 'priority' },
          ]" />
        </n-form-item>
        <n-form-item label="分配百分比" v-if="ruleForm.rule_type === 'percentage'">
          <n-input-number v-model:value="ruleForm.allocation_pct" :min="0" :max="100" style="width: 120px;" /> %
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showRuleModal = false">取消</n-button>
          <n-button type="primary" @click="handleCreateRule">创建</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, computed, onMounted } from 'vue'
import { NButton, NTag, useMessage } from 'naive-ui'
import { allocationApi } from '@/api/modules/allocation'

const message = useMessage()
const loading = ref(false)
const warehouses = ref<any[]>([])
const rules = ref<any[]>([])
const inventoryData = ref<any[]>([])
const querySkuId = ref('')
const showWarehouseModal = ref(false)
const showRuleModal = ref(false)

const whForm = ref({ name: '', code: '', address: '' })
const ruleForm = ref({ name: '', warehouse_id: null, rule_type: 'percentage', allocation_pct: 100 })

const whOptions = computed(() => warehouses.value.map((w: any) => ({ label: w.name, value: w.id })))

const whColumns = [
  { title: '仓库名称', key: 'name' },
  { title: '编码', key: 'code', render: (r: any) => r.code || '-' },
  { title: '地址', key: 'address', render: (r: any) => r.address || '-' },
  { title: '默认', key: 'is_default', width: 70, render: (r: any) => r.is_default ? '✅' : '' },
  { title: '状态', key: 'status', width: 70, render: (r: any) => r.status === 1 ? '启用' : '禁用' },
]

const ruleColumns = [
  { title: '规则名称', key: 'name' },
  { title: '优先级', key: 'priority', width: 70 },
  { title: '类型', key: 'rule_type', width: 100, render: (r: any) => ({ percentage: '百分比', fixed: '固定数量', priority: '优先分配' })[r.rule_type] ?? r.rule_type },
  { title: '仓库', key: 'warehouse_name' },
  { title: '分配比例', key: 'allocation_pct', width: 90, render: (r: any) => r.rule_type === 'percentage' ? `${r.allocation_pct}%` : '-' },
]

const invColumns = [
  { title: '仓库', key: 'warehouse_name' },
  { title: '库存量', key: 'quantity' },
  { title: '锁定', key: 'locked_quantity' },
  { title: '可用', key: 'available_qty', render: (r: any) => h('span', { style: `color: ${r.available_qty > 0 ? '#18a058' : '#d03050'}` }, r.available_qty) },
  { title: '安全库存', key: 'safety_stock' },
]

async function fetchWarehouses() {
  try { const res = await allocationApi.listWarehouses(); warehouses.value = res.data?.data ?? [] } catch {}
}
async function fetchRules() {
  try { const res = await allocationApi.listRules(); rules.value = res.data?.data ?? [] } catch {}
}
async function fetchInventory() {
  if (!querySkuId.value) return
  try {
    const res = await allocationApi.getWarehouseInventory(Number(querySkuId.value))
    inventoryData.value = res.data?.data ?? []
  } catch { message.error('查询失败') }
}
async function handleCreateWarehouse() {
  try { await allocationApi.createWarehouse(whForm.value); showWarehouseModal.value = false; fetchWarehouses(); message.success('创建成功') } catch { message.error('创建失败') }
}
async function handleCreateRule() {
  try { await allocationApi.createRule(ruleForm.value); showRuleModal.value = false; fetchRules(); message.success('规则已创建') } catch { message.error('创建失败') }
}
async function handleAutoAllocate() {
  if (!querySkuId.value) return
  try {
    const res = await allocationApi.autoAllocate(Number(querySkuId.value))
    message.success('自动分配完成')
    fetchInventory()
  } catch { message.error('分配失败：库存不足或已分配') }
}
async function handleGenerateMock() {
  try { const res = await allocationApi.generateMock(); message.success('模拟数据已生成'); fetchWarehouses(); fetchRules() } catch { message.error('生成失败') }
}

onMounted(async () => { await fetchWarehouses(); await fetchRules() })
</script>
