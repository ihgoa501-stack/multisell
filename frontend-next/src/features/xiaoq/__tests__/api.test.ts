import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/lib/api-client', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

import apiClient from '@/lib/api-client';
import { getXiaoQCapabilities, getXiaoQIdentity, sendXiaoQMessage } from '../api';

describe('xiao-q api', () => {
  beforeEach(() => vi.clearAllMocks());

  it('loads identity from the canonical endpoint', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      code: 0,
      message: 'ok',
      data: { agent_id: 'xiao-q', name: '小Q', mode: 'read_only' },
    });

    await expect(getXiaoQIdentity()).resolves.toMatchObject({ name: '小Q' });
    expect(apiClient.get).toHaveBeenCalledWith('/v1/xiao-q/identity');
  });

  it('returns an empty capability list when data is absent', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ code: 0, message: 'ok' });
    await expect(getXiaoQCapabilities()).resolves.toEqual([]);
  });

  it('normalizes the current read-only capability contract', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      code: 0, message: 'ok',
      data: [{ id: 'demand_case.read', description: '读取案件', risk: 'read', side_effect: false }],
    });
    await expect(getXiaoQCapabilities()).resolves.toEqual([expect.objectContaining({
      code: 'demand_case.read', mode: 'read_only', available: true,
    })]);
  });

  it('posts only the supported message context', async () => {
    const response = {
      trace_id: 'trace-1', agent_id: 'xiao-q', answer: '证据不足',
      truth_status: 'unknown' as const, mode: 'read_only' as const,
      evidence: [], unknowns: ['缺少来源'], links: [],
    };
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: response });

    await expect(sendXiaoQMessage({ message: '能继续吗？', demand_case_id: 7 })).resolves.toEqual(response);
    expect(apiClient.post).toHaveBeenCalledWith('/v1/xiao-q/messages', {
      message: '能继续吗？', demand_case_id: 7,
    });
  });

  it('normalizes the real backend links object into canonical link items', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({
      code: 0,
      message: 'ok',
      data: {
        trace_id: 'trace-2', agent_id: 'xiao_q', answer: '回答', mode: 'read_only',
        truth_status: 'inferred', evidence: [], unknowns: [],
        links: {
          demand_case: '/demand-cases/9',
          decision_card: '/demand-cases/9#decision-card',
          trace: '/ai/traces/trace-2',
        },
        provenance: { provider: 'openai', model: 'model' },
      },
    });

    const response = await sendXiaoQMessage({ message: '问题', demand_case_id: 9 });
    expect(response.links).toEqual([
      { label: '查看候选市场案件', href: '/demand-cases/9' },
      { label: '查看决策卡', href: '/demand-cases/9#decision-card' },
      { label: '查看运行追踪', href: '/ai/traces/trace-2' },
    ]);
  });

  it('keeps canonical link arrays unchanged', async () => {
    const links = [{ label: '查看案件', href: '/demand-cases/3' }];
    vi.mocked(apiClient.post).mockResolvedValue({
      code: 0, message: 'ok',
      data: { trace_id: 't', agent_id: 'xiao_q', answer: '回答', truth_status: 'actual', mode: 'read_only', evidence: [], unknowns: [], links },
    });
    await expect(sendXiaoQMessage({ message: '问题', demand_case_id: 3 })).resolves.toMatchObject({ links });
  });

  it('rejects an empty message response', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok' });
    await expect(sendXiaoQMessage({ message: '问题', demand_case_id: 1 })).rejects.toThrow('小Q没有返回回答');
  });
});
