'use client';

import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

export default function AftersalesPage() {
  return (
    <CrudListPage
      resource="/aftersales"
      title="售后"
      singular="售后单"
      searchPlaceholder="搜索订单ID / SKU ID / 原因..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '订单ID', dataIndex: 'order_id', width: 100 },
        { title: 'SKU ID', dataIndex: 'sku_id', width: 90 },
        { title: '退货数量', dataIndex: 'return_quantity', width: 100 },
        { title: '原因', dataIndex: 'reason', width: 200 },
        { title: '退款金额', dataIndex: 'refund_amount', width: 120, render: fmtMoney },
        { title: '状态', dataIndex: 'status', width: 110 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'order_id', label: '订单ID', type: 'number', required: true },
        { name: 'sku_id', label: 'SKU ID', type: 'number', required: true },
        { name: 'return_quantity', label: '退货数量', type: 'number', initialValue: 1 },
        { name: 'reason', label: '原因', type: 'textarea', required: true },
        { name: 'refund_amount', label: '退款金额', type: 'number' },
        { name: 'status', label: '状态', initialValue: 'pending' },
      ]}
    />
  );
}
