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
});
