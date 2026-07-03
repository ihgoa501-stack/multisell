'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

const SOURCE_OPTIONS = [
  { label: 'Excel', value: 'excel' },
  { label: 'CSV', value: 'csv' },
  { label: 'API', value: 'api' },
  { label: '手动', value: 'manual' },
];

const STATUS_OPTIONS = [
  { label: '待处理', value: 'pending' },
  { label: '处理中', value: 'processing' },
  { label: '已完成', value: 'completed' },
  { label: '失败', value: 'failed' },
];

export default function ImportBatchesPage() {
  return (
    <CrudListPage
      resource="/import-batch"
      title="导入批次"
      singular="导入批次"
      searchPlaceholder="搜索文件名..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '来源类型', dataIndex: 'source_type', width: 110 },
        { title: '文件名', dataIndex: 'file_name' },
        { title: '总行数', dataIndex: 'total_rows', width: 100 },
        { title: '成功数', dataIndex: 'success_count', width: 100 },
        { title: '失败数', dataIndex: 'error_count', width: 100 },
        { title: '状态', dataIndex: 'status', width: 110 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'source_type', label: '来源类型', type: 'select', options: SOURCE_OPTIONS },
        { name: 'file_name', label: '文件名', required: true },
        { name: 'status', label: '状态', type: 'select', options: STATUS_OPTIONS },
      ]}
    />
  );
}
