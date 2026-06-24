'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function NotificationsPage() {
  return (
    <CrudListPage
      resource="/notification"
      title="通知"
      singular="通知"
      searchPlaceholder="搜索通知标题 / 类型..."
      editable={false}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '通知类型', dataIndex: 'alert_type', width: 130 },
        { title: '标题', dataIndex: 'title', width: 220 },
        { title: '内容', dataIndex: 'content', width: 320 },
        { title: '严重级别', dataIndex: 'severity', width: 100 },
        { title: '已读', dataIndex: 'is_read', width: 80 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'alert_type', label: '通知类型' },
        { name: 'title', label: '标题' },
        { name: 'content', label: '内容', type: 'textarea' },
        { name: 'severity', label: '严重级别' },
        { name: 'is_read', label: '已读' },
      ]}
    />
  );
}
