'use client';

import { useState } from 'react';
import { Card, Button, Tag, Spin, Result, Descriptions, Alert, message } from 'antd';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useSandboxListingStore } from '../store';
import apiClient from '@/lib/api-client';
import { useRouter } from 'next/navigation';

export default function StepExecution() {
  const { listingTaskId, goBack } = useSandboxListingStore();
  const router = useRouter();
  const [executed, setExecuted] = useState(false);

  const { data: task, isLoading, refetch } = useQuery({
    queryKey: ['listing-task', listingTaskId],
    queryFn: () => apiClient.get(`/v1/listing-tasks/${listingTaskId}`).then(r => (r.data as { task: { status: string; last_error?: string; approval_id?: number; external_reference_id?: string } }).task),
    enabled: !!listingTaskId,
  });

  const executeMutation = useMutation({
    mutationFn: () => apiClient.post(`/v1/listing-task/${listingTaskId}/execute`),
    onSuccess: () => {
      message.success('沙箱任务执行成功');
      setExecuted(true);
      refetch();
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : '执行失败';
      message.error(msg);
      setExecuted(true);
      refetch();
    },
  });

  if (isLoading) return <Spin />;

  const isSuccess = task?.status === 'completed' || task?.status === 'approved';
  const isFailed = task?.status === 'failed';
  const isPending = !executed && (task?.status === 'blocked' || task?.status === 'pending' || task?.status === 'approved');

  return (
    <div>
      <Card title="执行沙箱上架任务">
        {isPending && (
          <div>
            <Result status="info" title="等待执行" subTitle={`点击"执行沙箱任务"开始`} />
            <div style={{ display: 'flex', gap: 8, justifyContent: 'center' }}>
              <Button onClick={goBack}>返回</Button>
              <Button type="primary" loading={executeMutation.isPending}
                onClick={() => executeMutation.mutate()}>
                执行沙箱任务
              </Button>
            </div>
          </div>
        )}

        {isSuccess && (
          <Result
            status="success"
            title="沙箱执行成功"
            subTitle="任务已完成"
            extra={[
              <Button key="detail" type="primary"
                onClick={() => router.push(`/listing-tasks/${listingTaskId}`)}>
                查看完整任务详情
              </Button>,
            ]}
          />
        )}

        {isFailed && (
          <Result
            status="error"
            title="执行失败"
            subTitle={task?.last_error || '未知错误'}
            extra={[
              <Button key="detail" type="primary"
                onClick={() => router.push(`/listing-tasks/${listingTaskId}`)}>
                查看失败详情
              </Button>,
              <Button key="retry" onClick={() => executeMutation.mutate()}>重试</Button>,
            ]}
          />
        )}
      </Card>

      {/* Execution info */}
      {(isSuccess || isFailed) && (
        <Card size="small" title="执行信息">
          <Descriptions column={1} size="small">
            <Descriptions.Item label="状态"><Tag>{task?.status}</Tag></Descriptions.Item>
            <Descriptions.Item label="执行模式">
              <Tag color="orange">Sandbox</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="审批 ID">{task?.approval_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="外部引用 ID">{task?.external_reference_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="审计记录">
              <Button type="link" size="small"
                onClick={() => router.push(`/operation-logs?resource=listing_task:${listingTaskId}`)}>
                查看审计日志
              </Button>
            </Descriptions.Item>
          </Descriptions>
        </Card>
      )}
    </div>
  );
}
