'use client';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function FeedbackCategoriesPage() {
  return (
    <CrudListPage
      resource="/feedback/categories"
      title="反馈分类"
      singular="分类"
      searchPlaceholder="搜索分类名称..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '分类名称', dataIndex: 'name', width: 200 },
        { title: '颜色', dataIndex: 'color', width: 80, render: (v: unknown) => { const c = v as string; return c && <span style={{ color: c }}>■ {c}</span>; } },
        { title: '排序', dataIndex: 'sort_order', width: 60 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'project_id', label: '项目ID', type: 'number', required: true },
        { name: 'name', label: '分类名称', required: true },
        { name: 'color', label: '颜色' },
        { name: 'icon', label: '图标' },
        { name: 'sort_order', label: '排序', type: 'number', initialValue: 0 },
      ]}
    />
  );
}
