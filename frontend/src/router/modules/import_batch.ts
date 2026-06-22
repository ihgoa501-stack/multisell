import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'import-batch',
    name: 'ImportBatch',
    component: () => import('@/views/import_batch/ImportBatch.vue'),
    meta: { title: '导入管理', icon: 'download', menu: true, perm: 'import:view' },
  },
]
