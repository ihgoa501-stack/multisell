'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function ListingTasksPage() {
  return (
    <CrudListPage
      resource="/listing-tasks"
      title="Listing 任务"
      singular="Listing 任务"
      searchPlaceholder="搜索任务 / 错误信息..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '商品ID', dataIndex: 'product_id', width: 90 },
        { title: '平台ID', dataIndex: 'platform_id', width: 90 },
        { title: '状态', dataIndex: 'status', width: 110 },
        { title: '重试次数', dataIndex: 'retry_count', width: 100 },
        { title: '错误信息', dataIndex: 'error_message', width: 280 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
        { title: '更新时间', dataIndex: 'updated_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'product_id', label: '商品ID', type: 'number', required: true },
        { name: 'platform_id', label: '平台ID', type: 'number', required: true },
        { name: 'status', label: '状态', initialValue: 'pending' },
        { name: 'retry_count', label: '重试次数', type: 'number', initialValue: 0 },
        { name: 'error_message', label: '错误信息', type: 'textarea' },
      ]}
    />
  );
}
