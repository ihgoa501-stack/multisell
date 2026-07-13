import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';

vi.mock('next/navigation', () => ({ useParams: () => ({ id: '9' }) }));
vi.mock('@/lib/api-client', () => ({ default: { get: vi.fn(), postApproved: vi.fn() } }));

import apiClient from '@/lib/api-client';
import BusinessDecisionDetailPage from '@/app/(main)/business-decisions/[id]/page';

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}><BusinessDecisionDetailPage /></QueryClientProvider>);
}

const detail = {
  case: { id: 9, question: '是否继续补货？', target: '控制缺货风险', object_type: 'platform_order_ingest', object_id: 41, truth_status: 'external_observed', unknowns: ['下一周需求未知'], manifest_sha256: 'a'.repeat(64), created_at: '2026-07-12T00:00:00Z' },
  fact_snapshot: { source_table: 'platform_order_ingest', source_observed_at: '2026-07-12T00:00:00Z', payload_sha256: 'b'.repeat(64), truth_status: 'external_observed' },
  ai_recommendations: [{ id: 3, recommendation: '先暂停并补证', rationale: '需求仍未知', truth_status: 'inferred', unknowns: ['下一周需求未知'], manifest_sha256: 'a'.repeat(64), created_at: '2026-07-12T00:01:00Z' }],
  owner_decisions: [],
};
const pageSource = readFileSync(`${process.cwd()}/src/app/(main)/business-decisions/[id]/page.tsx`, 'utf8');

describe('business decision authoritative page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(apiClient.get).mockResolvedValue({ code: 0, message: 'ok', data: detail });
    vi.mocked(apiClient.postApproved).mockResolvedValue({ code: 0, message: 'ok', data: { id: 5 } });
  });

  it('keeps inferred recommendations visibly separate from Owner decisions', async () => {
    renderPage();
    expect(await screen.findByText('是否继续补货？')).toBeInTheDocument();
    expect(screen.getByText('AI 建议（不是决定）')).toBeInTheDocument();
    expect(screen.getByText('先暂停并补证')).toBeInTheDocument();
    expect(screen.getByText('尚无 Owner 决定')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /选择执行/ })).not.toBeInTheDocument();
  });

  it('saves only a deliberate non-executing Owner decision with the frozen manifest', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('是否继续补货？');
    await user.click(screen.getByRole('button', { name: '记录决定' }));
    await user.type(screen.getByLabelText('你的理由'), '先补充一周真实需求数据');
    await user.type(screen.getByLabelText('已批准的一次性审批 ID'), '77');
    await user.click(screen.getByRole('button', { name: '不可变保存 Owner 决定' }));

    await waitFor(() => expect(apiClient.postApproved).toHaveBeenCalledWith('/v1/business-decisions/9/owner-decisions', expect.objectContaining({
      decision: 'request_more_evidence', reason: '先补充一周真实需求数据', manifest_sha256: 'a'.repeat(64),
      idempotency_key: expect.stringMatching(/^owner-business-decision-9-/),
    }), expect.objectContaining({ approvalId: 77, idempotencyKey: expect.stringMatching(/^owner-business-decision-9-/) })));
  });

  it('lets the Owner recover an interrupted action without redispatching it', () => {
    expect(pageSource).toContain("a.status==='executing'?'恢复为待对账':'显式执行'");
    expect(pageSource).toContain("a.status!=='approved_pending_execution'&&a.status!=='executing'");
  });
});
