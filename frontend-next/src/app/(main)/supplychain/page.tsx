'use client';

import { useCallback } from 'react';
import { Tag } from 'antd';
import apiClient from '@/lib/api-client';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';
import type { CrudColumn } from '@/components/crud/CrudListPage';

const STATUS_MAP: Record<string, { color: string; label: string }> = {
  pending: { color: 'default', label: '待处理' },
  processing: { color: 'processing', label: '处理中' },
  completed: { color: 'success', label: '已完成' },
  failed: { color: 'error', label: '失败' },
  cancelled: { color: 'warning', label: '已取消' },
};

const columns: CrudColumn[] = [
  { title: 'ID', dataIndex: 'id', width: 280 },
  { title: '来源类型', dataIndex: 'source_type', width: 120 },
  { title: '来源ID', dataIndex: 'source_id', width: 150 },
  {
    title: '状态',
    dataIndex: 'status',
    width: 100,
    render: (value: unknown) => {
      const status = STATUS_MAP[value as string] ?? { color: 'default', label: value as string };
      return <Tag color={status.color}>{status.label}</Tag>;
    },
  },
  { title: '创建时间', dataIndex: 'created_at', width: 180, render: fmtDate },
  { title: '更新时间', dataIndex: 'updated_at', width: 180, render: fmtDate },
];

const fields = [
  { name: 'source_type', label: '来源类型', type: 'text' as const, required: true },
  { name: 'source_id', label: '来源ID', type: 'text' as const, required: true },
  { name: 'status', label: '状态', type: 'text' as const, initialValue: 'pending' },
];

export default function SupplyChainPage() {
  const handleRowClick = useCallback((record: Record<string, unknown>) => {
    window.location.href = `/supplychain/${record.id}`;
  }, []);

  return (
    <CrudListPage
      resource="/supplychain/flows"
      title="供应链追踪"
      singular="供应链流"
      searchPlaceholder=""
      columns={columns}
      fields={fields}
      editable={true}
      pageSize={20}
      rowKey="id"
    />
  );
}
