'use client';

import { useState } from 'react';
import { Alert, Card, Descriptions, Table, Spin, Result, Button, Space, Tag, message } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeftOutlined, PlayCircleOutlined, RedoOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import HighRiskConfirmDialog from '@/components/ui/HighRiskConfirmDialog';

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
  approval_id?: number;
  target_sale_price?: number;
  target_profit_margin?: number;
  destination_country: string;
  last_error: string;
  execution_mode?: number;
  external_reference_id?: string;
  external_reference_url?: string;
  created_by: string;
  updated_by: string;
  created_at: string;
  updated_at: string;
}
interface ListingTaskItem {
  id: number; task_id: number; product_id: number; platform_id: number;
  status: string; result?: unknown; error_message: string; retry_count: number; executed_at?: string;
}
interface TaskDetailResponse { task: ListingTask; items: ListingTaskItem[]; }
interface ApprovalRequest {
  id: number;
  target_type?: string;
  target_id?: number;
  status?: string;
  reason?: string;
}

const MODE_MAP: Record<number, { label: string; color: string }> = {
  0: { label: 'Dry-Run', color: 'default' },
  1: { label: 'Sandbox', color: 'orange' },
  2: { label: '审批后生产', color: 'red' },
  3: { label: '生产', color: 'red' },
};
const STATUS_COLORS: Record<string, string> = {
  pending: 'blue', completed: 'success', executing: 'processing', failed: 'error',
  approved: 'blue', pending_approval: 'gold', blocked: 'default', rejected: 'error', cancelled: 'default',
};

export default function ListingTaskDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;
  const queryClient = useQueryClient();
  const [confirmOpen, setConfirmOpen] = useState(false);

  const { data, isLoading, isError } = useQuery({
    queryKey: ['listing-task', id],
    queryFn: async () => (await apiClient.get<TaskDetailResponse>(`/v1/listing-tasks/${id}`)).data,
    retry: false,
  });
  const task = data?.task;
  const items = data?.items || [];
  const modeInfo = MODE_MAP[task?.execution_mode ?? 0] || { label: '未设置', color: 'default' };
  const isProduction = task?.execution_mode === 2 || task?.execution_mode === 3;

  const { data: approvalsData } = useQuery({
    queryKey: ['listing-task-approvals', id],
    queryFn: async () => {
      const res = await apiClient.get<ApprovalRequest[]>('/v1/approval', {
        status: '',
        request_type: 'listing_task',
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

  const executeMutation = useMutation({
    mutationFn: async () => apiClient.post(`/v1/listing-task/${id}/execute`),
    onSuccess: () => { message.success('执行任务已启动'); queryClient.invalidateQueries({ queryKey: ['listing-task', id] }); },
    onError: (err: Error) => { message.error(`执行失败: ${err.message}`); },
  });
  const retryMutation = useMutation({
    mutationFn: async () => apiClient.post(`/v1/listing-task/${id}/retry-failed`),
    onSuccess: () => { message.success('重试已触发'); queryClient.invalidateQueries({ queryKey: ['listing-task', id] }); },
    onError: (err: Error) => { message.error(`重试失败: ${err.message}`); },
  });

  const cols = [
    { title: '明细 ID', dataIndex: 'id', key: 'id', width: 80 },
    { title: '商品 ID', dataIndex: 'product_id', key: 'product_id', width: 90 },
    { title: '平台 ID', dataIndex: 'platform_id', key: 'platform_id', width: 90 },
    { title: '状态', dataIndex: 'status', key: 'status', width: 120, render: (s: string) => <Tag color={STATUS_COLORS[s] || 'default'}>{s.toUpperCase()}</Tag> },
    { title: '重试次数', dataIndex: 'retry_count', key: 'retry_count', width: 100 },
    { title: '错误信息', dataIndex: 'error_message', key: 'error_message' },
    { title: '执行时间', dataIndex: 'executed_at', key: 'executed_at', width: 160, render: (t?: string) => t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-' },
  ];

  return (
    <PageContainer title="Listing 任务详情">
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/listing-tasks')}>返回列表</Button>
        {task?.status === 'failed' ? (
          <Button type="primary" icon={<RedoOutlined />} loading={retryMutation.isPending} onClick={() => retryMutation.mutate()}>重试失败项</Button>
        ) : isProduction ? (
          <>
            <Button
              type="primary"
              danger
              icon={<PlayCircleOutlined />}
              loading={executeMutation.isPending}
              onClick={() => setConfirmOpen(true)}
              disabled={task?.status === 'completed' || task?.status === 'executing'}
            >
              启动执行
            </Button>
            <HighRiskConfirmDialog
              open={confirmOpen}
              riskLevel="high"
              actionName="生产发布"
              detail={{ targetLabel: `listing_task:${id}` }}
              expectedConsequence="商品将发布到平台并对外可见。"
              auditDestination="审计记录将写入 operation_log。"
              rollbackNote="如平台发布失败，任务会进入 failed 状态并保留重试入口。"
              confirmText="确认启动"
              confirmLoading={executeMutation.isPending}
              environmentMode="production"
              onCancel={() => setConfirmOpen(false)}
              onConfirm={() => {
                executeMutation.mutate();
                setConfirmOpen(false);
              }}
            />
          </>
        ) : task?.status === 'blocked' ? (
          <Button type="primary" onClick={() => router.push('/approval')}>去审批</Button>
        ) : (
          <Button
            type="primary"
            icon={<PlayCircleOutlined />}
            loading={executeMutation.isPending}
            onClick={() => executeMutation.mutate()}
            disabled={task?.status === 'completed' || task?.status === 'executing'}
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
          {task?.status !== 'completed' && task?.status !== 'failed' && (
            <Alert
              type={task?.status === 'approved' ? 'success' : task?.status === 'blocked' && (isApproved || isRejected) ? (isApproved ? 'success' : 'error') : 'warning'}
              message={
                task?.status === 'approved' ? '审批已通过，可以执行'
                : task?.status === 'blocked' && isApproved ? '审批已通过，可以执行'
                : task?.status === 'blocked' && isRejected ? '审批已拒绝，任务保持阻塞'
                : task?.status === 'executing' ? '任务正在执行中'
                : '该任务等待 Owner 审批'
              }
              description={publishApproval?.reason || '审批通过后，系统将允许该任务的执行。'}
              showIcon
            />
          )}
          {task?.status === 'failed' && task?.last_error ? <Alert type="error" message="任务执行失败" description={task.last_error} showIcon /> : null}
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
                <Tag color={task?.status === 'completed' ? 'green' : task?.status === 'executing' ? 'orange' : task?.status === 'approved' ? 'blue' : task?.status === 'failed' ? 'red' : 'default'}>
                  {task?.status?.toUpperCase()}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="执行模式"><Tag color={modeInfo.color}>{modeInfo.label}</Tag></Descriptions.Item>
              <Descriptions.Item label="审批 ID">{task?.approval_id ?? '-'}</Descriptions.Item>
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
              {task?.decision_snapshot != null ? (
                <Descriptions.Item label="评估快照" span={2}>
                  <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: 12, color: '#888' }}>
                    {JSON.stringify(task.decision_snapshot, null, 2)}
                  </pre>
                </Descriptions.Item>
              ) : null}
            </Descriptions>
          </Card>

          {/* External Reference */}
          {task?.external_reference_id ? <Card title="平台引用" size="small">
            <Descriptions column={1} size="small">
              <Descriptions.Item label="平台商品 ID">{task.external_reference_id}</Descriptions.Item>
              {task.external_reference_url ? <Descriptions.Item label="平台 URL"><a href={task.external_reference_url} target="_blank" rel="noopener">{task.external_reference_url}</a></Descriptions.Item> : null}
            </Descriptions>
          </Card> : null}

          <Card title="任务执行明细">
            <Table dataSource={items} columns={cols} rowKey="id" pagination={false} size="small" />
          </Card>
        </Space>
      )}
    </PageContainer>
  );
}
