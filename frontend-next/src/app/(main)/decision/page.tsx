'use client';

import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

export default function DecisionPage() {
  return (
    <CrudListPage
      resource="/decision"
      title="上架前决策"
      singular="决策"
      searchPlaceholder="搜索 SKU ID / 平台 / 风险等级..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: 'SKU ID', dataIndex: 'sku_id', width: 90 },
        { title: '平台ID', dataIndex: 'platform_id', width: 90 },
        { title: '国家', dataIndex: 'country_code', width: 90 },
        { title: '预估利润', dataIndex: 'estimated_profit', width: 120, render: fmtMoney },
        { title: '利润率(%)', dataIndex: 'profit_margin', width: 110 },
        { title: '风险等级', dataIndex: 'risk_level', width: 100 },
        { title: '建议', dataIndex: 'recommendation', width: 150 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'sku_id', label: 'SKU ID', type: 'number', required: true },
        { name: 'platform_id', label: '平台ID', type: 'number', required: true },
        { name: 'country_code', label: '国家' },
        { name: 'estimated_profit', label: '预估利润', type: 'number' },
        { name: 'profit_margin', label: '利润率(%)', type: 'number' },
        { name: 'risk_level', label: '风险等级' },
        { name: 'recommendation', label: '建议' },
        { name: 'status', label: '状态', initialValue: 'pending' },
      ]}
    />
  );
}
