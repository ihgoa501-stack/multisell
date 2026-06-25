'use client';

import CrudListPage from '@/components/crud/CrudListPage';

const STATUS_OPTIONS = [
  { label: '启用', value: 'active' },
  { label: '停用', value: 'inactive' },
];

export default function SettingsRbacPage() {
  return (
    <CrudListPage
      resource="/rbac/roles"
      title="RBAC 角色管理"
      singular="角色"
      searchPlaceholder="搜索角色..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '角色名', dataIndex: 'name', width: 180 },
        { title: '描述', dataIndex: 'description' },
        { title: '状态', dataIndex: 'status', width: 120 },
      ]}
      fields={[
        { name: 'name', label: '角色名', required: true },
        { name: 'description', label: '描述', type: 'textarea' },
        { name: 'status', label: '状态', type: 'select', options: STATUS_OPTIONS },
      ]}
    />
  );
}
