'use client';

import { useEffect, useState } from 'react';
import {
  Button, Card, Col, Progress, Row, Statistic, Table, Tag, Switch, message,
} from 'antd';
import {
  ExperimentOutlined, ReloadOutlined, DeleteOutlined,
  RiseOutlined, FallOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';

dayjs.extend(relativeTime);

interface MetabolismLog {
  id: number;
  event_id: number;
  source: string;
  total_score: number;
  impact_score: number;
  ref_score: number;
  freshness_score: number;
  semantic_score: number;
  sem_skipped: boolean;
  excretable: boolean;
  reason: string;
  created_at: string;
}

export default function MetabolismPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [dryRun, setDryRun] = useState(true);

  // Fetch metabolism logs
  const { data, isLoading } = useQuery({
    queryKey: ['metabolism', page],
    queryFn: async () => {
      const res = await apiClient.get<{ data: MetabolismLog[]; total: number }>('/v1/metabolism', {
        page: String(page),
        page_size: '20',
      });
      return res.data;
    },
    refetchInterval: 30_000,
  });

  // Trigger M1 dry-run
  const dryRunMutation = useMutation({
    mutationFn: async () => {
      await apiClient.post('/v1/metabolism/dry-run', {});
    },
    onSuccess: () => {
      message.success('M1 代谢评分已触发');
      queryClient.invalidateQueries({ queryKey: ['metabolism'] });
    },
    onError: () => message.error('触发失败'),
  });

  // Compute stats
  const logs = data?.data ?? [];
  const total = data?.total ?? 0;
  const avgScore = logs.length
    ? (logs.reduce((s, l) => s + l.total_score, 0) / logs.length).toFixed(3)
    : '0';
  const excretableCount = logs.filter((l) => l.excretable).length;
  const scoreRanges = {
    low: logs.filter((l) => l.total_score < 0.3).length,
    mid: logs.filter((l) => l.total_score >= 0.3 && l.total_score < 0.7).length,
    high: logs.filter((l) => l.total_score >= 0.7).length,
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 100,
      render: (s: string) => <Tag>{s}</Tag>,
    },
    {
      title: '总分',
      dataIndex: 'total_score',
      key: 'total_score',
      width: 80,
      render: (v: number) => (
        <span style={{ color: v >= 0.7 ? '#ff4d4f' : '#52c41a', fontWeight: 600 }}>
          {v.toFixed(2)}
        </span>
      ),
    },
    {
      title: '维度',
      key: 'dims',
      width: 160,
      render: (_: unknown, r: MetabolismLog) => (
        <span style={{ fontSize: 12, color: '#888' }}>
          I:{r.impact_score.toFixed(2)} R:{r.ref_score.toFixed(2)}
          F:{r.freshness_score.toFixed(2)}
          {!r.sem_skipped && ` S:${r.semantic_score.toFixed(2)}`}
        </span>
      ),
    },
    {
      title: '可排泄',
      dataIndex: 'excretable',
      key: 'excretable',
      width: 80,
      render: (v: boolean) => v
        ? <Tag color="red">是</Tag>
        : <Tag color="green">否</Tag>,
    },
    {
      title: '原因',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 120,
      render: (v: string) => dayjs(v).fromNow(),
    },
  ];

  return (
    <PageContainer
      title="代谢管理"
      subtitle="M1 排泄系统 — 四维评分引擎状态监控"
    >
      {/* Stats cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="已评分事件"
              value={total}
              prefix={<RiseOutlined />}
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="可排泄率"
              value={total > 0 ? ((excretableCount / total) * 100).toFixed(1) : 0}
              suffix="%"
              prefix={<DeleteOutlined />}
              valueStyle={{ color: excretableCount > 0 ? '#ff4d4f' : '#52c41a' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="平均分"
              value={avgScore}
              precision={3}
              prefix={<ExperimentOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="评分分布"
              value={`${scoreRanges.low}/${scoreRanges.mid}/${scoreRanges.high}`}
              prefix={<RiseOutlined />}
              suffix="(低/中/高)"
            />
          </Card>
        </Col>
      </Row>

      {/* Score distribution bar */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <div style={{ marginBottom: 8 }}>评分分布</div>
        <Progress
          percent={total > 0 ? (scoreRanges.low / total) * 100 : 0}
          strokeColor="#52c41a"
          format={() => `低 ${scoreRanges.low}`}
          style={{ marginBottom: 4 }}
        />
        <Progress
          percent={total > 0 ? (scoreRanges.mid / total) * 100 : 0}
          strokeColor="#1677ff"
          format={() => `中 ${scoreRanges.mid}`}
          style={{ marginBottom: 4 }}
        />
        <Progress
          percent={total > 0 ? (scoreRanges.high / total) * 100 : 0}
          strokeColor="#ff4d4f"
          format={() => `高 ${scoreRanges.high}`}
        />
      </Card>

      {/* Actions */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <span>Dry-run 模式:</span>
          <Switch
            checked={dryRun}
            onChange={(v) => {
              setDryRun(v);
              message.info(dryRun ? '切换到评分+排泄模式' : '切换到仅评分模式');
            }}
          />
          <Button
            type="primary"
            icon={<ReloadOutlined />}
            loading={dryRunMutation.isPending}
            onClick={() => dryRunMutation.mutate()}
          >
            立即触发评分
          </Button>
          <span style={{ fontSize: 12, color: '#888' }}>
            {dryRun ? '仅评分，不执行排泄' : '评分 + 实际排泄'}
          </span>
        </div>
      </Card>

      {/* Log table */}
      <Card>
        <Table
          dataSource={logs}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={{
            current: page,
            pageSize: 20,
            total,
            onChange: setPage,
            showSizeChanger: false,
          }}
          size="small"
        />
      </Card>
    </PageContainer>
  );
}
