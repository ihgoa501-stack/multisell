'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function OperationLogsPage() {
  return (
    <CrudListPage
      resource="/operation-log"
      title="操作日志"
      singular="日志"
      editable={false}
      searchPlaceholder="搜索操作/操作人..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '操作人', dataIndex: 'operator', width: 140 },
        { title: '动作', dataIndex: 'action', width: 160 },
        { title: '目标类型', dataIndex: 'target_type', width: 140 },
        { title: '目标 ID', dataIndex: 'target_id', width: 140 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[]}
    />
  );
}
