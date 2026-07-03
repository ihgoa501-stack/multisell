'use client';

import CrudListPage from '@/components/crud/CrudListPage';

export default function DisputesPage() {
  return (
    <CrudListPage
      resource="/aftersales/disputes"
      title="争议管理"
      singular="争议"
      searchPlaceholder=""
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '交易ID', dataIndex: 'transaction_id', width: 140 },
        { title: '平台', dataIndex: 'platform', width: 100 },
        { title: '索赔类型', dataIndex: 'claim_type', width: 120 },
        { title: '金额', dataIndex: 'amount', width: 110, render: (v) => `¥${Number(v || 0).toFixed(2)}` },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '决策分数', dataIndex: 'decision_score', width: 100, render: (v) => Number(v || 0).toFixed(1) },
        { title: '决策来源', dataIndex: 'decision_source', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: (v) => v ? new Date(v as string).toLocaleString('zh-CN') : '-' },
      ]}
      fields={[
        { name: 'transaction_id', label: '交易ID', type: 'text', required: true },
        { name: 'platform', label: '平台', type: 'text', required: true },
        { name: 'claim_type', label: '索赔类型', type: 'text', required: true },
        { name: 'amount', label: '金额', type: 'number' },
        { name: 'evidence', label: '证据', type: 'textarea' },
      ]}
    />
  );
}
