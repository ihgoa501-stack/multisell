'use client';

import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

export default function SkuPage() {
  return (
    <CrudListPage
      resource="/skus"
      title="SKU"
      singular="SKU"
      searchPlaceholder="搜索 SKU 编码 / 条形码 / 规格描述..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: 'SKU编码', dataIndex: 'code', width: 160 },
        { title: '条形码', dataIndex: 'barcode', width: 150 },
        { title: '规格描述', dataIndex: 'spec_desc', width: 180 },
        { title: '商品ID', dataIndex: 'product_id', width: 90 },
        { title: '售价', dataIndex: 'price', width: 110, render: fmtMoney },
        { title: '成本价', dataIndex: 'cost_price', width: 110, render: fmtMoney },
        { title: '市场价', dataIndex: 'market_price', width: 110, render: fmtMoney },
        { title: '库存', dataIndex: 'stock', width: 90 },
        { title: '重量(kg)', dataIndex: 'weight', width: 100 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'code', label: 'SKU编码', required: true },
        { name: 'barcode', label: '条形码' },
        { name: 'spec_desc', label: '规格描述' },
        { name: 'product_id', label: '商品ID', type: 'number' },
        { name: 'price', label: '售价', type: 'number' },
        { name: 'cost_price', label: '成本价', type: 'number' },
        { name: 'market_price', label: '市场价', type: 'number' },
        { name: 'stock', label: '库存', type: 'number' },
        { name: 'weight', label: '重量(kg)', type: 'number' },
        { name: 'status', label: '状态', initialValue: 'active' },
      ]}
    />
  );
}
