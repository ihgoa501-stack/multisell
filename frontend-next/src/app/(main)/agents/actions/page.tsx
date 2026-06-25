'use client';

import { Tag } from 'antd';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

const STATUS_COLORS: Record<string, string> = {
  suggested: 'blue',
  approved: 'green',
  executing: 'cyan',
  executed: 'green',
  rejected: 'red',
  failed: 'red',
  reviewed: 'default',
};

function StatusTag({ value }: { value: unknown }) {
  const s = String(value ?? '');
  return <Tag color={STATUS_COLORS[s] ?? 'default'}>{s || '-'}</Tag>;
}

export default function AgentsActionsPage() {
  return (
    <CrudListPage
      resource="/v1/ai/actions"
      title="Agent Actions"
      singular="Action"
      editable={false}
      searchPlaceholder="搜索 action..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: 'Agent', dataIndex: 'agent_id', width: 160 },
        { title: '类型', dataIndex: 'type', width: 140 },
        { title: '标题', dataIndex: 'title' },
        { title: '状态', dataIndex: 'status', width: 120, render: (v) => <StatusTag value={v} /> },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[]}
    />
  );
}
