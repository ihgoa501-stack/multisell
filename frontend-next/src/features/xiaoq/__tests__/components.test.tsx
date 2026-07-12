import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { XiaoQAnswerCard, XiaoQBoundaryBanner } from '../components';

describe('XiaoQ components', () => {
  it('states the read-only boundary and does not offer execution', () => {
    render(<XiaoQBoundaryBanner identity={{ agent_id: 'xiao-q', name: '小Q', mode: 'read_only' }} />);
    expect(screen.getByText('只读模式')).toBeInTheDocument();
    expect(screen.getByText(/不会直接执行发布、采购、价格/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /执行/ })).not.toBeInTheDocument();
  });

  it('keeps unknown evidence visibly distinct and shows sources', () => {
    render(
      <XiaoQAnswerCard response={{
        trace_id: 'trace-1', agent_id: 'xiao-q', answer: '目前不能判断。',
        truth_status: 'unknown', mode: 'read_only',
        evidence: [{ title: '平台费用', truth_status: 'mock', source_url: 'https://example.com/source', run_id: 'run-7', snapshot_id: 'snap-8' }],
        unknowns: ['真实平台费用未知'],
        links: [{ label: '查看候选市场', href: '/demand-cases' }],
        provenance: { source: 'demand-case' },
      }} />,
    );

    expect(screen.getByText('未知')).toBeInTheDocument();
    expect(screen.getByText('模拟')).toBeInTheDocument();
    expect(screen.getByText('真实平台费用未知')).toBeInTheDocument();
    expect(screen.getByText(/run-7/)).toBeInTheDocument();
    expect(screen.getByText(/snap-8/)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '查看来源' })).toHaveAttribute('href', 'https://example.com/source');
    expect(screen.getByRole('link', { name: '查看候选市场' })).toHaveAttribute('href', '/demand-cases');
    expect(screen.queryByRole('button', { name: /批准|执行/ })).not.toBeInTheDocument();
  });

  it('shows capability permission, status and approval policy', async () => {
    const { XiaoQCapabilities } = await import('../components');
    render(<XiaoQCapabilities capabilities={[{
      code: 'demand_case.read', name: '读取案件', mode: 'read_only', available: true,
      required_permission: 'demand_case.read', status: 'active', approval_required: false,
    }]} />);
    expect(screen.getByText(/demand_case\.read/)).toBeInTheDocument();
    expect(screen.getByText('active')).toBeInTheDocument();
    expect(screen.getByText('无需审批')).toBeInTheDocument();
  });
});
