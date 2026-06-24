'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function ProductsPage() {
  return (
    <CrudListPage
      resource="/products"
      title="商品"
      singular="商品"
      searchPlaceholder="搜索商品名称 / 编码 / 副标题..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '商品名称', dataIndex: 'name', width: 200 },
        { title: '副标题', dataIndex: 'subtitle', width: 200 },
        { title: '品牌ID', dataIndex: 'brand_id', width: 90 },
        { title: '分类ID', dataIndex: 'category_id', width: 90 },
        { title: '单位', dataIndex: 'unit', width: 80 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '货品类型', dataIndex: 'cargo_type', width: 110 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'name', label: '商品名称', required: true },
        { name: 'subtitle', label: '副标题' },
        { name: 'brand_id', label: '品牌ID', type: 'number' },
        { name: 'category_id', label: '分类ID', type: 'number' },
        { name: 'unit', label: '单位' },
        { name: 'status', label: '状态', initialValue: 'active' },
        { name: 'main_image', label: '主图URL' },
        { name: 'cargo_type', label: '货品类型' },
      ]}
    />
  );
}
