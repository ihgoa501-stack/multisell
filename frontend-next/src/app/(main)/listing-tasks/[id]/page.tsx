'use client';

import { Alert, Card, Descriptions, Table, Spin, Result, Button, Space, Tag, message } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeftOutlined, PlayCircleOutlined, RedoOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';

interface ListingTask {
  id: number;
  product_id: number;
  platform_id: number;
  sku_id?: number;
  product_listing_id?: number;
  source_type: string;
  source_item_key: string;
  status: string;
  missing_requirements?: unknown;
  decision_snapshot?: unknown;
  target_sale_price?: number;
  target_profit_margin?: number;
  destination_country: string;
  last_error: string;
  created_by: string;
  updated_by: string;
  created_at: string;
  updated_at: string;
}

interface ListingTaskItem {
  id: number;
  task_id: number;
  product_id: number;
  platform_id: number;
  status: string;
  result?: unknown;
  error_message: string;
  retry_count: number;
  executed_at?: string;
}

interface TaskDetailResponse {
  task: ListingTask;
  items: ListingTaskItem[];
}

interface ApprovalRequest {
  id: number;
  status: string;
  request_type: string;
  target_type?: string;
  target_id?: number;
  risk_level?: string;
  reason?: string;
}

export default function ListingTaskDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['listing-task', id],
    queryFn: async () => {
      const res = await apiClient.get<TaskDetailResponse>(`/v1/listing-tasks/${id}`);
      return res.data;
    },
    retry: false,
  });

  const task = data?.task;
  const items = data?.items || [];

  // Fetch approvals for this listing task
  const { data: approvalsData } = useQuery({
    queryKey: ['listing-task-approvals', id],
    queryFn: async () => {
      const res = await apiClient.get<ApprovalRequest[]>('/v1/approval', {
        status: '',
        request_type: 'publish',
        page: '1',
        size: '100',
      });
      return (Array.isArray(res.data) ? res.data : []).filter(
        (a) => a.target_type === 'listing_task' && String(a.target_id) === String(id)
      );
    },
  });

  const publishApproval = approvalsData?.[0];
  const isApproved = publishApproval?.status === 'approved';
  const isRejected = publishApproval?.status === 'rejected';
  const isPendingApproval = task?.status === 'blocked' && publishApproval?.status === 'pending';

  const executeMutation = useMutation({
    mutationFn: async () => {
      // Uses the /listing-task prefix for execution actions
      return apiClient.post(`/v1/listing-task/${id}/execute`);
    },
    onSuccess: () => {
      message.success('执行任务已启动');
      queryClient.invalidateQueries({ queryKey: ['listing-task', id] });
    },
    onError: (err: Error) => {
      message.error(`执行失败: ${err.message}`);
    },
  });

  const retryMutation = useMutation({
    mutationFn: async () => {
      // Uses the /listing-task prefix for retry actions
      return apiClient.post(`/v1/listing-task/${id}/retry-failed`);
    },
    onSuccess: () => {
      message.success('重试已触发');
      queryClient.invalidateQueries({ queryKey: ['listing-task', id] });
    },
    onError: (err: Error) => {
      message.error(`重试失败: ${err.message}`);
    },
  });

  const columns = [
    { title: '明细 ID', dataIndex: 'id', key: 'id', width: 80 },
    { title: '商品 ID', dataIndex: 'product_id', key: 'product_id', width: 90 },
    { title: '平台 ID', dataIndex: 'platform_id', key: 'platform_id', width: 90 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => {
        const colors: Record<string, string> = {
          pending: 'blue',
          running: 'orange',
          success: 'green',
          failed: 'red',
        };
        return <Tag color={colors[status] || 'blue'}>{status.toUpperCase()}</Tag>;
      },
    },
    { title: '重试次数', dataIndex: 'retry_count', key: 'retry_count', width: 100 },
    { title: '错误信息', dataIndex: 'error_message', key: 'error_message' },
    {
      title: '执行时间',
      dataIndex: 'executed_at',
      key: 'executed_at',
      width: 160,
      render: (t?: string) => (t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
  ];

  return (
    <PageContainer title="Listing 任务详情">
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/listing-tasks')}>
          返回列表
        </Button>
        {task?.status === 'blocked' ? (
          <Button type="primary" onClick={() => router.push('/approval')}>
            去审批
          </Button>
        ) : task?.status === 'failed' ? (
          <Button
            type="primary"
            icon={<RedoOutlined />}
            loading={retryMutation.isPending}
            onClick={() => retryMutation.mutate()}
          >
            重试失败项
          </Button>
        ) : (
          <Button
            type="primary"
            icon={<PlayCircleOutlined />}
            loading={executeMutation.isPending}
            onClick={() => executeMutation.mutate()}
            disabled={task?.status === 'success' || task?.status === 'running'}
          >
            启动执行
          </Button>
        )}
      </Space>

      {isLoading ? (
        <Card>
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin tip="加载中..." />
          </div>
        </Card>
      ) : isError || !data ? (
        <Card>
          <Result status="info" title="任务详情" subTitle="暂无详情数据或任务不存在" />
        </Card>
      ) : (
        <Space direction="vertical" size="middle" style={{ display: 'flex' }}>
          {task?.status === 'blocked' && (
            <Alert
              type={isApproved ? 'success' : isRejected ? 'error' : 'warning'}
              message={isApproved ? '审批已通过，可以执行' : isRejected ? '审批已拒绝，任务保持阻塞' : '该任务等待 Owner 审批'}
              description={publishApproval?.reason || '审批通过前，系统不会执行该刊登任务。'}
              showIcon
            />
          )}
          <Card title="基本信息">
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="任务 ID">{task?.id}</Descriptions.Item>
              <Descriptions.Item label="商品 ID">{task?.product_id}</Descriptions.Item>
              <Descriptions.Item label="平台 ID">{task?.platform_id}</Descriptions.Item>
              <Descriptions.Item label="SKU ID">{task?.sku_id || '-'}</Descriptions.Item>
              <Descriptions.Item label="来源类型">{task?.source_type}</Descriptions.Item>
              <Descriptions.Item label="目标国家">{task?.destination_country || '-'}</Descriptions.Item>
              <Descriptions.Item label="目标售价">
                {task?.target_sale_price ? `¥${task.target_sale_price.toFixed(2)}` : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="目标利润率">
                {task?.target_profit_margin ? `${task.target_profit_margin.toFixed(2)}%` : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={task?.status === 'success' ? 'green' : task?.status === 'running' ? 'orange' : 'red'}>
                  {task?.status?.toUpperCase()}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="创建者">{task?.created_by || '-'}</Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {task?.created_at ? dayjs(task.created_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="更新时间">
                {task?.updated_at ? dayjs(task.updated_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
              {task?.last_error && (
                <Descriptions.Item label="最后一次错误" span={2}>
                  <pre style={{ color: '#ff4d4f', margin: 0, whiteSpace: 'pre-wrap' }}>{task.last_error}</pre>
                </Descriptions.Item>
              )}
            </Descriptions>
          </Card>

          <Card title="任务执行明细">
            <Table
              dataSource={items}
              columns={columns}
              rowKey="id"
              pagination={false}
              size="small"
            />
          </Card>
        </Space>
      )}
    </PageContainer>
  );
}
