'use client';

import { Alert, Empty, Space, Tag, Typography } from 'antd';

const { Link, Text } = Typography;

export const DEMAND_DIMENSIONS = [
  { key: 'demand', label: '需求' },
  { key: 'competition', label: '竞争' },
  { key: 'acquisition', label: '获客' },
  { key: 'fulfillment', label: '履约' },
  { key: 'compliance', label: '合规' },
  { key: 'payment', label: '收款' },
  { key: 'aftersales', label: '售后' },
  { key: 'profit_verifiability', label: '利润可验证性' },
] as const;

export type DemandDimensionKey = (typeof DEMAND_DIMENSIONS)[number]['key'];
export type ComparisonEvidenceRole = 'support' | 'counter' | 'conflict';

export interface ComparisonEvidence {
  id: number | string;
  role: ComparisonEvidenceRole;
  truth: string;
  summary: string;
  sourceUri?: string;
  observedAt?: string;
}

export interface ComparisonDimension {
  dimension: DemandDimensionKey;
  evidence: ComparisonEvidence[];
  unknowns?: string[];
}

export interface ComparisonCandidate {
  id: number | string;
  region: string;
  consumer: string;
  needScenario: string;
  salesChannel: string;
  dimensions: ComparisonDimension[];
  strongestCounterevidence?: string;
  unknowns?: string[];
  stopCondition?: string;
}

export interface ComparisonMatrixProps {
  candidates: ComparisonCandidate[];
}

const roleMeta: Record<ComparisonEvidenceRole, { label: string; color: string }> = {
  support: { label: '支持', color: 'blue' },
  counter: { label: '反证', color: 'red' },
  conflict: { label: '冲突', color: 'orange' },
};

function EvidenceCell({ candidate, dimension }: { candidate: ComparisonCandidate; dimension: DemandDimensionKey }) {
  const result = candidate.dimensions.find((item) => item.dimension === dimension);
  const evidence = result?.evidence ?? [];
  const unknowns = result?.unknowns ?? [];

  if (evidence.length === 0 && unknowns.length === 0) {
    return <Text type="warning">unknown：尚无可核验证据</Text>;
  }

  return (
    <Space orientation="vertical" size={10} style={{ width: '100%' }}>
      {evidence.map((item) => {
        const role = roleMeta[item.role];
        return (
          <div key={item.id} style={{ borderBottom: '1px solid #f0f0f0', paddingBottom: 8 }}>
            <Space size={4} wrap>
              <Tag color={role.color}>{role.label}</Tag>
              <Tag>{item.truth || 'unknown'}</Tag>
            </Space>
            <div><Text>{item.summary}</Text></div>
            <div>
              {item.sourceUri ? (
                <Link href={item.sourceUri} target="_blank" rel="noreferrer">原始来源</Link>
              ) : (
                <Text type="warning">来源缺失</Text>
              )}
              <Text type={item.observedAt ? 'secondary' : 'warning'}>
                {' · '}{item.observedAt || '观察时间缺失'}
              </Text>
            </div>
          </div>
        );
      })}
      {unknowns.length > 0 && (
        <div>
          <Text strong>本维度 unknown：</Text>
          <Text type="warning">{unknowns.join('；')}</Text>
        </div>
      )}
    </Space>
  );
}

function CandidateSummary({ candidate }: { candidate: ComparisonCandidate }) {
  return (
    <Space orientation="vertical" size={6} style={{ width: '100%' }}>
      <div><Text strong>最强反证：</Text><Text>{candidate.strongestCounterevidence || 'unknown：尚未识别'}</Text></div>
      <div><Text strong>整体 unknown：</Text><Text type="warning">{candidate.unknowns?.length ? candidate.unknowns.join('；') : '无已登记项（不代表没有未知）'}</Text></div>
      <div><Text strong>停止线：</Text><Text>{candidate.stopCondition || 'unknown：尚未冻结'}</Text></div>
    </Space>
  );
}

export default function ComparisonMatrix({ candidates }: ComparisonMatrixProps) {
  if (candidates.length === 0) {
    return <Empty description="尚无候选市场可比较" />;
  }

  return (
    <section aria-label="候选市场八维比较">
      <Alert
        type="info"
        showIcon
        title="同框比较只帮助 Owner 审议"
        description="支持、反证和冲突证据必须连同真实性、来源和观察时间一起判断；材料齐全不代表市场已经被 Owner 选中。"
        style={{ marginBottom: 16 }}
      />
      <div style={{ overflowX: 'auto' }}>
        <table style={{ borderCollapse: 'collapse', minWidth: Math.max(900, 260 + candidates.length * 360), width: '100%' }}>
          <thead>
            <tr>
              <th scope="col" style={headerStyle}>比较维度</th>
              {candidates.map((candidate) => (
                <th key={candidate.id} scope="col" style={headerStyle}>
                  <Space orientation="vertical" size={0}>
                    <Text strong>{candidate.region} × {candidate.consumer}</Text>
                    <Text type="secondary">{candidate.needScenario}</Text>
                    <Tag>{candidate.salesChannel}</Tag>
                  </Space>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {DEMAND_DIMENSIONS.map(({ key, label }) => (
              <tr key={key}>
                <th scope="row" style={rowHeaderStyle}>{label}</th>
                {candidates.map((candidate) => (
                  <td key={candidate.id} style={cellStyle}><EvidenceCell candidate={candidate} dimension={key} /></td>
                ))}
              </tr>
            ))}
            <tr>
              <th scope="row" style={rowHeaderStyle}>审议边界</th>
              {candidates.map((candidate) => (
                <td key={candidate.id} style={cellStyle}><CandidateSummary candidate={candidate} /></td>
              ))}
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  );
}

const headerStyle = {
  background: '#fafafa',
  border: '1px solid #f0f0f0',
  minWidth: 240,
  padding: 12,
  textAlign: 'left' as const,
  verticalAlign: 'top' as const,
};

const rowHeaderStyle = {
  ...headerStyle,
  minWidth: 150,
  width: 180,
};

const cellStyle = {
  border: '1px solid #f0f0f0',
  minWidth: 340,
  padding: 12,
  verticalAlign: 'top' as const,
};
