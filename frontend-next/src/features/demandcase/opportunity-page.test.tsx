import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('next/navigation', () => ({ usePathname: () => '/product-opportunities', useSearchParams: () => new URLSearchParams() }));
vi.mock('@/lib/api-client', () => ({ default: { get: vi.fn(), post: vi.fn() } }));

import ProductOpportunitiesPage from '@/app/(main)/product-opportunities/page';
import apiClient from '@/lib/api-client';

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}><ProductOpportunitiesPage /></QueryClientProvider>);
}

describe('Product opportunity Owner page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(apiClient.get).mockResolvedValue({ code: 0, message: 'ok', data: [{
      id: 1, owner_id: 7, demand_case_id: 2, market_decision_id: 3,
      title: '便携饮水方案', consumer_problem: '出行饮水不便', product_thesis: '防漏饮水器',
      target_channel: '独立站', value_hypothesis: '更易携带', price_hypothesis: '价格待验证',
      source_uri: 'https://example.test/source', truth_status: 'quoted', strongest_counterevidence: '普通水碗更便宜',
      unknowns: ['获客成本'], stop_condition: '费用不可核验时停止', status: 'draft', version: 1,
      content_hash: 'a'.repeat(64), created_at: '', updated_at: '',
    }] });
  });

  it('shows the opportunity as a hypothesis and blocks approval before readiness', async () => {
    renderPage();
    expect(await screen.findByText('便携饮水方案')).toBeInTheDocument();
    expect(screen.getByText('商品机会不是候选商品或 Listing')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Owner 批准' })).toBeDisabled();
    expect(screen.queryByRole('button', { name: /采购|发布|投放/ })).not.toBeInTheDocument();
  });
});
