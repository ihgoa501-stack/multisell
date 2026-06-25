'use client';

import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

const statusOptions = [
  { label: '草稿', value: 'draft' },
  { label: '待审批', value: 'pending' },
  { label: '已审批', value: 'approved' },
  { label: '部分到货', value: 'partial' },
  { label: '已完成', value: 'completed' },
];

const statusMap: Record<string, string> = {};
for (const s of statusOptions) {
  statusMap[s.value] = s.label;
}

export default function PurchasePage() {
  return (
    <CrudListPage
      resource="/purchase"
      title="采购订单"
      singular="采购订单"
      searchPlaceholder="搜索订单号 / 供应商..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '订单号', dataIndex: 'order_no', width: 180 },
        { title: '供应商', dataIndex: 'supplier', width: 160 },
        { title: '总金额', dataIndex: 'total_amount', width: 120, render: fmtMoney },
        {
          title: '状态',
          dataIndex: 'status',
          width: 120,
          render: (v: unknown) => statusMap[String(v)] ?? String(v),
        },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'order_no', label: '订单号', required: true },
        { name: 'supplier', label: '供应商', required: true },
        { name: 'total_amount', label: '总金额', type: 'number', required: true },
        {
          name: 'status',
          label: '状态',
          type: 'select',
          options: statusOptions,
          initialValue: 'draft',
        },
      ]}
    />
  );
}
