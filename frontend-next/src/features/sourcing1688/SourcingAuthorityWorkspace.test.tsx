import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import apiClient from '@/lib/api-client';
import SourcingAuthorityWorkspace, { COMPLIANCE_TYPES, COST_TYPES, complianceBlocker, normalizedMinorHalfUp, skuCombinationFacts } from './SourcingAuthorityWorkspace';

vi.mock('@/lib/api-client', () => ({ default: { get: vi.fn(), post: vi.fn() } }));

function renderWorkspace() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}><SourcingAuthorityWorkspace sourceID={12} taskLinkID={34} snapshotID={56} productID={78} /></QueryClientProvider>);
}

describe('SourcingAuthorityWorkspace', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(apiClient.get).mockImplementation(async (path: string) => {
      if (path.endsWith('/sku-mappings')) return { code: 0, message: 'ok', data: [{ id: 90, product_id: 78, internal_sku_id: 91, supplier_sku: 'SUP-1', internal_sku: 'INT-1', channel_sku: 'CH-1', snapshot_id: 56, version: 2 }] } as never;
      if (path.endsWith('/sku-workspace')) return { code: 0, message: 'ok', data: {
        source_id: 12, task_link_id: 34, snapshot_id: 56, observed_at: '2026-07-12T00:00:00Z',
        target: { sales_channel: 'amazon', target_locale: 'en-US', product_opportunity_id: 7, platform_ids: [3] },
        dimensions: [{ name: '颜色', values: ['白色', '黑色'], source: 'declared' }],
        combinations: [
          { key: '白色', supplier_sku: 'SUP-1', spec: '白色', values: { 颜色: '白色' }, quoted_price: 12.5, stock_status: 'observed', quoted_stock: 8, issues: [], duplicate: false, mapping: { id: 90, product_id: 78, internal_sku_id: 91, supplier_sku: 'SUP-1', internal_sku: 'INT-1', channel_sku: 'CH-1', snapshot_id: 56, version: 2 } },
          { key: '黑色', supplier_sku: '', spec: '黑色', values: { 颜色: '黑色' }, stock_status: 'unknown', issues: ['missing_price'], duplicate: false },
        ],
        duplicate_combinations: [], missing_price: ['黑色'], missing_stock: ['黑色'],
        missing_combinations: { status: 'calculated', combinations: [], reason: '' }, canonical_mappings: [], status: 'needs_attention', blockers: ['1个组合缺少报价'],
      } } as never;
      if (path.endsWith('/cost-versions')) return { code: 0, message: 'ok', data: [] } as never;
      if (path.endsWith('/material-assets')) return { code: 0, message: 'ok', data: { assets: [] } } as never;
      if (path.endsWith('/draft')) return { code: 0, message: 'ok', data: { draft: { approval_status: '' } } } as never;
      return { code: 0, message: 'ok', data: [{ id: 1, requirement_code: 'brand_ip', requirement_text: '品牌授权', evidence_source: 'evidence://brand', truth_status: 'quoted', scope: 'product 78', country_code: 'US', channel_code: 'amazon', observed_at: '2026-07-12T00:00:00Z', review_status: 'pending' }] } as never;
    });
  });

  it('loads exact task mappings, cost versions and compliance evidence', async () => {
    renderWorkspace();
    expect((await screen.findAllByText(/SUP-1 → INT-1 → CH-1/)).length).toBeGreaterThan(0);
    expect(screen.getByText('quoted 不是 actual，不能通过')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Owner批准' })).toBeDisabled();
    expect(screen.getByText(/发布阻塞：6 类证据未通过/)).toBeInTheDocument();
    expect(screen.getByText('颜色：白色 / 黑色（页面声明）')).toBeInTheDocument();
    expect(screen.getByText('¥12.5')).toBeInTheDocument();
    expect(screen.getAllByText('unknown').length).toBeGreaterThan(0);
    expect(screen.getAllByText('SUP-1 → INT-1 → CH-1').length).toBeGreaterThan(0);
    await waitFor(() => expect(apiClient.get).toHaveBeenCalledWith('/v1/sourcing-1688/12/task-links/34/sku-mappings'));
    expect(apiClient.get).toHaveBeenCalledWith('/v1/sourcing-1688/12/task-links/34/sku-workspace');
    expect(apiClient.get).toHaveBeenCalledWith('/v1/sourcing-1688/12/cost-versions');
    expect(apiClient.get).toHaveBeenCalledWith('/v1/sourcing-1688/12/task-links/34/compliance-evidence');
  });

  it('uses exactly ten canonical minor-unit cost types and six compliance types', () => {
    expect(COST_TYPES).toHaveLength(10);
    expect(new Set(COST_TYPES).size).toBe(10);
    expect(COMPLIANCE_TYPES).toEqual(['brand_ip', 'patent', 'certification', 'dangerous_goods', 'material', 'labeling_instructions']);
  });

  it('treats quoted, expired and revoked evidence as blockers', () => {
    const base = { id: 1, requirement_code: 'brand_ip', requirement_text: '', evidence_source: '', truth_status: 'actual', scope: '', country_code: 'US', channel_code: 'amazon', observed_at: '2026-07-01T00:00:00Z', review_status: 'approved' };
    expect(complianceBlocker({ ...base, truth_status: 'quoted' })).toContain('不是 actual');
    expect(complianceBlocker({ ...base, expires_at: '2026-07-02T00:00:00Z' }, new Date('2026-07-03T00:00:00Z'))).toBe('已过期');
    expect(complianceBlocker({ ...base, revoked_at: '2026-07-02T00:00:00Z', revocation_reason: '撤证' })).toBe('已撤销：撤证');
  });

  it('never turns absent SKU price, inventory or mapping into zero-valued facts', () => {
    expect(skuCombinationFacts({ key: 'black', supplier_sku: '', spec: 'black', values: { color: 'black' }, stock_status: 'unknown', issues: ['missing_price'], duplicate: false })).toEqual({
      price: 'unknown', stock: 'unknown', mapping: 'unmapped',
    });
  });

  it('previews decimal exchange rates with integer half-up rounding', () => {
    expect(normalizedMinorHalfUp('101', '7.25')).toBe('732');
    expect(normalizedMinorHalfUp('1', '0.5')).toBe('1');
    expect(normalizedMinorHalfUp('', '7.25')).toBeNull();
  });
});
