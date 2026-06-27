'use client';

import { useMemo, useState } from 'react';
import { Card, Col, Row, Space, Statistic, Table, Tag, Button, message } from 'antd';
import {
  ExperimentOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import type { PageResult } from '@/types/api';

// ─────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────

interface MetabolismLog {
  id: number;
  source: string;
  score: number;
  dims: string;
  excretable: boolean;
  reason?: string;
  created_at: string;
}

interface MetabolismSummary {
  total_scored: number;
  excretable_count: number;
  excretable_rate: number;
  avg_score: number;
  last_m1_run_at: string;
  score_distribution: ScoreBucket[];
}

interface ScoreBucket {
  range: string;
  min: number;
  max: number;
  count: number;
}

// ─────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────

function scoreColor(score: number): string {
  if (score >= 0.7) return 'red';
  if (score >= 0.5) return 'orange';
  if (score >= 0.3) return 'blue';
  return 'green';
}

function scoreLabel(score: number): string {
  return (score * 100).toFixed(0) + '%';
}

function buildScoreBuckets(logs: MetabolismLog[]): ScoreBucket[] {
  const buckets: ScoreBucket[] = [
    { range: '0 – 0.3', min: 0, max: 0.3, count: 0 },
    { range: '0.3 – 0.5', min: 0.3, max: 0.5, count: 0 },
    { range: '0.5 – 0.7', min: 0.5, max: 0.7, count: 0 },
    { range: '0.7 – 1.0', min: 0.7, max: 1.0, count: 0 },
  ];

  for (const log of logs) {
    const bucket = buckets.find(
      (b) => log.score >= b.min && log.score < b.max,
    );
    if (bucket) bucket.count += 1;
  }

  return buckets;
}

// ─────────────────────────────────────────────────────────
// Page
// ─────────────────────────────────────────────────────────

export default function MetabolismPage() {
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const pageSize = 20;

  // ── Paginated log list with 30 s auto-refresh ──
  const { data: listData, isLoading } = useQuery<PageResult<MetabolismLog>>({
    queryKey: ['metabolism', 'logs', page],
    queryFn: () =>
      apiClient.getPage<MetabolismLog>('/v1/metabolism', {
        page: String(page),
        page_size: String(pageSize),
      }),
    refetchInterval: 30_000,
  });

  const logs = listData?.data ?? [];
  const total = listData?.total ?? 0;

  // ── Stats computed from the first page of logs ──
  // In a production setup a dedicated /v1/metabolism/stats endpoint
  // would provide accurate aggregates across all records.
  const stats = useMemo<MetabolismSummary>(() => {
    const totalScored = total;
    const excretableCount = logs.filter((l) => l.excretable).length;
    const excretableRate =
      totalScored > 0 ? excretableCount / totalScored : 0;
    const avgScore =
      logs.length > 0
        ? logs.reduce((s, l) => s + l.score, 0) / logs.length
        : 0;
    const lastRun = logs.length > 0 ? logs[0].created_at : '';
    const buckets = buildScoreBuckets(logs);

    return {
      total_scored: totalScored,
      excretable_count: excretableCount,
      excretable_rate: excretableRate,
      avg_score: avgScore,
      last_m1_run_at: lastRun,
      score_distribution: buckets,
    };
  }, [logs, total]);

  const bucketMax = useMemo(
    () => Math.max(...stats.score_distribution.map((b) => b.count), 1),
    [stats.score_distribution],
  );

  // ── Dry-run ──
  const dryRunMut = useMutation({
    mutationFn: () =>
      apiClient.post<unknown>('/v1/metabolism/dry-run'),
    onSuccess: () => {
      message.success('Dry-run 完成');
      qc.invalidateQueries({ queryKey: ['metabolism'] });
    },
    onError: () => message.error('Dry-run 失败'),
  });

  // ── Manual M1 trigger ──
  const triggerMut = useMutation({
    mutationFn: () => apiClient.post<unknown>('/v1/metabolism/run'),
    onSuccess: () => {
      message.success('M1 触发成功');
      qc.invalidateQueries({ queryKey: ['metabolism'] });
    },
    onError: () => message.error('M1 触发失败'),
  });

  // ── Table columns ──
  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: 'Source',
      dataIndex: 'source',
      width: 130,
    },
    {
      title: 'Score',
      dataIndex: 'score',
      width: 100,
      render: (v: number) => (
        <Tag color={scoreColor(v)}>{scoreLabel(v)}</Tag>
      ),
    },
    {
      title: 'Dims',
      dataIndex: 'dims',
      width: 220,
      ellipsis: true,
    },
    {
      title: 'Excretable',
      dataIndex: 'excretable',
      width: 100,
      render: (v: boolean) =>
        v ? (
          <Tag color="red">是</Tag>
        ) : (
          <Tag color="default">否</Tag>
        ),
    },
    {
      title: 'Reason',
      dataIndex: 'reason',
      ellipsis: true,
    },
    {
      title: 'Time',
      dataIndex: 'created_at',
      width: 160,
      render: (t: string) =>
        t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '-',
    },
  ];

  // ── Render ──
  return (
    <div style={{ padding: 24 }}>
      {/* ── Header ── */}
      <Space
        style={{
          marginBottom: 20,
          justifyContent: 'space-between',
          width: '100%',
        }}
        align="start"
      >
        <div>
          <h1
            style={{
              fontFamily: 'var(--ds)',
              fontWeight: 700,
              fontSize: 'var(--text-h1)',
              margin: 0,
            }}
          >
            <ExperimentOutlined style={{ marginRight: 8 }} />
            代谢管理
          </h1>
        </div>

        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => dryRunMut.mutate()}
            loading={dryRunMut.isPending}
          >
            Dry-Run
          </Button>
          <Button
            type="primary"
            icon={<ThunderboltOutlined />}
            onClick={() => triggerMut.mutate()}
            loading={triggerMut.isPending}
          >
            触发 M1
          </Button>
        </Space>
      </Space>

      {/* ── Stats cards ── */}
      <Row gutter={[16, 16]} style={{ marginBottom: 20 }}>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic title="总评分项" value={stats.total_scored} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="可排出率"
              value={stats.excretable_rate}
              precision={2}
              valueStyle={{
                color:
                  stats.excretable_rate > 0.5 ? '#cf1322' : '#52c41a',
              }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="平均分数"
              value={stats.avg_score}
              precision={3}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="上次 M1 运行"
              value={
                stats.last_m1_run_at
                  ? dayjs(stats.last_m1_run_at).format('MM-DD HH:mm')
                  : 'N/A'
              }
            />
          </Card>
        </Col>
      </Row>

      {/* ── Score distribution bar chart ── */}
      <Card title="分数分布" style={{ marginBottom: 20 }}>
        {stats.score_distribution.length === 0 ? (
          <div
            style={{
              textAlign: 'center',
              padding: 24,
              color: '#999',
            }}
          >
            暂无数据
          </div>
        ) : (
          <div
            style={{
              display: 'flex',
              gap: 16,
              alignItems: 'flex-end',
              height: 160,
              padding: '0 16px',
            }}
          >
            {stats.score_distribution.map((bucket) => (
              <div
                key={bucket.range}
                style={{
                  flex: 1,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  height: '100%',
                  justifyContent: 'flex-end',
                }}
              >
                <div
                  style={{
                    fontSize: 12,
                    marginBottom: 4,
                    color: '#666',
                  }}
                >
                  {bucket.count}
                </div>
                <div
                  style={{
                    width: '100%',
                    maxWidth: 60,
                    height: `${(bucket.count / bucketMax) * 120}px`,
                    background:
                      'linear-gradient(180deg, #1677ff 0%, #69b1ff 100%)',
                    borderRadius: '4px 4px 0 0',
                    transition: 'height .3s',
                    minHeight: bucket.count > 0 ? 6 : 2,
                  }}
                />
                <div
                  style={{
                    fontSize: 11,
                    marginTop: 6,
                    color: '#999',
                  }}
                >
                  {bucket.range}
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* ── Log table ── */}
      <Card
        title="代谢日志"
        styles={{ body: { padding: 0 } }}
      >
        <Table
          dataSource={logs}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={{
            current: page,
            pageSize,
            total,
            onChange: (p) => setPage(p),
            showTotal: (t) => `共 ${t} 条`,
          }}
          size="middle"
        />
      </Card>
    </div>
  );
}
