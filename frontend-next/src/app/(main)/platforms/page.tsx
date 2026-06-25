'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function PlatformsPage() {
  return (
    <CrudListPage
      resource="/platforms"
      title="平台"
      singular="平台"
      searchPlaceholder="搜索平台名称 / 编码..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '平台名称', dataIndex: 'name', width: 180 },
        { title: '编码', dataIndex: 'code', width: 120 },
        { title: '国家', dataIndex: 'country', width: 120 },
        { title: '配置', dataIndex: 'config', width: 220 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'name', label: '平台名称', required: true },
        { name: 'code', label: '编码' },
        { name: 'country', label: '国家' },
        { name: 'config', label: '配置', type: 'textarea' },
        { name: 'status', label: '状态', initialValue: 'active' },
      ]}
    />
  );
}
