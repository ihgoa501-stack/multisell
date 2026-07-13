import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('next/navigation', () => ({ usePathname: () => '/xiaoq' }));

vi.mock('../api', () => ({
  getXiaoQIdentity: vi.fn(),
  getXiaoQCapabilities: vi.fn(),
  sendXiaoQMessage: vi.fn(),
}));

import XiaoQPage from '@/app/(main)/xiaoq/page';
import { getXiaoQCapabilities, getXiaoQIdentity, sendXiaoQMessage } from '../api';

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}><XiaoQPage /></QueryClientProvider>);
}

describe('XiaoQ page target routing', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getXiaoQIdentity).mockResolvedValue({ agent_id: 'xiao_q', name: '小Q', mode: 'read_only_v1' });
    vi.mocked(getXiaoQCapabilities).mockResolvedValue([]);
    vi.mocked(sendXiaoQMessage).mockResolvedValue({
      trace_id: 'trace-exp', agent_id: 'xiao_q', target_type: 'experiment', experiment_id: 'EXP-1',
      answer: '尚未通过闸门', truth_status: 'inferred', mode: 'read_only_v1', evidence: [], unknowns: [], links: [],
    });
  });

  it('keeps candidate market as the default target', () => {
    renderPage();
    expect(screen.getByLabelText('候选市场案件 ID')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '这个案件还缺什么关键证据？' })).toBeDisabled();
    expect(screen.queryByLabelText('经营实验 ID')).not.toBeInTheDocument();
  });

  it('switches to experiment prompts and submits the current backend target contract', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByLabelText('查询对象'));
    await user.click(await screen.findByText('经营实验'));
    await user.type(screen.getByLabelText('经营实验 ID'), 'EXP-1');
    await user.click(screen.getByRole('button', { name: '哪些闸门正在阻断实验？' }));

    await waitFor(() => expect(sendXiaoQMessage).toHaveBeenCalledWith({
      message: '哪些闸门正在阻断实验？',
      target_type: 'experiment',
      experiment_id: 'EXP-1',
    }, expect.anything()));
    expect(screen.queryByRole('button', { name: /批准|执行/ })).not.toBeInTheDocument();
  });

  it('switches to the controlled 1688 draft and submits source_id without actions', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByLabelText('查询对象'));
    await user.click(await screen.findByText('1688受控草稿'));
    await user.type(screen.getByLabelText('1688 来源 ID'), '42');
    await user.click(screen.getByRole('button', { name: '这条来源的快照是否完整？' }));

    await waitFor(() => expect(sendXiaoQMessage).toHaveBeenCalledWith({
      message: '这条来源的快照是否完整？',
      target_type: 'sourcing_1688',
      source_id: 42,
    }, expect.anything()));
    expect(screen.queryByRole('button', { name: /批准|执行|发布|采购/ })).not.toBeInTheDocument();
  });

  it('does not expose the legacy experiment-based business closure', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByLabelText('查询对象'));
    expect(screen.queryByText('订单与最终利润')).not.toBeInTheDocument();
    expect(screen.getByText('订单经营事实')).toBeInTheDocument();
  });

  it('queries the authoritative Unit 4-5 fact chain by order id', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByLabelText('查询对象'));
    await user.click(await screen.findByText('订单经营事实'));
    await user.type(screen.getByLabelText('订单 ID'), '81');
    await user.click(screen.getByRole('button', { name: '库存、履约和售后有哪些阻断？' }));

    await waitFor(() => expect(sendXiaoQMessage).toHaveBeenCalledWith({
      message: '库存、履约和售后有哪些阻断？',
      target_type: 'operating_facts',
      order_id: 81,
    }, expect.anything()));
    expect(screen.getByText(/不会发货、退款、调库存或收款/)).toBeInTheDocument();
  });

  it('keeps an AI recommendation separate from an Owner decision', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByLabelText('查询对象'));
    await user.click(await screen.findByText('经营决策案卷'));
    await user.type(screen.getByLabelText('经营决策案卷 ID'), '9');
    await user.click(screen.getByRole('button', { name: '根据现有事实生成一条新建议' }));

    await waitFor(() => expect(sendXiaoQMessage).toHaveBeenCalledWith(expect.objectContaining({
      message: '根据现有事实生成一条新建议',
      target_type: 'business_decision',
      decision_case_id: 9,
      create_recommendation: true,
      idempotency_key: expect.stringMatching(/^xiao-q-recommendation-9-/),
    }), expect.anything()));
    expect(screen.getByText(/不会生成、批准或执行 Owner 决定/)).toBeInTheDocument();
  });
});
