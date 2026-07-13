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

    await expect(sendXiaoQMessage({ message: '能继续吗？', demand_case_id: 7 })).resolves.toMatchObject(response);
    expect(apiClient.post).toHaveBeenCalledWith('/v1/xiao-q/messages', {
      message: '能继续吗？', demand_case_id: 7,
    });
  });

  it('posts the explicit experiment target using the current backend contract', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({
      code: 0,
      message: 'ok',
      data: {
        trace_id: 'trace-exp', agent_id: 'xiao_q', target_type: 'experiment',
        experiment_id: 'EXP-2026-001', answer: '实验仍被证据闸门阻断。',
        truth_status: 'inferred', mode: 'read_only_v1', evidence: [],
        unknowns: ['缺少付款证据'], links: [],
      },
    });

    await expect(sendXiaoQMessage({
      message: '为什么还不能进入下一步？',
      target_type: 'experiment',
      experiment_id: 'EXP-2026-001',
    })).resolves.toMatchObject({ target_type: 'experiment', experiment_id: 'EXP-2026-001' });
    expect(apiClient.post).toHaveBeenCalledWith('/v1/xiao-q/messages', {
      message: '为什么还不能进入下一步？',
      target_type: 'experiment',
      experiment_id: 'EXP-2026-001',
    });
  });

  it('posts the controlled sourcing target using source_id', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({
      code: 0,
      message: 'ok',
      data: {
        trace_id: 'trace-source', agent_id: 'xiao_q', target_type: 'sourcing_1688', source_id: 42,
        answer: '只读来源说明', truth_status: 'inferred', mode: 'read_only_v1',
        evidence: [], unknowns: ['费用仍未知'], links: [],
      },
    });

    await expect(sendXiaoQMessage({
      message: '成本还缺什么？', target_type: 'sourcing_1688', source_id: 42,
    })).resolves.toMatchObject({ target_type: 'sourcing_1688', source_id: 42 });
    expect(apiClient.post).toHaveBeenCalledWith('/v1/xiao-q/messages', {
      message: '成本还缺什么？', target_type: 'sourcing_1688', source_id: 42,
    });
  });

  it('normalizes missing collections and experiment backend links', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({
      code: 0,
      message: 'ok',
      data: {
        trace_id: 'trace-exp', agent_id: 'xiao_q', target_type: 'experiment',
        experiment_id: 'EXP-1', answer: '回答', truth_status: 'inferred',
        links: {
          experiment: '/experiments?experiment_id=EXP-1',
          gate_status: '/api/v1/experiments/EXP-1/owner-summary',
          trace: '/api/v1/xiao-q/traces/trace-exp',
        },
      },
    });

    await expect(sendXiaoQMessage({
      message: '问题', target_type: 'experiment', experiment_id: 'EXP-1',
    })).resolves.toMatchObject({
      evidence: [],
      unknowns: [],
      links: [
        { label: '查看经营实验', href: '/experiments?experiment_id=EXP-1' },
        { label: '查看实验闸门状态', href: '/api/v1/experiments/EXP-1/owner-summary' },
        { label: '查看运行追踪', href: '/api/v1/xiao-q/traces/trace-exp' },
      ],
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

  it('preserves order fact blockers and authoritative deep links', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: {
      trace_id: 'order-trace', agent_id: 'xiao_q', target_type: 'operating_facts', order_id: 81,
      answer: '事实说明', truth_status: 'inferred', mode: 'read_only_v1', evidence: [],
      unknowns: ['签收未知'], blockers: ['现金未对账'], links: [{ label: '查看订单', href: '/orders/81' }],
    } });
    await expect(sendXiaoQMessage({ message: '状态？', target_type: 'operating_facts', order_id: 81 }))
      .resolves.toMatchObject({ order_id: 81, blockers: ['现金未对账'] });
  });

  it('posts recommendation creation explicitly without creating an Owner decision', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: {
      trace_id: 'decision-trace', agent_id: 'xiao_q', target_type: 'business_decision', decision_case_id: 9,
      answer: '推断建议', truth_status: 'inferred', mode: 'decision_support_v1', recommendation_id: 17, evidence: [], unknowns: [], links: [],
    } });
    const request = { message: '生成建议', target_type: 'business_decision' as const, decision_case_id: 9, create_recommendation: true, idempotency_key: 'recommend-9-a' };
    await expect(sendXiaoQMessage(request)).resolves.toMatchObject({ truth_status: 'inferred', decision_case_id: 9, recommendation_id: 17 });
    expect(apiClient.post).toHaveBeenCalledWith('/v1/xiao-q/messages', request);
  });
});
