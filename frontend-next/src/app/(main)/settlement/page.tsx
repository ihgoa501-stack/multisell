'use client';

import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

export default function SettlementPage() {
  return (
    <CrudListPage
      resource="/settlement"
      title="结算"
      singular="结算单"
      searchPlaceholder="搜索结算单号 / 平台..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '平台ID', dataIndex: 'platform_id', width: 90 },
        { title: '结算单号', dataIndex: 'settlement_no', width: 180 },
        { title: '账期开始', dataIndex: 'period_start', width: 130, render: fmtDate },
        { title: '账期结束', dataIndex: 'period_end', width: 130, render: fmtDate },
        { title: '净额合计', dataIndex: 'total_net', width: 130, render: fmtMoney },
        { title: '状态', dataIndex: 'status', width: 110 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'platform_id', label: '平台ID', type: 'number', required: true },
        { name: 'settlement_no', label: '结算单号', required: true },
        { name: 'period_start', label: '账期开始' },
        { name: 'period_end', label: '账期结束' },
        { name: 'total_net', label: '净额合计', type: 'number' },
        { name: 'status', label: '状态', initialValue: 'pending' },
      ]}
    />
  );
}
