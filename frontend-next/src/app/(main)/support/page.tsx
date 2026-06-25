'use client';

import { Tag } from 'antd';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

const statusMap: Record<string, { color: string; label: string }> = {
  open: { color: 'blue', label: '开启' },
  in_progress: { color: 'orange', label: '处理中' },
  resolved: { color: 'green', label: '已解决' },
  closed: { color: 'default', label: '关闭' },
};

const priorityMap: Record<string, { color: string; label: string }> = {
  low: { color: 'default', label: '低' },
  medium: { color: 'blue', label: '中' },
  high: { color: 'orange', label: '高' },
  urgent: { color: 'red', label: '紧急' },
};

export default function SupportPage() {
  return (
    <CrudListPage
      resource="/support"
      title="客服中心"
      singular="会话"
      searchPlaceholder="搜索会话ID / 客户名称 / 主题..."
      columns={[
        { title: '会话ID', dataIndex: 'id', width: 80 },
        { title: '客户名称', dataIndex: 'customer_name', width: 140 },
        { title: '主题', dataIndex: 'subject', width: 240 },
        {
          title: '状态',
          dataIndex: 'status',
          width: 100,
          render: (v) => {
            const s = statusMap[String(v)] ?? { color: 'default', label: String(v) };
            return <Tag color={s.color}>{s.label}</Tag>;
          },
        },
        {
          title: '优先级',
          dataIndex: 'priority',
          width: 90,
          render: (v) => {
            const p = priorityMap[String(v)] ?? { color: 'default', label: String(v) };
            return <Tag color={p.color}>{p.label}</Tag>;
          },
        },
        { title: '最后消息时间', dataIndex: 'last_message_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'customer_name', label: '客户名称', required: true },
        { name: 'subject', label: '主题' },
        {
          name: 'status',
          label: '状态',
          type: 'select',
          initialValue: 'open',
          options: [
            { label: '开启', value: 'open' },
            { label: '处理中', value: 'in_progress' },
            { label: '已解决', value: 'resolved' },
            { label: '关闭', value: 'closed' },
          ],
        },
        {
          name: 'priority',
          label: '优先级',
          type: 'select',
          initialValue: 'medium',
          options: [
            { label: '低', value: 'low' },
            { label: '中', value: 'medium' },
            { label: '高', value: 'high' },
            { label: '紧急', value: 'urgent' },
          ],
        },
      ]}
      filters={[
        {
          key: 'status',
          label: '状态',
          options: [
            { label: '开启', value: 'open' },
            { label: '处理中', value: 'in_progress' },
            { label: '已解决', value: 'resolved' },
            { label: '关闭', value: 'closed' },
          ],
        },
        {
          key: 'priority',
          label: '优先级',
          options: [
            { label: '低', value: 'low' },
            { label: '中', value: 'medium' },
            { label: '高', value: 'high' },
            { label: '紧急', value: 'urgent' },
          ],
        },
      ]}
    />
  );
}
