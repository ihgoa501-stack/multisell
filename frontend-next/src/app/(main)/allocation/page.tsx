'use client';

import CrudListPage from '@/components/crud/CrudListPage';

const TYPE_OPTIONS = [
  { label: '人工', value: 'manual' },
  { label: '自动', value: 'auto' },
  { label: '系统', value: 'system' },
];

const STATUS_OPTIONS = [
  { label: '启用', value: 'active' },
  { label: '停用', value: 'inactive' },
];

export default function AllocationPage() {
  return (
    <CrudListPage
      resource="/allocation"
      title="分摊规则"
      singular="分摊规则"
      searchPlaceholder="搜索..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '名称', dataIndex: 'name' },
        { title: '类型', dataIndex: 'type', width: 120 },
        { title: '状态', dataIndex: 'status', width: 120 },
      ]}
      fields={[
        { name: 'name', label: '名称', required: true },
        { name: 'type', label: '类型', type: 'select', options: TYPE_OPTIONS },
        { name: 'status', label: '状态', type: 'select', options: STATUS_OPTIONS },
      ]}
    />
  );
}
