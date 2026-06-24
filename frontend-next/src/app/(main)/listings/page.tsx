'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function ListingsPage() {
  return (
    <CrudListPage
      resource="/listings"
      title="Listing"
      singular="Listing"
      searchPlaceholder="搜索外部ID / Listing URL..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '平台ID', dataIndex: 'platform_id', width: 90 },
        { title: '商品ID', dataIndex: 'product_id', width: 90 },
        { title: '外部ID', dataIndex: 'external_id', width: 150 },
        { title: '状态', dataIndex: 'status', width: 110 },
        { title: 'Listing URL', dataIndex: 'listing_url', width: 280 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'platform_id', label: '平台ID', type: 'number', required: true },
        { name: 'product_id', label: '商品ID', type: 'number', required: true },
        { name: 'external_id', label: '外部ID' },
        { name: 'listing_url', label: 'Listing URL' },
        { name: 'status', label: '状态', initialValue: 'draft' },
      ]}
    />
  );
}
