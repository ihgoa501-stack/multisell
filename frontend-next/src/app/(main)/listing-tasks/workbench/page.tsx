'use client';

import { Card, Statistic, Button, Space, message } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';
import apiClient from '@/lib/api-client';

export default function ListingTasksWorkbenchPage() {
  const qc = useQueryClient();

  const { data: stats } = useQuery({
    queryKey: ['listing-task', 'stats'],
    queryFn: async () => {
      const res = await apiClient.get<{ pending?: number; total?: number }>('/v1/listing-task/stats');
      return res.data;
    },
    retry: false,
  });

  const retryAll = useMutation({
    mutationFn: async () => apiClient.post('/v1/listing-task/retry-all'),
    onSuccess: () => {
      message.success('已触发批量重试');
      qc.invalidateQueries({ queryKey: ['crud', '/listing-task'] });
    },
    onError: (e: Error) => message.error(`重试失败: ${e.message}`),
  });

  return (
    <div style={{ padding: 24 }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', marginBottom: 16 }}>AI Listing 工作台</h1>

      <Card style={{ marginBottom: 16 }}>
        <Space size="large">
          <Statistic title="未完成任务" value={stats?.pending ?? 0} />
          <Statistic title="总任务数" value={stats?.total ?? 0} />
          <Button
            type="primary"
            icon={<ReloadOutlined />}
            loading={retryAll.isPending}
            onClick={() => retryAll.mutate()}
          >
            批量重试
          </Button>
        </Space>
      </Card>

      <CrudListPage
        resource="/listing-task"
        title="Listing 任务"
        singular="任务"
        editable={false}
        searchPlaceholder="搜索任务..."
        columns={[
          { title: 'ID', dataIndex: 'id', width: 70 },
          { title: '商品 ID', dataIndex: 'product_id', width: 110 },
          { title: '平台 ID', dataIndex: 'platform_id', width: 110 },
          { title: '状态', dataIndex: 'status', width: 120 },
          { title: '重试次数', dataIndex: 'retry_count', width: 100 },
          { title: '错误信息', dataIndex: 'error_message' },
          { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
        ]}
        fields={[]}
      />
    </div>
  );
}
