'use client';

import { Tag, Tooltip, Typography } from 'antd';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

const { Text } = Typography;

const statusColorMap: Record<string, string> = {
  blocked: 'red',
  pending: 'blue',
  executing: 'orange',
  completed: 'green',
  failed: 'red',
  cancelled: 'default',
  approved: 'green',
  rejected: 'red',
  pending_approval: 'orange',
};

const statusLabelMap: Record<string, string> = {
  blocked: '已阻断',
  pending: '待处理',
  pending_approval: '待审批',
  executing: '执行中',
  completed: '已完成',
  failed: '失败',
  cancelled: '已取消',
  approved: '已批准',
  rejected: '已拒绝',
};

const statusTooltipMap: Record<string, string> = {
  blocked: '该任务已被系统阻断，需查看审计日志了解原因',
  pending: '任务等待处理',
  executing: '任务正在执行中',
  completed: '任务已完成',
  failed: '任务执行失败，可查看错误信息',
  cancelled: '任务已取消',
  pending_approval: '等待 Owner 审批',
};

export default function ListingTasksPage() {
  return (
    <CrudListPage
      resource="/listing-tasks"
      title="Listing 任务"
      singular="Listing 任务"
      searchPlaceholder="搜索来源键 / 目的国..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '商品ID', dataIndex: 'product_id', width: 90 },
        { title: '平台ID', dataIndex: 'platform_id', width: 90 },
        {
          title: '状态',
          dataIndex: 'status',
          width: 100,
          render: (v: unknown) => {
            const status = String(v);
            const tip = statusTooltipMap[status];
            return (
              <Tooltip title={tip || ''}>
                <Tag color={statusColorMap[status] || 'default'}>
                  {statusLabelMap[status] || status}
                </Tag>
              </Tooltip>
            );
          },
        },
        { title: '目标售价', dataIndex: 'target_sale_price', width: 110 },
        { title: '目的国', dataIndex: 'destination_country', width: 80 },
        {
          title: '错误信息',
          dataIndex: 'last_error',
          width: 280,
          render: (v: unknown) => {
            if (!v || v === '') return <Text type="secondary">-</Text>;
            const msg = String(v);
            return (
              <Tooltip title={msg}>
                <Text type="danger" ellipsis style={{ maxWidth: 260, cursor: 'pointer' }}>
                  {msg}
                </Text>
              </Tooltip>
            );
          },
        },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
        { title: '更新时间', dataIndex: 'updated_at', width: 160, render: fmtDate },
        { title: '审批ID', dataIndex: 'approval_id', width: 90 },
        {
          title: '审批状态',
          dataIndex: 'approval_status',
          width: 110,
          render: (v: unknown) => {
            if (!v) return '-';
            const apprColors: Record<string, string> = { pending: 'blue', approved: 'green', rejected: 'red' };
            const apprLabels: Record<string, string> = { pending: '审批中', approved: '已批准', rejected: '已拒绝' };
            return <Tag color={apprColors[String(v)] || 'default'}>{apprLabels[String(v)] || String(v)}</Tag>;
          },
        },
      ]}
      fields={[
        { name: 'product_id', label: '商品ID', type: 'number', required: true },
        { name: 'platform_id', label: '平台ID', type: 'number', required: true },
        { name: 'status', label: '状态', initialValue: 'blocked' },
        { name: 'destination_country', label: '目的国' },
        { name: 'target_sale_price', label: '目标售价', type: 'number' },
        { name: 'last_error', label: '错误信息', type: 'textarea' },
      ]}
    />
  );
}
