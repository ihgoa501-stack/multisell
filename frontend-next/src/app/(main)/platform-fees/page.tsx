'use client';

import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

export default function PlatformFeesPage() {
  return (
    <CrudListPage
      resource="/platform-fee"
      title="平台费用规则"
      singular="平台费用规则"
      searchPlaceholder="搜索费用类型 / 平台..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '平台ID', dataIndex: 'platform_id', width: 90 },
        { title: '费用类型', dataIndex: 'fee_type', width: 130 },
        { title: '费率(%)', dataIndex: 'fee_rate_pct', width: 110 },
        { title: '固定费用', dataIndex: 'fixed_amount', width: 120, render: fmtMoney },
        { title: '币种', dataIndex: 'currency', width: 90 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'platform_id', label: '平台ID', type: 'number', required: true },
        { name: 'fee_type', label: '费用类型', required: true },
        { name: 'fee_rate_pct', label: '费率(%)', type: 'number' },
        { name: 'fixed_amount', label: '固定费用', type: 'number' },
        { name: 'currency', label: '币种', initialValue: 'CNY' },
        { name: 'status', label: '状态', initialValue: 'active' },
      ]}
    />
  );
}
