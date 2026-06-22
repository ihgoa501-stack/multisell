<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px;">
      <div>
        <h3 style="margin: 0;">库存分配</h3>
        <span style="color: var(--ant-color-text-secondary);">多仓库库存管理与分配</span>
      </div>
      <a-button @click="handleGenerateMock">初始化模拟数据</a-button>
    </div>

    <a-tabs v-model:activeKey="activeTab" style="margin-top: 12px;">
      <!-- 仓库管理 -->
      <a-tab-pane key="warehouses" tab="仓库管理">
        <a-space style="margin-bottom: 12px;">
          <a-button type="primary" @click="showWarehouseModal = true">新建仓库</a-button>
        </a-space>
        <a-table
          :columns="whColumns"
          :data-source="warehouses"
          :loading="loading"
          :pagination="false"
          row-key="id"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'code'">
              {{ record.code || '-' }}
            </template>
            <template v-else-if="column.dataIndex === 'address'">
              {{ record.address || '-' }}
            </template>
            <template v-else-if="column.dataIndex === 'is_default'">
              <span v-if="record.is_default" style="color: var(--ant-color-success);">是</span>
            </template>
            <template v-else-if="column.dataIndex === 'status'">
              {{ record.status === 1 ? '启用' : '禁用' }}
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <!-- 分配规则 -->
      <a-tab-pane key="rules" tab="分配规则">
        <a-space style="margin-bottom: 12px;">
          <a-button type="primary" @click="showRuleModal = true">新建规则</a-button>
        </a-space>
        <a-table
          :columns="ruleColumns"
          :data-source="rules"
          :loading="loading"
          :pagination="false"
          row-key="id"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'rule_type'">
              {{ ({ percentage: '百分比', fixed: '固定数量', priority: '优先分配' } as Record<string, string>)[record.rule_type] ?? record.rule_type }}
            </template>
            <template v-else-if="column.dataIndex === 'allocation_pct'">
              {{ record.rule_type === 'percentage' ? `${record.allocation_pct}%` : '-' }}
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <!-- 库存分布 -->
      <a-tab-pane key="inventory" tab="库存分布">
        <a-space style="margin-bottom: 12px;">
          <a-input v-model:value="querySkuId" placeholder="SKU ID" style="width: 120px;" />
          <a-button @click="fetchInventory">查询</a-button>
          <a-button v-if="inventoryData.length" @click="handleAutoAllocate">自动分配</a-button>
        </a-space>
        <a-table
          v-if="inventoryData.length"
          :columns="invColumns"
          :data-source="inventoryData"
          :pagination="false"
          row-key="warehouse_name"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'available_qty'">
              <span :style="{ color: record.available_qty > 0 ? 'var(--ant-color-success)' : 'var(--ant-color-error)' }">
                {{ record.available_qty }}
              </span>
            </template>
          </template>
        </a-table>
        <a-empty v-else description="输入SKU ID查询库存分布" />
      </a-tab-pane>
    </a-tabs>

    <!-- 仓库弹窗 -->
    <a-modal v-model:open="showWarehouseModal" title="新建仓库" :footer="null" style="width: 450px;">
      <a-form :model="whForm" layout="horizontal" :label-col="{ style: { width: '100px' } }">
        <a-form-item label="名称" required><a-input v-model:value="whForm.name" /></a-form-item>
        <a-form-item label="编码"><a-input v-model:value="whForm.code" /></a-form-item>
        <a-form-item label="地址"><a-input v-model:value="whForm.address" /></a-form-item>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="showWarehouseModal = false">取消</a-button>
          <a-button type="primary" @click="handleCreateWarehouse">创建</a-button>
        </a-space>
      </template>
    </a-modal>

    <!-- 规则弹窗 -->
    <a-modal v-model:open="showRuleModal" title="新建分配规则" :footer="null" style="width: 450px;">
      <a-form :model="ruleForm" layout="horizontal" :label-col="{ style: { width: '110px' } }">
        <a-form-item label="规则名称" required><a-input v-model:value="ruleForm.name" /></a-form-item>
        <a-form-item label="仓库" required>
          <a-select v-model:value="ruleForm.warehouse_id" :options="whOptions" />
        </a-form-item>
        <a-form-item label="规则类型">
          <a-select v-model:value="ruleForm.rule_type" :options="[
            { label: '百分比分配', value: 'percentage' },
            { label: '固定数量', value: 'fixed' },
            { label: '优先分配', value: 'priority' },
          ]" />
        </a-form-item>
        <a-form-item label="分配百分比" v-if="ruleForm.rule_type === 'percentage'">
          <a-input-number v-model:value="ruleForm.allocation_pct" :min="0" :max="100" style="width: 120px;" /> %
        </a-form-item>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="showRuleModal = false">取消</a-button>
          <a-button type="primary" @click="handleCreateRule">创建</a-button>
        </a-space>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { allocationApi } from '@/api/modules/allocation'

const activeTab = ref('warehouses')
const loading = ref(false)
const warehouses = ref<any[]>([])
const rules = ref<any[]>([])
const inventoryData = ref<any[]>([])
const querySkuId = ref('')
const showWarehouseModal = ref(false)
const showRuleModal = ref(false)

const whForm = ref({ name: '', code: '', address: '' })
const ruleForm = ref({ name: '', warehouse_id: null as number | null, rule_type: 'percentage', allocation_pct: 100 })

const whOptions = computed(() => warehouses.value.map((w: any) => ({ label: w.name, value: w.id })))

const whColumns = [
  { title: '仓库名称', dataIndex: 'name', key: 'name' },
  { title: '编码', dataIndex: 'code', key: 'code' },
  { title: '地址', dataIndex: 'address', key: 'address' },
  { title: '默认', dataIndex: 'is_default', key: 'is_default', width: 70 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 70 },
]

const ruleColumns = [
  { title: '规则名称', dataIndex: 'name', key: 'name' },
  { title: '优先级', dataIndex: 'priority', key: 'priority', width: 70 },
  { title: '类型', dataIndex: 'rule_type', key: 'rule_type', width: 100 },
  { title: '仓库', dataIndex: 'warehouse_name', key: 'warehouse_name' },
  { title: '分配比例', dataIndex: 'allocation_pct', key: 'allocation_pct', width: 90 },
]

const invColumns = [
  { title: '仓库', dataIndex: 'warehouse_name', key: 'warehouse_name' },
  { title: '库存量', dataIndex: 'quantity', key: 'quantity' },
  { title: '锁定', dataIndex: 'locked_quantity', key: 'locked_quantity' },
  { title: '可用', dataIndex: 'available_qty', key: 'available_qty' },
  { title: '安全库存', dataIndex: 'safety_stock', key: 'safety_stock' },
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
    await allocationApi.autoAllocate(Number(querySkuId.value))
    message.success('自动分配完成')
    fetchInventory()
  } catch { message.error('分配失败：库存不足或已分配') }
}
async function handleGenerateMock() {
  try { await allocationApi.generateMock(); message.success('模拟数据已生成'); fetchWarehouses(); fetchRules() } catch { message.error('生成失败') }
}

onMounted(async () => { await fetchWarehouses(); await fetchRules() })
</script>
