'use client';

import CrudListPage from '@/components/crud/CrudListPage';

const reasonOptions = [
  { label: '低于安全库存', value: 'low_stock' },
  { label: '预测需求', value: 'forecast' },
  { label: '促销活动', value: 'promotion' },
  { label: '手动触发', value: 'manual' },
];

const reasonMap: Record<string, string> = {};
for (const r of reasonOptions) {
  reasonMap[r.value] = r.label;
}

const statusOptions = [
  { label: '待处理', value: 'pending' },
  { label: '已下单', value: 'ordered' },
  { label: '已忽略', value: 'ignored' },
];

const statusMap: Record<string, string> = {};
for (const s of statusOptions) {
  statusMap[s.value] = s.label;
}

export default function PurchaseSuggestionsPage() {
  return (
    <CrudListPage
      resource="/purchase/suggestions"
      title="采购建议"
      singular="采购建议"
      searchPlaceholder="搜索 SKU..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: 'SKU', dataIndex: 'sku', width: 160 },
        {
          title: '建议数量',
          dataIndex: 'suggested_quantity',
          width: 110,
        },
        {
          title: '原因',
          dataIndex: 'reason',
          width: 130,
          render: (v: unknown) => reasonMap[String(v)] ?? String(v),
        },
        {
          title: '状态',
          dataIndex: 'status',
          width: 100,
          render: (v: unknown) => statusMap[String(v)] ?? String(v),
        },
      ]}
      fields={[
        { name: 'sku', label: 'SKU', required: true },
        { name: 'suggested_quantity', label: '建议数量', type: 'number', required: true },
        {
          name: 'reason',
          label: '原因',
          type: 'select',
          options: reasonOptions,
          required: true,
        },
        {
          name: 'status',
          label: '状态',
          type: 'select',
          options: statusOptions,
          initialValue: 'pending',
        },
      ]}
    />
  );
}
