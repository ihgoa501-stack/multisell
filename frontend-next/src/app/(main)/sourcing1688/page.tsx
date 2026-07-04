'use client';

import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

export default function Sourcing1688Page() {
  return (
    <CrudListPage
      resource="/sourcing-1688"
      title="1688 选品"
      singular="选品记录"
      searchPlaceholder="搜索供应商名称 / 1688 链接..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '供应商名称', dataIndex: 'supplier_name', width: 200 },
        { title: '来源链接', dataIndex: 'source_url', width: 280 },
        { title: '1688价格', dataIndex: 'price_1688', width: 120, render: fmtMoney },
        { title: '起订量', dataIndex: 'min_order_qty', width: 100 },
        { title: '状态', dataIndex: 'status', width: 110 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'supplier_name', label: '供应商名称', required: true },
        { name: 'source_url', label: '来源链接', required: true },
        { name: 'price_1688', label: '1688价格', type: 'number' },
        { name: 'min_order_qty', label: '起订量', type: 'number', initialValue: 1 },
        { name: 'status', label: '状态', initialValue: 'pending' },
      ]}
    />
  );
}
