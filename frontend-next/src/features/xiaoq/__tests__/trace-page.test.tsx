import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const push = vi.fn();
vi.mock('next/navigation', () => ({
  useParams: () => ({ traceId: 'trc-agent-1' }),
  useRouter: () => ({ push }),
}));
vi.mock('@/lib/api-client', () => ({ default: { get: vi.fn() } }));

import XiaoQTracePage from '@/app/(main)/xiaoq/traces/[traceId]/page';
import apiClient from '@/lib/api-client';

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}><XiaoQTracePage /></QueryClientProvider>);
}

describe('XiaoQ owner-scoped trace page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(apiClient.get).mockResolvedValue({
      data: {
        trace: {
          trace_id: 'trc-agent-1', agent_id: 'xiao_q', decision_point: 'demand_case_agent', status: 'completed',
          model_provider: 'openai', model_name: 'test-model', prompt_version: 'xiaoq-agent-runtime-v1', token_count: 42,
          final_output: { answer: '当前仍为证据不足。', truth_status: 'inferred' },
        },
        events: [
          { seq: 1, event_type: 'tool_requested', content: 'demand_case_read', payload: { model_note: '先读取案件。' } },
          { seq: 2, event_type: 'capability_call', content: 'demand_case.read', payload: { status: 'succeeded' } },
        ],
        evidence: [{ id: 1, source_type: 'demand_evidence', source_id: '9', title: '公开来源线索', summary: 'quoted' }],
      },
    } as never);
  });

  it('replays model, public tool reason, final answer and evidence without action controls', async () => {
    renderPage();
    expect(await screen.findByText('当前仍为证据不足。')).toBeInTheDocument();
    expect(screen.getByText('test-model')).toBeInTheDocument();
    expect(screen.getByText('模型请求工具')).toBeInTheDocument();
    expect(screen.getByText(/先读取案件/)).toBeInTheDocument();
    expect(screen.getByText('公开来源线索')).toBeInTheDocument();
    expect(apiClient.get).toHaveBeenCalledWith('/v1/xiao-q/traces/trc-agent-1');
    expect(screen.queryByRole('button', { name: /批准|执行/ })).not.toBeInTheDocument();
  });
});
