import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import ComparisonMatrix, { DEMAND_DIMENSIONS, type ComparisonCandidate } from './ComparisonMatrix';

const candidate: ComparisonCandidate = {
  id: 1,
  region: '美国',
  consumer: '养猫家庭',
  needScenario: '日常清理猫毛',
  salesChannel: 'Amazon US',
  strongestCounterevidence: '普通粘毛器价格更低且更容易购买',
  unknowns: ['真实获客成本', '退货率'],
  stopCondition: '平台费用无法由账单核验时停止',
  dimensions: DEMAND_DIMENSIONS.map(({ key, label }, index) => ({
    dimension: key,
    evidence: index === 0 ? [{
      id: 'demand-support',
      role: 'support',
      truth: 'quoted',
      summary: `${label}公开资料显示存在搜索需求`,
      sourceUri: 'https://example.test/demand',
      observedAt: '2026-07-11T08:00:00Z',
    }, {
      id: 'demand-counter',
      role: 'counter',
      truth: 'inferred',
      summary: '搜索不等于购买',
    }, {
      id: 'demand-conflict',
      role: 'conflict',
      truth: 'quoted',
      summary: '两份公开资料趋势相反',
      sourceUri: 'https://example.test/conflict',
      observedAt: '2026-07-10',
    }] : [{
      id: `${key}-evidence`,
      role: 'support',
      truth: 'estimated',
      summary: `${label}仍需核验`,
      sourceUri: `https://example.test/${key}`,
      observedAt: '2026-07-11',
    }],
    unknowns: index === 2 ? ['渠道广告账户数据缺失'] : [],
  })),
};

describe('ComparisonMatrix', () => {
  it('renders candidates as columns and all eight required dimensions as rows', () => {
    render(<ComparisonMatrix candidates={[candidate, { ...candidate, id: 2, region: '德国', salesChannel: 'Amazon DE' }]} />);

    const matrix = screen.getByRole('region', { name: '候选市场八维比较' });
    expect(within(matrix).getByText('美国 × 养猫家庭')).toBeInTheDocument();
    expect(within(matrix).getByText('德国 × 养猫家庭')).toBeInTheDocument();
    for (const { label } of DEMAND_DIMENSIONS) {
      expect(within(matrix).getByRole('rowheader', { name: label })).toBeInTheDocument();
    }
  });

  it('keeps evidence role, truth, provenance, counterevidence, unknowns and stop line visible', () => {
    render(<ComparisonMatrix candidates={[candidate]} />);

    expect(screen.getAllByText('支持')).not.toHaveLength(0);
    expect(screen.getByText('反证')).toBeInTheDocument();
    expect(screen.getByText('冲突')).toBeInTheDocument();
    expect(screen.getAllByText('quoted')).not.toHaveLength(0);
    expect(screen.getAllByRole('link', { name: '原始来源' })[0]).toHaveAttribute('href', 'https://example.test/demand');
    expect(screen.getByText(/2026-07-11T08:00:00Z/)).toBeInTheDocument();
    expect(screen.getByText(/来源缺失/)).toBeInTheDocument();
    expect(screen.getByText(/观察时间缺失/)).toBeInTheDocument();
    expect(screen.getByText(/普通粘毛器价格更低/)).toBeInTheDocument();
    expect(screen.getByText(/真实获客成本；退货率/)).toBeInTheDocument();
    expect(screen.getByText(/渠道广告账户数据缺失/)).toBeInTheDocument();
    expect(screen.getByText(/平台费用无法由账单核验时停止/)).toBeInTheDocument();
    expect(screen.getByText(/材料齐全不代表市场已经被 Owner 选中/)).toBeInTheDocument();
  });

  it('does not expose downstream external-write actions', () => {
    render(<ComparisonMatrix candidates={[candidate]} />);

    expect(screen.queryByRole('button')).not.toBeInTheDocument();
    expect(screen.queryByText(/采购|发布|投放/)).not.toBeInTheDocument();
  });

  it('shows an explicit unknown state when a dimension has no evidence', () => {
    const incomplete = {
      ...candidate,
      dimensions: candidate.dimensions.filter((item) => item.dimension !== 'compliance'),
    };
    render(<ComparisonMatrix candidates={[incomplete]} />);

    const complianceRow = screen.getByRole('rowheader', { name: '合规' }).closest('tr');
    expect(complianceRow).not.toBeNull();
    expect(within(complianceRow!).getByText('unknown：尚无可核验证据')).toBeInTheDocument();
  });
});
