'use client';

import { useState } from 'react';
import { Alert, Card, Descriptions, Table, Spin, Result, Button, Space, Tag, message } from 'antd';
import HighRiskConfirmDialog from '@/components/ui/HighRiskConfirmDialog';
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
  execution_mode: number;
  approval_id?: number;
  external_reference_id?: string;
  platform_validation?: Array<{ field: string; valid: boolean }>;
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

const EXECUTION_MODE_LABELS: Record<number, string> = {
  0: 'Dry-Run',
  1: 'Sandbox',
  2: 'Approval Required',
  3: 'Production',
};

const EXECUTION_MODE_COLORS: Record<number, string> = {
  0: 'default',
  1: 'orange',
  2: 'purple',
  3: 'red',
};

function needsExecutionConfirmation(mode?: number) {
  return mode === 2 || mode === 3;
}

function executionModeToEnvironment(mode?: number): 'dry_run' | 'sandbox' | 'production' {
  if (mode === 0) return 'dry_run';
  if (mode === 1) return 'sandbox';
  return 'production';
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

  const [executeDialogOpen, setExecuteDialogOpen] = useState(false);

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
          <>
            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              loading={executeMutation.isPending}
              onClick={() => {
                if (needsExecutionConfirmation(task?.execution_mode)) setExecuteDialogOpen(true);
                else executeMutation.mutate();
              }}
              disabled={task?.status === 'success' || task?.status === 'running'}
            >
              启动执行
            </Button>
            <HighRiskConfirmDialog
              open={executeDialogOpen}
              actionName="执行刊登任务"
              riskLevel="high"
              detail={{ targetLabel: task?.id ? `Listing Task #${task.id}` : '' }}
              environmentMode={executionModeToEnvironment(task?.execution_mode)}
              expectedConsequence="将发布商品到线上平台，买家可见，此操作不可撤回"
              confirmLoading={executeMutation.isPending}
              onConfirm={() => { executeMutation.mutate(); setExecuteDialogOpen(false); }}
              onCancel={() => setExecuteDialogOpen(false)}
            />
          </>
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
                <Space>
                  <Tag color={task?.status === 'success' ? 'green' : task?.status === 'running' ? 'orange' : task?.status === 'failed' ? 'red' : 'default'}>
                    {task?.status?.toUpperCase()}
                  </Tag>
                  {task?.execution_mode !== undefined && (
                    <Tag color={EXECUTION_MODE_COLORS[task.execution_mode] ?? 'default'}>
                      {EXECUTION_MODE_LABELS[task.execution_mode] ?? 'Unknown'}
                    </Tag>
                  )}
                </Space>
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

          {/* ===== Phase 3: External Reference ===== */}
          {task?.external_reference_id && (
            <Card title="外部参考" size="small">
              <Descriptions column={1} size="small">
                <Descriptions.Item label="外部平台ID">{task.external_reference_id}</Descriptions.Item>
              </Descriptions>
            </Card>
          )}

          {/* ===== Phase 3: Platform Validation ===== */}
          {task?.platform_validation && task.platform_validation.length > 0 && (
            <Card title="平台字段校验" size="small">
              <Table
                dataSource={task.platform_validation}
                rowKey="field"
                size="small"
                pagination={false}
                columns={[
                  { title: '字段', dataIndex: 'field', key: 'field', render: (v: string) => v },
                  { title: '状态', dataIndex: 'valid', key: 'valid', render: (v: boolean) => v ? <Tag color="green">有效</Tag> : <Tag color="red">无效</Tag> },
                ]}
              />
            </Card>
          )}

          {/* ===== Phase 3: Failure Section ===== */}
          {task?.status === 'failed' && (
            <Card title="失败详情" size="small">
              <pre style={{ color: '#ff4d4f', whiteSpace: 'pre-wrap', margin: 0 }}>{task.last_error || '未知错误'}</pre>
              <Button type="primary" icon={<RedoOutlined />} loading={retryMutation.isPending} onClick={() => retryMutation.mutate()} style={{ marginTop: 8 }}>
                重试
              </Button>
            </Card>
          )}

          <Card title="任务执行明细">
            <Table
              dataSource={items}
              columns={columns}
              rowKey="id"
              pagination={false}
              size="small"
            />
          </Card>
          {/* Execution info */}
          {task && (
            <Card size="small" title="执行信息" style={{ marginTop: 16 }}>
              <Descriptions column={1} size="small">
                <Descriptions.Item label="执行模式">
                  <Tag color={EXECUTION_MODE_COLORS[task.execution_mode ?? 0]}>
                    {EXECUTION_MODE_LABELS[task.execution_mode ?? 0] ?? 'Unknown'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="审批 ID">
                  {(task.approval_id ?? publishApproval?.id) ? (
                    <Button type="link" size="small" onClick={() => router.push(`/approval/${task.approval_id ?? publishApproval?.id}`)}>
                      #{task.approval_id ?? publishApproval?.id}
                    </Button>
                  ) : '-'}
                </Descriptions.Item>
                <Descriptions.Item label="外部引用 ID">
                  {task.external_reference_id || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="审计记录">
                  <Button type="link" size="small" onClick={() => router.push(`/operation-logs?resource=listing_task:${task.id}`)}>
                    查看操作日志
                  </Button>
                </Descriptions.Item>
              </Descriptions>
              <div style={{ marginTop: 8 }}>
                <Button type="link" size="small" onClick={() => router.push(`/sandbox-listing?candidate_id=${task.product_id}`)}>
                  查看关联证据卡
                </Button>
              </div>
            </Card>
          )}
        </Space>
      )}
    </PageContainer>
  );
}
