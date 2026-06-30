'use client';

import { Button, Card, Empty, Spin, Table, Tag, message } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import { useSearchParams } from 'next/navigation';
import { Suspense } from 'react';

interface LifecycleStep {
  id: number;
  product_id: number;
  step: string;
  agent_id: string;
  status: string;
  started_at?: string;
  completed_at?: string;
  duration_ms: number;
  result?: string;
  error?: string;
}

const STEP_LABELS: Record<string, string> = {
  sourcing: '选品',
  enrichment: '内容生成',
  compliance: '合规检查',
  pricing: '定价',
  listing: '刊登',
  monitoring: '监控',
  delisting: '下架',
};

const STATUS_COLORS: Record<string, string> = {
  pending: 'default',
  running: 'processing',
  completed: 'success',
  failed: 'error',
  skipped: 'warning',
};

function PipelineContent() {
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const productId = searchParams.get('product_id');

  const { data: steps, isLoading } = useQuery({
    queryKey: ['orchestration', 'pipeline', productId],
    queryFn: async () => {
      if (!productId) return [];
      const res = await apiClient.get<LifecycleStep[]>(`/v1/orchestration/products/${productId}/pipeline`);
      return res.data ?? [];
    },
    enabled: !!productId,
  });

  if (!productId) {
    return <Empty description="请提供 product_id 查询参数" />;
  }

  if (isLoading) {
    return <Spin tip="加载中..." style={{ display: 'block', marginTop: 48 }} />;
  }

  if (!steps || steps.length === 0) {
    return <Empty description="该产品尚未启动生命周期" />;
  }

  const totalDuration = steps.reduce((sum, s) => sum + s.duration_ms, 0);
  const completedCount = steps.filter((s) => s.status === 'completed').length;

  return (
    <div>
      {/* Summary Card */}
      <Card style={{ marginBottom: 24 }}>
        <div style={{ display: 'flex', gap: 48 }}>
          <div>
            <div style={{ color: '#999', fontSize: 12 }}>产品 ID</div>
            <div style={{ fontSize: 20, fontWeight: 700 }}>#{productId}</div>
          </div>
          <div>
            <div style={{ color: '#999', fontSize: 12 }}>进度</div>
            <div style={{ fontSize: 20, fontWeight: 700 }}>{completedCount}/{steps.length}</div>
          </div>
          <div>
            <div style={{ color: '#999', fontSize: 12 }}>总耗时</div>
            <div style={{ fontSize: 20, fontWeight: 700 }}>{Math.round(totalDuration / 1000)}s</div>
          </div>
        </div>
      </Card>

      {/* Stepped progress visualization */}
      <Card
        title="流水线进度"
        extra={
          <Button icon={<ReloadOutlined />} onClick={() => queryClient.invalidateQueries({ queryKey: ['orchestration', 'pipeline', productId] })}>
            刷新
          </Button>
        }
        style={{ marginBottom: 24 }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 0, overflowX: 'auto', padding: '24px 8px' }}>
          {steps.map((step, index) => {
            const isLast = index === steps.length - 1;
            const isActive = step.status === 'running';
            const isCompleted = step.status === 'completed';
            const isFailed = step.status === 'failed';
            const isSkipped = step.status === 'skipped';

            let circleColor = '#d9d9d9';
            if (isCompleted) circleColor = '#52c41a';
            else if (isActive) circleColor = '#1677ff';
            else if (isFailed) circleColor = '#ff4d4f';
            else if (isSkipped) circleColor = '#faad14';

            return (
              <div key={step.id} style={{ display: 'flex', alignItems: 'center', flex: 1, minWidth: 140 }}>
                <div style={{
                  textAlign: 'center', flex: 1,
                  padding: '12px 8px', borderRadius: 8,
                  background: isActive ? '#f0f5ff' : undefined,
                  border: isActive ? '1px solid #91caff' : undefined,
                }}>
                  <div style={{
                    width: 44, height: 44, borderRadius: '50%',
                    backgroundColor: circleColor, color: '#fff',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    margin: '0 auto', fontWeight: 700, fontSize: 16,
                    boxShadow: isActive ? '0 0 0 4px #e6f4ff' : 'none',
                    transition: 'all 0.3s',
                  }}>
                    {isCompleted ? '✓' : isFailed ? '✗' : index + 1}
                  </div>
                  <div style={{ marginTop: 10, fontWeight: 600, fontSize: 14 }}>{STEP_LABELS[step.step] || step.step}</div>
                  <div style={{ fontSize: 12, color: '#666', marginTop: 2 }}>{step.agent_id}</div>
                  <div style={{ marginTop: 6 }}>
                    <Tag color={STATUS_COLORS[step.status]}>{step.status}</Tag>
                  </div>
                  {step.duration_ms > 0 && (
                    <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
                      {(step.duration_ms / 1000).toFixed(1)}s
                    </div>
                  )}
                  {step.error && (
                    <div style={{ fontSize: 11, color: '#ff4d4f', marginTop: 4, maxWidth: 160, wordBreak: 'break-word', background: '#fff2f0', padding: '2px 6px', borderRadius: 4 }}>
                      {step.error}
                    </div>
                  )}
                  {isFailed && (
                    <Button
                      type="link"
                      size="small"
                      style={{ marginTop: 8 }}
                      onClick={async () => {
                        try {
                          await apiClient.post(`/v1/orchestration/products/${productId}/pipeline/step/${step.step}/retry`);
                          message.success('重试已触发');
                          queryClient.invalidateQueries({ queryKey: ['orchestration', 'pipeline', productId] });
                        } catch (err: unknown) {
                          const e = err as Error;
                          message.error(`重试失败: ${e.message}`);
                        }
                      }}
                    >
                      重试
                    </Button>
                  )}
                </div>
                {!isLast && (
                  <div style={{
                    flex: 1, height: 3, backgroundColor: circleColor, margin: '0 8px',
                    marginTop: -80, borderRadius: 2,
                  }} />
                )}
              </div>
            );
          })}
        </div>
      </Card>

      {/* Detail Table */}
      <Card title="步骤详情">
        <Table
          rowKey="id"
          dataSource={steps}
          pagination={false}
          size="small"
          columns={[
            { title: '步骤', dataIndex: 'step', width: 120, render: (val: string) => STEP_LABELS[val] || val },
            { title: 'Agent', dataIndex: 'agent_id', width: 100 },
            { title: '状态', dataIndex: 'status', width: 100, render: (val: string) => <Tag color={STATUS_COLORS[val]}>{val}</Tag> },
            { title: '耗时(ms)', dataIndex: 'duration_ms', width: 100 },
            {
              title: '开始时间', dataIndex: 'started_at', width: 160,
              render: (val: string) => val ? val.slice(0, 16).replace('T', ' ') : '-',
            },
            {
              title: '完成时间', dataIndex: 'completed_at', width: 160,
              render: (val: string) => val ? val.slice(0, 16).replace('T', ' ') : '-',
            },
            {
              title: '结果', dataIndex: 'result',
              render: (val: string) => val ? <pre style={{ margin: 0, fontSize: 11, maxWidth: 200, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{val}</pre> : '-',
            },
            {
              title: '错误', dataIndex: 'error',
              render: (val: string) => val ? <span style={{ color: '#ff4d4f' }}>{val}</span> : '-',
            },
          ]}
        />
      </Card>
    </div>
  );
}

export default function PipelineDetailPage() {
  return (
    <div style={{ padding: 24 }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', marginBottom: 24 }}>
        流水线详情
      </h1>
      <Suspense fallback={<Spin tip="加载中..." style={{ display: 'block', marginTop: 48 }} />}>
        <PipelineContent />
      </Suspense>
    </div>
  );
}
