'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function BrandsPage() {
  return (
    <CrudListPage
      resource="/brands"
      title="品牌"
      singular="品牌"
      searchPlaceholder="搜索品牌名称..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '品牌名称', dataIndex: 'name', width: 200 },
        { title: 'Logo', dataIndex: 'logo', width: 200 },
        { title: '国家', dataIndex: 'country', width: 120 },
        { title: '排序', dataIndex: 'sort_order', width: 90 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'name', label: '品牌名称', required: true },
        { name: 'logo', label: 'Logo URL' },
        { name: 'country', label: '国家' },
        { name: 'sort_order', label: '排序', type: 'number', initialValue: 0 },
        { name: 'status', label: '状态', initialValue: 'active' },
      ]}
    />
  );
}
