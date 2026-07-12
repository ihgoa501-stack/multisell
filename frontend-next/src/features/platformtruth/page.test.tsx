import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('next/navigation', () => ({ usePathname: () => '/platform-truth' }));
vi.mock('./api', () => ({ getPlatformTruth: vi.fn() }));

import PlatformTruthPage from '@/app/(main)/platform-truth/page';
import { getPlatformTruth } from './api';

const contract = {
  version: '2026-07-12', direction: '只供 Owner 本人使用的完整平台',
  truth_levels: [{ code: 'actual', meaning: '直接核验事实', can_be_direct: false }],
  claim_levels: [{ code: 'implemented', meaning: '代码存在' }],
  system_boundaries: [
    { code: 'fact', name: '经营事实系统', responsibility: '保存事实', must_not: '冒充因果' },
    { code: 'decision', name: '经营决策系统', responsibility: '保存决策', must_not: '替代事实' },
  ],
  object_identity_rules: [{ code: 'stable_id', rule: '对象必须有稳定 ID' }],
  source_rules: [{ code: 'provenance', rule: '事实必须保留来源' }],
  domain_dispositions: [
    { id: 'order', name: '订单', system: 'fact', disposition: 'reuse', reason: '交易事实', evidence: 'implemented', xiao_q_support: 'active', owner_scope: 'single_owner', risk: 'high' },
    { id: 'decision', name: '经营决策', system: 'decision', disposition: 'rebuild', reason: '补齐反馈', evidence: 'planned', xiao_q_support: 'deferred', owner_scope: 'single_owner', risk: 'high' },
  ],
  boundary_rules: ['关联不等于因果'], unknowns: ['真实经营结果未知'],
};

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}><PlatformTruthPage /></QueryClientProvider>);
}

describe('Platform truth Owner page', () => {
  beforeEach(() => vi.mocked(getPlatformTruth).mockResolvedValue(contract));

  it('shows direction, boundaries, classifications and unknowns without mutation actions', async () => {
    renderPage();
    expect(await screen.findByText('只供 Owner 本人使用的完整平台')).toBeInTheDocument();
    expect(screen.getByText('经营事实系统')).toBeInTheDocument();
    expect(screen.getByText('订单')).toBeInTheDocument();
    expect(screen.getByText('真实经营结果未知')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /批准|执行|删除|修改/ })).not.toBeInTheDocument();
  });

  it('filters the domain registry for Owner inspection', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('订单');
    await user.type(screen.getByLabelText('搜索领域'), '经营决策');
    expect(screen.getAllByText('经营决策').length).toBeGreaterThan(0);
    expect(screen.queryByText('订单')).not.toBeInTheDocument();
  });
});
