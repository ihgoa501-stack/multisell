'use client';

import {
  Card, Col, Row, Statistic, Table, Tag, Typography,
} from 'antd';
import {
  CheckCircleOutlined, CloseCircleOutlined, ClockCircleOutlined,
  PauseCircleOutlined, PlayCircleOutlined,
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import { StatRowSkeleton } from '@/components/ui/PageSkeleton';

const { Text } = Typography;

interface MonitorStats {
  total_runs: number;
  by_status: Record<string, number>;
  average_duration_s: number;
  failure_by_step: Record<string, number>;
}

const STATUS_ICONS: Record<string, React.ReactNode> = {
  running: <PlayCircleOutlined style={{ color: 'var(--b4)' }} />,
  completed: <CheckCircleOutlined style={{ color: 'var(--g4)' }} />,
  failed: <CloseCircleOutlined style={{ color: 'var(--r4)' }} />,
  paused: <PauseCircleOutlined style={{ color: 'var(--y4)' }} />,
  pending: <ClockCircleOutlined style={{ color: 'var(--t3)' }} />,
};

const STATUS_LABELS: Record<string, string> = {
  running: '运行中',
  completed: '已完成',
  failed: '失败',
  paused: '已暂停',
  pending: '待运行',
};

export default function WorkflowMonitorPage() {
  const { data: stats, isLoading } = useQuery({
    queryKey: ['workflow-monitor'],
    queryFn: async () => {
      const res = await apiClient.get<MonitorStats>('/v1/workflow/monitor/stats');
      return res.data;
    },
    refetchInterval: 10000,
  });

  if (isLoading) {
    return (
      <PageContainer title="工作流监控" subtitle="平台工作流运行状态概览">
        <StatRowSkeleton count={4} />
      </PageContainer>
    );
  }

  if (!stats) {
    return (
      <PageContainer title="工作流监控" subtitle="平台工作流运行状态概览" empty emptyDesc="暂无监控数据" />
    );
  }

  const statusEntries = Object.entries(stats.by_status || {}).sort();
  const failureEntries = Object.entries(stats.failure_by_step || {}).sort((a, b) => b[1] - a[1]);

  return (
    <PageContainer title="工作流监控" subtitle="平台工作流运行状态概览">
      {/* Summary cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 'var(--space-lg)' }}>
        <Col xs={24} sm={12} lg={6}>
          <Card size="small">
            <Statistic title="总运行次数" value={stats.total_runs} suffix="次" />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card size="small">
            <Statistic
              title="平均耗时"
              value={stats.average_duration_s.toFixed(1)}
              suffix="秒"
              precision={1}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card size="small">
            <Statistic
              title="成功率"
              value={
                stats.total_runs > 0
                  ? ((stats.by_status['completed'] || 0) / stats.total_runs * 100).toFixed(1)
                  : 0
              }
              suffix="%"
              precision={1}
              valueStyle={{ color: 'var(--g4)' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card size="small">
            <Statistic
              title="失败次数"
              value={stats.by_status['failed'] || 0}
              suffix="次"
              valueStyle={{ color: 'var(--r4)' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Status distribution */}
      <Row gutter={[16, 16]} style={{ marginBottom: 'var(--space-lg)' }}>
        {statusEntries.map(([status, count]) => (
          <Col xs={12} sm={8} lg={4} key={status}>
            <Card size="small">
              <div style={{ textAlign: 'center' }}>
                <div style={{ fontSize: '1.5rem', marginBottom: 4 }}>
                  {STATUS_ICONS[status] || <ClockCircleOutlined />}
                </div>
                <Text type="secondary" style={{ fontSize: '0.78rem', display: 'block' }}>
                  {STATUS_LABELS[status] || status}
                </Text>
                <Text strong style={{ fontSize: '1.2rem' }}>
                  {count}
                </Text>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      {/* Failure by step */}
      <Card size="small" title="失败步骤分布" style={{ marginBottom: 'var(--space-lg)' }}>
        {failureEntries.length === 0 ? (
          <Text type="secondary">暂无失败记录</Text>
        ) : (
          <Table
            rowKey="0"
            size="small"
            dataSource={failureEntries.map(([step, count], idx) => ({ key: idx, step, count }))}
            columns={[
              { title: '步骤名称', dataIndex: 'step' },
              { title: '失败次数', dataIndex: 'count', width: 100 },
              {
                title: '占比', width: 200,
                render: (_: unknown, r: { count: number }) => {
                  const total = stats.by_status['failed'] || 1;
                  const pct = (r.count / total * 100).toFixed(1);
                  return (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <div style={{
                        height: 8, borderRadius: 4, background: 'var(--r4)',
                        width: `${Math.min(100, parseFloat(pct))}%`,
                        transition: 'width 0.3s',
                      }} />
                      <Text style={{ fontSize: '0.78rem' }}>{pct}%</Text>
                    </div>
                  );
                },
              },
            ]}
            pagination={false}
          />
        )}
      </Card>

      {/* Recent runs summary */}
      <Card size="small" title="运行状态汇总">
        <Row gutter={[16, 16]}>
          {['completed', 'failed', 'running', 'paused', 'pending'].map((status) => (
            <Col xs={12} sm={8} lg={4} key={status}>
              <div style={{
                padding: 'var(--space-sm)',
                background: 'var(--bg-component)',
                borderRadius: 6,
                textAlign: 'center',
              }}>
                <div style={{ fontSize: '0.78rem', color: 'var(--t3)', marginBottom: 4 }}>
                  {STATUS_LABELS[status] || status}
                </div>
                <Tag
                  color={
                    status === 'completed' ? 'success' :
                    status === 'failed' ? 'error' :
                    status === 'running' ? 'processing' :
                    status === 'paused' ? 'warning' : 'default'
                  }
                  style={{ fontSize: '1rem', padding: '2px 12px' }}
                >
                  {stats.by_status[status] || 0}
                </Tag>
              </div>
            </Col>
          ))}
        </Row>
      </Card>
    </PageContainer>
  );
}
