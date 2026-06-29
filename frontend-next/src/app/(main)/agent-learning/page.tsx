'use client';

import { Card, Col, Row, Spin, Statistic, Tag, Typography, Space, Alert } from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined, MinusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import type { Result } from '@/types/api';

const { Title, Text } = Typography;

interface AgentAccuracyItem {
  agent_id: string;
  period: string;
  total_decisions: number;
  correct_decisions: number;
  accuracy_pct: number;
  trend: string;
}

const AGENT_LABELS: Record<string, string> = {
  A2: 'Listing Optimizer',
  A3: 'Price Watch',
  A8: 'Product Scout',
  G3: 'Discount Risk',
};

const AGENT_DESCRIPTIONS: Record<string, string> = {
  A2: 'Listing optimization decisions accuracy',
  A3: 'Pricing and profit prediction accuracy',
  A8: 'Sourcing demand prediction accuracy',
  G3: 'Compliance and risk assessment accuracy',
};

export default function AgentLearningPage() {
  const { data, isLoading, isError, error } = useQuery<Result<AgentAccuracyItem[]>>({
    queryKey: ['agent-learning-accuracy'],
    queryFn: () => apiClient.get<AgentAccuracyItem[]>('/v1/agent-learning/accuracy'),
    refetchInterval: 60_000,
  });

  const records = data?.data || [];

  // Group by agent_id
  const grouped: Record<string, AgentAccuracyItem[]> = {};
  for (const rec of records) {
    if (!grouped[rec.agent_id]) grouped[rec.agent_id] = [];
    grouped[rec.agent_id].push(rec);
  }

  // Get the 30d period record for each agent (preferred), or the first available
  const getPrimaryRecord = (items: AgentAccuracyItem[]): AgentAccuracyItem | undefined => {
    return items.find((r) => r.period === '30d') || items[0];
  };

  const getAccuracyColor = (pct: number): string => {
    if (pct >= 80) return '#52c41a';
    if (pct >= 60) return '#faad14';
    return '#ff4d4f';
  };

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'improving':
        return <ArrowUpOutlined style={{ color: '#52c41a', fontSize: 18 }} />;
      case 'declining':
        return <ArrowDownOutlined style={{ color: '#ff4d4f', fontSize: 18 }} />;
      default:
        return <MinusOutlined style={{ color: '#8c8c8c', fontSize: 18 }} />;
    }
  };

  const getTrendLabel = (trend: string): string => {
    switch (trend) {
      case 'improving': return 'Improving';
      case 'declining': return 'Declining';
      default: return 'Stable';
    }
  };

  const formatAccuracy = (pct: number): string => pct.toFixed(1) + '%';

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ marginBottom: 16, justifyContent: 'space-between', width: '100%' }}>
        <div>
          <Title level={3} style={{ margin: 0 }}>Agent Learning Loop</Title>
          <Text type="secondary">
            Decision evaluation accuracy for tracked agents over 30-day periods
          </Text>
        </div>
      </Space>

      {isLoading && (
        <div style={{ textAlign: 'center', padding: 60 }}>
          <Spin size="large" />
        </div>
      )}

      {isError && (
        <Alert
          message="Failed to load accuracy data"
          description={(error as Error)?.message || 'An error occurred'}
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      {!isLoading && !isError && records.length === 0 && (
        <Alert
          message="No data yet"
          description="Accuracy data will appear here once agents have made decisions and evaluations have been completed."
          type="info"
          showIcon
        />
      )}

      {!isLoading && !isError && records.length > 0 && (
        <Row gutter={[16, 16]}>
          {Object.entries(grouped).map(([agentId, items]) => {
            const primary = getPrimaryRecord(items);
            if (!primary) return null;

            const accuracyColor = getAccuracyColor(primary.accuracy_pct);
            const historyItems = items.filter((r) => r.period !== '30d');

            return (
              <Col xs={24} sm={12} lg={8} key={agentId}>
                <Card
                  hoverable
                  style={{ borderRadius: 8 }}
                  actions={
                    historyItems.length > 0
                      ? historyItems.map((item) => (
                          <div key={item.period}>
                            <Text type="secondary" style={{ fontSize: 11 }}>
                              {item.period}
                            </Text>
                            <br />
                            <Text
                              strong
                              style={{ color: getAccuracyColor(item.accuracy_pct) }}
                            >
                              {formatAccuracy(item.accuracy_pct)}
                            </Text>
                          </div>
                        ))
                      : undefined
                  }
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <div>
                      <Title level={4} style={{ margin: 0 }}>
                        {agentId}
                      </Title>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {AGENT_LABELS[agentId] || agentId}
                      </Text>
                      <br />
                      <Text type="secondary" style={{ fontSize: 11 }}>
                        {AGENT_DESCRIPTIONS[agentId] || ''}
                      </Text>
                    </div>
                    {getTrendIcon(primary.trend)}
                  </div>

                  <div style={{ marginTop: 16, textAlign: 'center' }}>
                    <Statistic
                      title="Accuracy"
                      value={formatAccuracy(primary.accuracy_pct)}
                      valueStyle={{ color: accuracyColor, fontSize: 32, fontWeight: 700 }}
                      suffix={
                        <Tag color={primary.trend === 'improving' ? 'green' : primary.trend === 'declining' ? 'red' : 'default'}>
                          {getTrendLabel(primary.trend)}
                        </Tag>
                      }
                    />
                  </div>

                  <div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-around' }}>
                    <div style={{ textAlign: 'center' }}>
                      <Text type="secondary" style={{ fontSize: 11 }}>Total Decisions</Text>
                      <br />
                      <Text strong style={{ fontSize: 18 }}>
                        {primary.total_decisions}
                      </Text>
                    </div>
                    <div style={{ textAlign: 'center' }}>
                      <Text type="secondary" style={{ fontSize: 11 }}>Correct</Text>
                      <br />
                      <Text strong style={{ fontSize: 18, color: '#52c41a' }}>
                        {primary.correct_decisions}
                      </Text>
                    </div>
                  </div>
                </Card>
              </Col>
            );
          })}
        </Row>
      )}
    </div>
  );
}
