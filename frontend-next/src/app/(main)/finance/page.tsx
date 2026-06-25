'use client';

import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

export default function FinancePage() {
  return (
    <CrudListPage
      resource="/finance/accounts"
      title="财务账户"
      singular="财务账户"
      searchPlaceholder="搜索账户名称 / 类型..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '账户名称', dataIndex: 'name', width: 200 },
        { title: '账户类型', dataIndex: 'account_type', width: 130 },
        { title: '平台ID', dataIndex: 'platform_id', width: 90 },
        { title: '币种', dataIndex: 'currency', width: 90 },
        { title: '余额', dataIndex: 'balance', width: 130, render: fmtMoney },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'name', label: '账户名称', required: true },
        { name: 'account_type', label: '账户类型', required: true },
        { name: 'platform_id', label: '平台ID', type: 'number' },
        { name: 'currency', label: '币种', initialValue: 'CNY' },
        { name: 'balance', label: '余额', type: 'number', initialValue: 0 },
        { name: 'status', label: '状态', initialValue: 'active' },
      ]}
    />
  );
}
