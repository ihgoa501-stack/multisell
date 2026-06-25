'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function PlatformIntegrationsPage() {
  return (
    <CrudListPage
      resource="/platform-integrations"
      title="平台对接"
      singular="平台对接"
      searchPlaceholder="搜索店铺名称 / 账号ID..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '店铺名称', dataIndex: 'store_name', width: 200 },
        { title: '账号ID', dataIndex: 'account_id', width: 160 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '同步状态', dataIndex: 'sync_status', width: 110 },
        { title: '最近同步', dataIndex: 'last_sync_at', width: 160, render: fmtDate },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'store_name', label: '店铺名称', required: true },
        { name: 'account_id', label: '账号ID', required: true },
        { name: 'status', label: '状态', initialValue: 'active' },
        { name: 'sync_status', label: '同步状态' },
      ]}
    />
  );
}
