<template>
  <div>
    <n-page-header subtitle="通过Excel批量导入商品/SKU/价格/库存">
      <template #title>📥 导入管理</template>
    </n-page-header>

    <!-- ── 上传导入文件 ── -->
    <n-card title="上传导入文件" :bordered="false" style="margin-top: 12px">
      <n-space vertical>
        <n-space align="center">
          <n-select
            v-model:value="importType"
            :options="importTypeOptions"
            placeholder="选择导入类型"
            style="width: 200px"
            clearable
          />
          <n-button :disabled="!importType" @click="downloadTemplate">
            下载模板
          </n-button>
        </n-space>

        <n-upload
          ref="uploadRef"
          :default-upload="false"
          accept=".xlsx,.xls"
          :max="1"
          :disabled="!importType || uploading"
          @change="handleFileChange"
        >
          <n-upload-dragger>
            <div style="padding: 32px 0">
              <n-icon size="48" color="#18a058">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="17 8 12 3 7 8"/>
                  <line x1="12" y1="3" x2="12" y2="15"/>
                </svg>
              </n-icon>
              <p style="margin-top: 12px; font-size: 14px; color: #666;">
                点击或拖拽 Excel 文件到此区域上传
              </p>
              <p style="font-size: 12px; color: #999; margin-top: 4px;">
                仅支持 .xlsx 和 .xls 文件
              </p>
            </div>
          </n-upload-dragger>
        </n-upload>
      </n-space>
    </n-card>

    <!-- ── 预览 ── -->
    <n-card
      v-if="previewResult"
      title="预览"
      :bordered="false"
      style="margin-top: 12px"
      :segmented="{ content: true }"
    >
      <n-space vertical size="large">
        <!-- 统计 -->
        <n-space>
          <n-statistic label="总行数" :value="previewResult.total_rows" />
          <n-statistic label="有效" :value="previewResult.valid_rows">
            <template #suffix>条</template>
          </n-statistic>
          <n-statistic label="错误" :value="previewResult.error_rows">
            <template #suffix>条</template>
          </n-statistic>
        </n-space>

        <!-- 行预览表格 -->
        <n-data-table
          v-if="previewRows.length > 0"
          :columns="previewColumns"
          :data="previewRows"
          :max-height="400"
          striped
          size="small"
          :bordered="false"
        />

        <!-- 执行/结果 -->
        <template v-if="!commitResult">
          <n-space>
            <n-button
              type="primary"
              :loading="committing"
              :disabled="previewResult.error_rows > 0"
              @click="handleCommit"
            >
              确认执行（{{ previewResult.valid_rows }}条有效，{{ previewResult.error_rows }}条错误）
            </n-button>
          </n-space>
        </template>

        <template v-else>
          <n-alert
            :type="commitResult.error_count > 0 ? 'warning' : 'success'"
            :show-icon="true"
            closable
            @close="resetPreview"
          >
            执行完成：成功 {{ commitResult.success_count }} 条，失败 {{ commitResult.error_count }} 条
          </n-alert>
          <n-space>
            <n-button @click="resetPreview">继续导入</n-button>
            <n-button
              v-if="commitResult.error_count > 0"
              @click="downloadErrors(commitResult.batch_id)"
            >
              下载错误报告
            </n-button>
          </n-space>
        </template>
      </n-space>
    </n-card>

    <!-- ── 导入批次列表 ── -->
    <n-card title="导入批次" :bordered="false" style="margin-top: 12px">
      <template #header-extra>
        <n-button size="small" @click="fetchBatches">刷新</n-button>
      </template>

      <n-data-table
        :columns="batchColumns"
        :data="batches"
        :loading="loadingBatches"
        :pagination="batchPagination"
        @update:page="onBatchPageChange"
        striped
        :bordered="false"
      />
    </n-card>

    <!-- ── 批次详情弹窗 ── -->
    <n-modal
      v-model:show="showDetailModal"
      title="批次详情"
      preset="card"
      style="width: 600px"
      :segmented="{ content: true }"
    >
      <n-descriptions
        v-if="currentBatch"
        label-placement="left"
        bordered
        :column="1"
        size="small"
      >
        <n-descriptions-item label="ID">{{ currentBatch.id }}</n-descriptions-item>
        <n-descriptions-item label="导入类型">
          {{ typeLabelMap[currentBatch.type] || currentBatch.type }}
        </n-descriptions-item>
        <n-descriptions-item label="文件名">
          {{ currentBatch.file_name || '-' }}
        </n-descriptions-item>
        <n-descriptions-item label="状态">
          <n-tag :type="statusType(currentBatch.status)" size="small">
            {{ statusLabel(currentBatch.status) }}
          </n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="总行数">{{ currentBatch.total_rows }}</n-descriptions-item>
        <n-descriptions-item label="成功">
          <span style="color: #18a058">{{ currentBatch.success_count }}</span>
        </n-descriptions-item>
        <n-descriptions-item label="失败">
          <span style="color: #d03050">{{ currentBatch.error_count }}</span>
        </n-descriptions-item>
        <n-descriptions-item label="备注">
          {{ currentBatch.error_summary || '-' }}
        </n-descriptions-item>
        <n-descriptions-item label="创建人">
          {{ currentBatch.created_by || '-' }}
        </n-descriptions-item>
        <n-descriptions-item label="创建时间">
          {{ formatTime(currentBatch.created_at) }}
        </n-descriptions-item>
      </n-descriptions>

      <template #footer>
        <n-space justify="end">
          <n-button
            v-if="currentBatch?.error_count > 0"
            @click="downloadErrors(currentBatch.id)"
          >
            下载错误报告
          </n-button>
          <n-button @click="showDetailModal = false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, computed, onMounted } from 'vue'
import { NTag, NButton, NSpace, useMessage } from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
import { importBatchApi } from '@/api/modules/importBatch'

const message = useMessage()

// ── 导入类型 ──────────────────────────────────────────────────────

const importType = ref<string | null>(null)
const importTypeOptions = [
  { label: '商品导入 (product)', value: 'product' },
  { label: 'SKU导入 (sku)', value: 'sku' },
  { label: '价格导入 (price)', value: 'price' },
  { label: '库存导入 (inventory)', value: 'inventory' },
]
const typeLabelMap: Record<string, string> = {
  product: '商品导入',
  sku: 'SKU导入',
  price: '价格导入',
  inventory: '库存导入',
}

// ── 上传 & 预览 ──────────────────────────────────────────────────

const uploadRef = ref<any>(null)
const uploading = ref(false)
const previewResult = ref<any>(null)

// ── 提交执行 ─────────────────────────────────────────────────────

const committing = ref(false)
const commitResult = ref<any>(null)

// ── 批次列表 ─────────────────────────────────────────────────────

const batches = ref<any[]>([])
const loadingBatches = ref(false)
const batchPagination = ref({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  onChange: (p: number) => {
    batchPagination.value.page = p
    fetchBatches()
  },
})

// ── 详情弹窗 ─────────────────────────────────────────────────────

const showDetailModal = ref(false)
const currentBatch = ref<any>(null)

// ── 预览行计算 ──────────────────────────────────────────────────

const previewRows = computed(() => {
  if (!previewResult.value) return []
  const errorMap = new Map<number, string>()
  for (const err of previewResult.value.errors || []) {
    errorMap.set(err.row_index, err.error_message)
  }
  const rows: any[] = []
  const total = previewResult.value.total_rows || 0
  for (let i = 0; i < total; i++) {
    const rowIndex = i + 2
    const errMsg = errorMap.get(rowIndex)
    rows.push({
      row_index: rowIndex,
      status: errMsg ? 'error' : 'pending',
      error_message: errMsg || '',
    })
  }
  return rows
})

// ── 表格列定义 ───────────────────────────────────────────────────

const previewColumns = [
  { title: '行号', key: 'row_index', width: 80 },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: any) => {
      const isError = row.status === 'error'
      return h(NTag, {
        type: isError ? 'error' : 'success',
        size: 'small',
      }, { default: () => isError ? '错误' : '等待执行' })
    },
  },
  { title: '错误信息', key: 'error_message', ellipsis: { tooltip: true } },
]

function statusType(status: string): 'warning' | 'info' | 'success' | 'error' | 'default' {
  const map: Record<string, 'warning' | 'info' | 'success' | 'error'> = {
    pending: 'warning',
    previewed: 'info',
    committed: 'success',
    failed: 'error',
  }
  return map[status] || 'default'
}

function statusLabel(status: string): string {
  const map: Record<string, string> = {
    pending: '待处理',
    previewed: '已预览',
    committed: '已执行',
    failed: '失败',
  }
  return map[status] || status
}

const batchColumns = [
  { title: 'ID', key: 'id', width: 70 },
  {
    title: '类型',
    key: 'type',
    width: 110,
    render: (r: any) => typeLabelMap[r.type] || r.type,
  },
  {
    title: '文件名',
    key: 'file_name',
    ellipsis: { tooltip: true },
    render: (r: any) => r.file_name || '-',
  },
  { title: '总行数', key: 'total_rows', width: 70 },
  {
    title: '成功',
    key: 'success_count',
    width: 70,
    render: (r: any) => h('span', { style: 'color: #18a058' }, r.success_count),
  },
  {
    title: '失败',
    key: 'error_count',
    width: 70,
    render: (r: any) => h('span', { style: 'color: #d03050' }, r.error_count),
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (r: any) => h(NTag, {
      type: statusType(r.status),
      size: 'small',
    }, { default: () => statusLabel(r.status) }),
  },
  {
    title: '创建人',
    key: 'created_by',
    width: 100,
    render: (r: any) => r.created_by || '-',
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 170,
    render: (r: any) => formatTime(r.created_at),
  },
  {
    title: '操作',
    key: 'actions',
    width: 180,
    fixed: 'right',
    render: (r: any) => h(NSpace, { justify: 'center' }, {
      default: () => [
        h(NButton, { size: 'small', onClick: () => viewBatchDetail(r) }, { default: () => '详情' }),
        r.error_count > 0
          ? h(NButton, {
            size: 'small',
            onClick: () => downloadErrors(r.id),
          }, { default: () => '错误报告' })
          : null,
      ],
    }),
  },
]

// ── 文件上传 ─────────────────────────────────────────────────────

function handleFileChange(data: { file: UploadFileInfo; fileList: UploadFileInfo[] }) {
  if (data.file.status === 'pending' && data.file.file) {
    uploadFile(data.file.file)
  }
}

async function uploadFile(file: File) {
  if (!importType.value) {
    message.warning('请先选择导入类型')
    return
  }
  uploading.value = true
  commitResult.value = null
  previewResult.value = null
  try {
    const res = await importBatchApi.uploadPreview(file, importType.value) as any
    previewResult.value = res.data
    message.success(
      `解析完成：${res.data?.valid_rows ?? 0}条有效，${res.data?.error_rows ?? 0}条错误`,
    )
  } catch (e: any) {
    message.error('上传失败：' + (e.message || ''))
  } finally {
    uploading.value = false
  }
}

// ── 模板下载 ─────────────────────────────────────────────────────

async function downloadTemplate() {
  if (!importType.value) return
  try {
    const blob = await importBatchApi.downloadTemplate(importType.value) as Blob
    if (blob.type?.includes('json')) {
      const text = await blob.text()
      const err = JSON.parse(text)
      message.error(err.message || '下载失败')
      return
    }
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${importType.value}_template.xlsx`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e: any) {
    message.error('下载失败：' + (e.message || ''))
  }
}

// ── 错误报告下载 ────────────────────────────────────────────────

async function downloadErrors(batchId: number) {
  try {
    const blob = await importBatchApi.downloadErrors(batchId) as Blob
    if (blob.type?.includes('json')) {
      const text = await blob.text()
      const err = JSON.parse(text)
      message.error(err.message || '下载失败')
      return
    }
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `import_batch_${batchId}_errors.xlsx`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e: any) {
    message.error('下载失败：' + (e.message || ''))
  }
}

// ── 提交执行 ─────────────────────────────────────────────────────

async function handleCommit() {
  if (!previewResult.value) return
  committing.value = true
  try {
    const res = await importBatchApi.commitBatch(previewResult.value.batch_id) as any
    commitResult.value = res.data
    message.success(
      `执行完成：${res.data?.success_count ?? 0}条成功`,
    )
    fetchBatches()
  } catch (e: any) {
    message.error('执行失败：' + (e.message || ''))
  } finally {
    committing.value = false
  }
}

function resetPreview() {
  previewResult.value = null
  commitResult.value = null
}

// ── 批次列表 ─────────────────────────────────────────────────────

async function fetchBatches() {
  loadingBatches.value = true
  try {
    const res = await importBatchApi.listBatches({
      page: batchPagination.value.page,
      page_size: batchPagination.value.pageSize,
    }) as any
    const body = res.data
    batches.value = body?.records ?? body?.data?.records ?? []
    batchPagination.value.itemCount = body?.total ?? body?.data?.total ?? 0
  } catch {
    message.error('加载批次列表失败')
  } finally {
    loadingBatches.value = false
  }
}

function onBatchPageChange(p: number) {
  batchPagination.value.page = p
  fetchBatches()
}

// ── 详情弹窗 ─────────────────────────────────────────────────────

function viewBatchDetail(batch: any) {
  currentBatch.value = batch
  showDetailModal.value = true
}

function formatTime(t: string | null | undefined): string {
  return t ? t.slice(0, 19).replace('T', ' ') : '-'
}

// ── 初始化 ───────────────────────────────────────────────────────

onMounted(fetchBatches)
</script>
