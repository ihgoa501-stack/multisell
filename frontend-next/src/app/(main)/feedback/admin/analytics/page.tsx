'use client';

import { useEffect, useState } from 'react';
import { Typography, Card, Row, Col, Spin, Select, Space, Statistic } from 'antd';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';

const { Title, Text } = Typography;

interface TrendPoint { date: string; count: number }
interface Analytics {
  trend: TrendPoint[];
  by_type: Record<string, number>;
  by_status: Record<string, number>;
  avg_process_hours: number;
  by_agent: Record<string, number>;
}

const statusLabels: Record<string, string> = {
  pending: '待审核', under_review: '审核中', accepted: '已采纳',
  rejected: '已拒绝', planned: '已规划', in_progress: '开发中',
  shipped: '已上线', declined: '已关闭',
};

const typeLabels: Record<string, string> = {
  bug: 'Bug', feature: '功能需求', improvement: '改进建议',
  question: '问题咨询', other: '其他',
};

export default function FeedbackAnalyticsPage() {
  const [projectId, setProjectId] = useState<number | null>(null);
  const [projects, setProjects] = useState<any[]>([]);
  const [data, setData] = useState<Analytics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiClient.get<any[]>('/v1/feedback/projects').then((res) => {
      if (res.code === 0 && res.data && res.data.length > 0) {
        setProjects(res.data);
        setProjectId(res.data[0].id);
      }
    });
  }, []);

  useEffect(() => {
    if (!projectId) return;
    setLoading(true);
    apiClient.get<any>(`/v1/feedback/projects/${projectId}/analytics`).then((res) => {
      if (res.code === 0) setData(res.data);
    }).finally(() => setLoading(false));
  }, [projectId]);

  const total = data ? Object.values(data.by_status).reduce((a, b) => a + b, 0) : 0;
  const maxTrend = data?.trend?.length ? Math.max(...data.trend.map((t) => t.count), 1) : 1;

  return (
    <div style={{ padding: 24 }}>
      <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
        <Title level={3} style={{ margin: 0 }}>反馈分析</Title>
        {projects.length > 1 && (
          <Space>
            <span>项目：</span>
            <Select value={projectId} onChange={setProjectId}
              options={projects.map((p) => ({ value: p.id, label: p.name }))} style={{ width: 200 }} />
          </Space>
        )}
      </Row>

      {loading ? <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div> : data ? (
        <>
          {/* Summary stats */}
          <Row gutter={16} style={{ marginBottom: 24 }}>
            <Col xs={12} sm={6}><Card><Statistic title="总反馈数" value={total} /></Card></Col>
            <Col xs={12} sm={6}>
              <Card><Statistic title="平均处理时间" value={data.avg_process_hours.toFixed(1)} suffix="小时" /></Card>
            </Col>
            <Col xs={12} sm={6}>
              <Card>
                <Statistic title="最活跃类型" value={
                  Object.entries(data.by_type).sort(([, a], [, b]) => b - a)[0]?.[0] || '-'
                } />
              </Card>
            </Col>
            <Col xs={12} sm={6}>
              <Card>
                <Statistic title="已完成率" value={total > 0 ? ((data.by_status.shipped || 0) / total * 100).toFixed(1) : 0}
                  suffix="%" />
              </Card>
            </Col>
          </Row>

          {/* Trend chart */}
          <Row gutter={16} style={{ marginBottom: 24 }}>
            <Col xs={24} lg={12}>
              <Card title="提交趋势（近12周）" size="small">
                <div style={{ display: 'flex', alignItems: 'flex-end', gap: 4, height: 160, padding: '8px 0' }}>
                  {data.trend.map((t) => (
                    <div key={t.date} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                      <Text style={{ fontSize: 10, marginBottom: 2 }}>{t.count}</Text>
                      <div style={{
                        width: '100%', height: `${(t.count / maxTrend) * 120}px`, minHeight: t.count > 0 ? 4 : 0,
                        background: '#1677ff', borderRadius: '4px 4px 0 0',
                      }} />
                      <Text style={{ fontSize: 9, marginTop: 4, writingMode: 'vertical-lr' }}>
                        {dayjs(t.date).format('MM/DD')}
                      </Text>
                    </div>
                  ))}
                  {data.trend.length === 0 && <div style={{ width: '100%', textAlign: 'center', color: '#999' }}>暂无数据</div>}
                </div>
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card title="分类分布" size="small">
                {Object.entries(data.by_type).map(([key, count]) => {
                  const pct = total > 0 ? (count / total * 100).toFixed(1) : '0';
                  return (
                    <div key={key} style={{ marginBottom: 8 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                        <Text style={{ fontSize: 13 }}>{typeLabels[key] || key}</Text>
                        <Text style={{ fontSize: 13 }}>{count} ({pct}%)</Text>
                      </div>
                      <div style={{ background: '#f0f0f0', borderRadius: 4, height: 8 }}>
                        <div style={{ width: `${Math.min(100, parseFloat(pct) * 2)}%`, background: '#1677ff', height: 8, borderRadius: 4 }} />
                      </div>
                    </div>
                  );
                })}
                {Object.keys(data.by_type).length === 0 && <Text type="secondary">暂无数据</Text>}
              </Card>
            </Col>
          </Row>

          {/* Status + Agent */}
          <Row gutter={16}>
            <Col xs={24} lg={12}>
              <Card title="状态分布" size="small">
                {Object.entries(data.by_status).map(([key, count]) => {
                  const pct = total > 0 ? (count / total * 100).toFixed(1) : '0';
                  return (
                    <div key={key} style={{ marginBottom: 8 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                        <Text style={{ fontSize: 13 }}>{statusLabels[key] || key}</Text>
                        <Text style={{ fontSize: 13 }}>{count}</Text>
                      </div>
                      <div style={{ background: '#f0f0f0', borderRadius: 4, height: 6 }}>
                        <div style={{ width: `${parseFloat(pct)}%`, background: '#52c41a', height: 6, borderRadius: 4 }} />
                      </div>
                    </div>
                  );
                })}
                {Object.keys(data.by_status).length === 0 && <Text type="secondary">暂无数据</Text>}
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card title="Agent 处理量" size="small">
                {Object.entries(data.by_agent).map(([key, count]) => (
                  <div key={key} style={{ marginBottom: 8 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                      <Text style={{ fontSize: 13 }}>Agent #{key}</Text>
                      <Text style={{ fontSize: 13 }}>{count} 条</Text>
                    </div>
                    <div style={{ background: '#f0f0f0', borderRadius: 4, height: 6 }}>
                      <div style={{ width: `${Math.min(100, count / 5 * 100)}%`, background: '#722ed1', height: 6, borderRadius: 4 }} />
                    </div>
                  </div>
                ))}
                {Object.keys(data.by_agent).length === 0 && <Text type="secondary">暂无 Agent 处理数据</Text>}
              </Card>
            </Col>
          </Row>
        </>
      ) : (
        <Card><Text type="secondary">暂无数据</Text></Card>
      )}
    </div>
  );
}
