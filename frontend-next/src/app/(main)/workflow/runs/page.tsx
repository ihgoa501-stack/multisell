'use client';

import { useState } from 'react';
import { Badge, Button, Card, Descriptions, message, Modal, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { BadgeProps } from 'antd';
import {
  ReloadOutlined, PauseCircleOutlined, PlayCircleOutlined, InfoCircleOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import dayjs from 'dayjs';

const { Text } = Typography;

interface WorkflowStepRun {
  id: number;
  step_name: string;
  step_type: string;
  status: string;
  input: string;
  output: string;
  error: string;
  attempt: number;
  max_attempts: number;
  started_at: string | null;
  completed_at: string | null;
  parent_id?: number | null;
}

interface WorkflowRun {
  id: number;
  workflow_def_id: number;
  name: string;
  status: string;
  context: string;
  error: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
  steps?: WorkflowStepRun[];
}

const STATUS_COLORS: Record<string, BadgeProps['status']> = {
  running: 'processing',
  completed: 'success',
  failed: 'error',
  paused: 'warning',
  pending: 'default',
};

const STATUS_LABELS: Record<string, string> = {
  running: '运行中',
  completed: '已完成',
  failed: '失败',
  paused: '已暂停',
  pending: '待运行',
};

const STEP_COLORS: Record<string, string> = {
  running: 'processing',
  completed: 'success',
  failed: 'error',
  skipped: 'default',
  pending: 'default',
};

export default function WorkflowRunsPage() {
  const queryClient = useQueryClient();
  const [detailRun, setDetailRun] = useState<WorkflowRun | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const { data: runs, isLoading } = useQuery({
    queryKey: ['workflow-runs'],
    queryFn: async () => {
      const res = await apiClient.get<WorkflowRun[]>('/v1/workflow/runs');
      return res.data || [];
    },
    refetchInterval: 5000,
  });

  const pauseMutation = useMutation({
    mutationFn: (id: number) => apiClient.post(`/v1/workflow/runs/${id}/pause`),
    onSuccess: () => { message.success('已暂停'); queryClient.invalidateQueries({ queryKey: ['workflow-runs'] }); },
    onError: () => message.error('暂停失败'),
  });

  const resumeMutation = useMutation({
    mutationFn: (id: number) => apiClient.post(`/v1/workflow/runs/${id}/resume`),
    onSuccess: () => { message.success('已恢复'); queryClient.invalidateQueries({ queryKey: ['workflow-runs'] }); },
    onError: () => message.error('恢复失败'),
  });

  const openDetail = async (run: WorkflowRun) => {
    try {
      const res = await apiClient.get<WorkflowRun>(`/v1/workflow/runs/${run.id}`);
      setDetailRun(res.data || run);
    } catch {
      setDetailRun(run);
    }
    setDetailOpen(true);
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', ellipsis: true },
    {
      title: '状态', dataIndex: 'status', width: 90,
      render: (s: string) => <Badge status={STATUS_COLORS[s]} text={STATUS_LABELS[s] || s} />,
    },
    {
      title: '开始时间', dataIndex: 'started_at', width: 160,
      render: (v: string) => v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '-',
    },
    {
      title: '完成时间', dataIndex: 'completed_at', width: 160,
      render: (v: string) => v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '-',
    },
    {
      title: '耗时', width: 80,
      render: (_: unknown, r: WorkflowRun) => {
        if (!r.started_at) return '-';
        const end = r.completed_at ? dayjs(r.completed_at) : dayjs();
        const dur = end.diff(dayjs(r.started_at), 'second');
        return `${dur}s`;
      },
    },
    {
      title: '操作', width: 160,
      render: (_: unknown, record: WorkflowRun) => (
        <Space size="small">
          <Button size="small" icon={<InfoCircleOutlined />} onClick={() => openDetail(record)}>详情</Button>
          {record.status === 'running' && (
            <Tooltip title="暂停">
              <Button size="small" icon={<PauseCircleOutlined />}
                onClick={() => pauseMutation.mutate(record.id)} />
            </Tooltip>
          )}
          {record.status === 'paused' && (
            <Tooltip title="恢复">
              <Button size="small" type="primary" icon={<PlayCircleOutlined />}
                onClick={() => resumeMutation.mutate(record.id)} />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      title="工作流运行"
      subtitle="平台所有工作流的运行实例"
      extra={
        <Button icon={<ReloadOutlined />} onClick={() => queryClient.invalidateQueries({ queryKey: ['workflow-runs'] })}>
          刷新
        </Button>
      }
    >
      <Card size="small" styles={{ body: { padding: 0 } }}>
        <Table<WorkflowRun>
          rowKey="id"
          columns={columns}
          dataSource={runs}
          loading={isLoading}
          scroll={{ x: 800 }}
        />
      </Card>

      {/* Detail modal */}
      <Modal
        title={detailRun ? `运行 #${detailRun.id} 详情` : ''}
        open={detailOpen}
        onCancel={() => { setDetailOpen(false); setDetailRun(null); }}
        footer={null}
        width={700}
      >
        {detailRun && (
          <Space direction="vertical" style={{ width: '100%' }}>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label="名称">{detailRun.name}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Badge status={STATUS_COLORS[detailRun.status]} text={STATUS_LABELS[detailRun.status] || detailRun.status} />
              </Descriptions.Item>
              <Descriptions.Item label="开始时间">
                {detailRun.started_at ? dayjs(detailRun.started_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="完成时间">
                {detailRun.completed_at ? dayjs(detailRun.completed_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="错误" span={2}>
                <Text type="danger">{detailRun.error || '-'}</Text>
              </Descriptions.Item>
            </Descriptions>

            <Text strong style={{ marginTop: 12 }}>步骤执行</Text>
            <Table<WorkflowStepRun>
              rowKey="id"
              size="small"
              dataSource={detailRun.steps || []}
              columns={[
                { title: '步骤', dataIndex: 'step_name' },
                {
                  title: '类型', dataIndex: 'step_type', width: 80,
                  render: (t: string) => <Tag>{t}</Tag>,
                },
                {
                  title: '状态', dataIndex: 'status', width: 80,
                  render: (s: string) => <Tag color={STEP_COLORS[s]}>{s}</Tag>,
                },
                { title: '尝试', dataIndex: 'attempt', width: 60 },
                {
                  title: '耗时', width: 70,
                  render: (_: unknown, sr: WorkflowStepRun) => {
                    if (!sr.started_at) return '-';
                    const end = sr.completed_at ? dayjs(sr.completed_at) : dayjs();
                    return `${end.diff(dayjs(sr.started_at), 'second')}s`;
                  },
                },
                {
                  title: '错误', dataIndex: 'error', ellipsis: true,
                  render: (e: string) => e ? <Text type="danger" style={{ fontSize: '0.78rem' }}>{e}</Text> : '-',
                },
              ]}
              pagination={false}
              scroll={{ x: 500 }}
            />
          </Space>
        )}
      </Modal>
    </PageContainer>
  );
}
