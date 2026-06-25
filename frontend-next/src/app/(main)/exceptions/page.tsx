'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

const SEVERITY_OPTIONS = [
  { label: '低', value: 'low' },
  { label: '中', value: 'medium' },
  { label: '高', value: 'high' },
  { label: '严重', value: 'critical' },
];

const STATUS_OPTIONS = [
  { label: '待处理', value: 'open' },
  { label: '已指派', value: 'assigned' },
  { label: '已解决', value: 'resolved' },
  { label: '已忽略', value: 'ignored' },
];

export default function ExceptionsPage() {
  return (
    <CrudListPage
      resource="/exceptions"
      title="异常"
      singular="异常"
      searchPlaceholder="搜索异常标题..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '标题', dataIndex: 'title' },
        { title: '来源模块', dataIndex: 'source_module', width: 140 },
        { title: '严重程度', dataIndex: 'severity', width: 110 },
        { title: '状态', dataIndex: 'status', width: 110 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'title', label: '标题', required: true },
        { name: 'source_module', label: '来源模块' },
        { name: 'severity', label: '严重程度', type: 'select', options: SEVERITY_OPTIONS },
        { name: 'status', label: '状态', type: 'select', options: STATUS_OPTIONS },
        { name: 'description', label: '描述', type: 'textarea' },
      ]}
    />
  );
}
