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

  it('shows experiment evidence, gate blockers, unknowns and links without actions', () => {
    render(<XiaoQAnswerCard response={{
      trace_id: 'trace-exp', agent_id: 'xiao_q', target_type: 'experiment',
      experiment_id: 'EXP-1', answer: '实验尚未通过闸门。',
      truth_status: 'inferred', mode: 'read_only_v1',
      evidence: [{ title: '支付记录', summary: 'payment/payment', truth_status: 'actual' }],
      unknowns: ['机会闸门尚未通过', '最终利润尚未确认（pending）'],
      links: [{ label: '经营实验', href: '/experiments?experiment_id=EXP-1' }],
    }} />);

    expect(screen.getByText('经营实验证据')).toBeInTheDocument();
    expect(screen.getByText('支付记录')).toBeInTheDocument();
    expect(screen.getByText('闸门阻断与仍然未知')).toBeInTheDocument();
    expect(screen.getByText('机会闸门尚未通过')).toBeInTheDocument();
    expect(screen.getByText('最终利润尚未确认（pending）')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '经营实验' })).toHaveAttribute('href', '/experiments?experiment_id=EXP-1');
    expect(screen.queryByRole('button', { name: /批准|执行/ })).not.toBeInTheDocument();
  });

  it('shows controlled sourcing limitations, snapshot hash and cost truth status without actions', () => {
    render(<XiaoQAnswerCard response={{
      trace_id: 'trace-source', agent_id: 'xiao_q', target_type: 'sourcing_1688', source_id: 42,
      answer: '当前只能核对受控来源与内部草稿。', truth_status: 'inferred', mode: 'read_only_v1',
      evidence: [
        { id: 7, title: '1688不可变来源快照', summary: 'snapshot', truth_status: 'quoted', snapshot_id: 7, snapshot_sha256: 'abc123hash' },
        { id: 8, title: '采购成本', summary: '12.80 CNY', truth_status: 'estimated', snapshot_id: 7, snapshot_sha256: 'abc123hash' },
      ],
      unknowns: ['供应商资质尚未外部核验', '采购成本（estimated）'],
      links: [{ label: '1688受控货源', href: '/sourcing1688?record_id=42' }],
    }} />);

    expect(screen.getByText('受控来源、快照与成本证据')).toBeInTheDocument();
    expect(screen.getByText('限制与仍然未知')).toBeInTheDocument();
    expect(screen.getByText('供应商资质尚未外部核验')).toBeInTheDocument();
    expect(screen.getAllByText(/abc123hash/)).toHaveLength(2);
    expect(screen.getByText('估算')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '1688受控货源' })).toHaveAttribute('href', '/sourcing1688?record_id=42');
    expect(screen.queryByRole('button', { name: /批准|执行|发布|采购/ })).not.toBeInTheDocument();
  });

  it('shows business closure truth, evidence, unknowns and links without execution actions', () => {
    render(<XiaoQAnswerCard response={{
      trace_id: 'trace-close', agent_id: 'xiao_q', target_type: 'business_closure',
      experiment_id: 'EXP-CLOSE-1', answer: '订单已记录，但最终利润仍未知。',
      truth_status: 'unknown', mode: 'read_only_v1',
      evidence: [
        { title: '订单履约记录', summary: '系统记录为已签收，尚无外部核验', truth_status: 'unknown' },
        { title: '结算对账', summary: '尚未全部匹配', truth_status: 'inferred' },
      ],
      unknowns: ['退货/争议观察期是否结束未知', '最终利润记录缺失'],
      links: [{ label: '经营实验', href: '/experiments?experiment_id=EXP-CLOSE-1' }],
    }} />);

    expect(screen.getByText('订单、结算与最终利润证据')).toBeInTheDocument();
    expect(screen.getAllByText('未知').length).toBeGreaterThan(0);
    expect(screen.getByText('订单履约记录')).toBeInTheDocument();
    expect(screen.getByText('仍未核验的经营事实')).toBeInTheDocument();
    expect(screen.getByText('退货/争议观察期是否结束未知')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '经营实验' })).toHaveAttribute('href', '/experiments?experiment_id=EXP-CLOSE-1');
    expect(screen.queryByRole('button', { name: /批准|执行|发货|退款|收款/ })).not.toBeInTheDocument();
  });

  it('separates operating unknowns from blockers and deep-links to facts', () => {
    render(<XiaoQAnswerCard response={{
      trace_id: 'trace-order', agent_id: 'xiao_q', target_type: 'operating_facts', order_id: 81,
      answer: '订单已摄取，现金尚未对账。', truth_status: 'inferred', mode: 'read_only_v1',
      evidence: [{ title: '平台订单快照', truth_status: 'actual', summary: '来源已冻结' }],
      unknowns: ['真实承运商签收尚未取得'],
      blockers: ['现金到账金额与结算应收不一致'],
      links: [{ label: '查看权威订单', href: '/orders/81' }],
    }} />);

    expect(screen.getByText('订单经营事实')).toBeInTheDocument();
    expect(screen.getByText('仍未核验的经营事实')).toBeInTheDocument();
    expect(screen.getByText('当前阻断原因')).toBeInTheDocument();
    expect(screen.getByText('现金到账金额与结算应收不一致')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '查看权威订单' })).toHaveAttribute('href', '/orders/81');
  });

  it('labels a business recommendation as inferred and never as an Owner decision', () => {
    render(<XiaoQAnswerCard response={{
      trace_id: 'trace-decision', agent_id: 'xiao_q', target_type: 'business_decision', decision_case_id: 9,
      answer: '建议暂停并补充现金凭证。', truth_status: 'inferred', mode: 'suggestion',
      evidence: [{ title: '经营事实快照', truth_status: 'actual', summary: 'manifest已冻结' }],
      unknowns: ['退款观察窗口是否结束'], blockers: [],
      links: [{ label: '查看权威决策案卷', href: '/business-decisions/9' }],
    }} />);

    expect(screen.getByText('AI 建议不是 Owner 决定')).toBeInTheDocument();
    expect(screen.getAllByText('推断').length).toBeGreaterThan(0);
    expect(screen.getByText(/只有你在权威经营决策页面明确提交的决定/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /批准|执行/ })).not.toBeInTheDocument();
  });
});
