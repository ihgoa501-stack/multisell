'use client';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function FeedbackTagsPage() {
  return (
    <CrudListPage
      resource="/feedback/tags"
      title="反馈标签"
      singular="标签"
      searchPlaceholder="搜索标签名称..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '标签名称', dataIndex: 'name', width: 200 },
        { title: '颜色', dataIndex: 'color', width: 80, render: (v: unknown) => { const c = v as string; return c && <span style={{ color: c }}>■ {c}</span>; } },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'project_id', label: '项目ID', type: 'number', required: true },
        { name: 'name', label: '标签名称', required: true },
        { name: 'color', label: '颜色' },
      ]}
    />
  );
}
