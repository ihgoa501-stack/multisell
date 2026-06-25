'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function InventoryPage() {
  return (
    <CrudListPage
      resource="/inventory"
      title="库存"
      singular="库存"
      searchPlaceholder="搜索 SKU ID / 批次号 / 仓库..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: 'SKU ID', dataIndex: 'sku_id', width: 90 },
        { title: '仓库ID', dataIndex: 'warehouse_id', width: 90 },
        { title: '可用库存', dataIndex: 'quantity', width: 100 },
        { title: '锁定库存', dataIndex: 'locked_quantity', width: 100 },
        { title: '安全库存', dataIndex: 'safety_stock', width: 100 },
        { title: '批次号', dataIndex: 'batch_no', width: 150 },
        { title: '更新时间', dataIndex: 'updated_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'sku_id', label: 'SKU ID', type: 'number', required: true },
        { name: 'warehouse_id', label: '仓库ID', type: 'number', required: true },
        { name: 'quantity', label: '可用库存', type: 'number', initialValue: 0 },
        { name: 'locked_quantity', label: '锁定库存', type: 'number', initialValue: 0 },
        { name: 'safety_stock', label: '安全库存', type: 'number', initialValue: 0 },
        { name: 'batch_no', label: '批次号' },
      ]}
    />
  );
}
