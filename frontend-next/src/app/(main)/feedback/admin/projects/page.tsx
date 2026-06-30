'use client';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function FeedbackProjectsPage() {
  return (
    <CrudListPage
      resource="/feedback/projects"
      title="反馈项目"
      singular="项目"
      searchPlaceholder="搜索项目名称..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '名称', dataIndex: 'name', width: 200 },
        { title: '标识', dataIndex: 'slug', width: 150 },
        { title: '描述', dataIndex: 'description' },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'name', label: '项目名称', required: true },
        { name: 'slug', label: '标识符', required: true },
        { name: 'description', label: '描述', type: 'textarea' },
        { name: 'settings', label: '设置 (JSON)', type: 'textarea', initialValue: '{}' },
      ]}
    />
  );
}
