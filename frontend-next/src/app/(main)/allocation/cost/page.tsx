'use client';

import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

export default function AllocationCostPage() {
  return (
    <CrudListPage
      resource="/allocation/cost"
      title="成本分摊批次"
      singular="成本批次"
      editable={false}
      searchPlaceholder="搜索批次号..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '批次号', dataIndex: 'batch_no', width: 180 },
        { title: '总金额', dataIndex: 'total_amount', width: 140, render: fmtMoney },
        { title: '状态', dataIndex: 'status', width: 120 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'batch_no', label: '批次号', required: true },
        { name: 'remark', label: '备注', type: 'textarea' },
      ]}
    />
  );
}
