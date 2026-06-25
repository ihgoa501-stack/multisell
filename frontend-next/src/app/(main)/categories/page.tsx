'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function CategoriesPage() {
  return (
    <CrudListPage
      resource="/categories"
      title="分类"
      singular="分类"
      searchPlaceholder="搜索分类名称..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '分类名称', dataIndex: 'name', width: 200 },
        { title: '父级ID', dataIndex: 'parent_id', width: 90 },
        { title: '层级', dataIndex: 'level', width: 90 },
        { title: '排序', dataIndex: 'sort_order', width: 90 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'name', label: '分类名称', required: true },
        { name: 'parent_id', label: '父级ID', type: 'number' },
        { name: 'level', label: '层级', type: 'number', initialValue: 1 },
        { name: 'sort_order', label: '排序', type: 'number', initialValue: 0 },
        { name: 'status', label: '状态', initialValue: 'active' },
      ]}
    />
  );
}
