'use client';

import { useState } from 'react';
import { Card, Button, Tag, message, Result } from 'antd';
import { useMutation } from '@tanstack/react-query';
import { useSandboxListingStore } from '../store';
import apiClient from '@/lib/api-client';

const decisionConfig: Record<string, { color: string; label: string }> = {
  list: { color: 'green', label: '建议上架' },
  cautious: { color: 'orange', label: '建议谨慎' },
  skip: { color: 'red', label: '不建议上架' },
  blocked: { color: 'default', label: '数据不足无法判断' },
};

export default function StepRecommendation() {
  const { candidateId, setListingTaskId, setApprovalId, goNext, goBack } = useSandboxListingStore();
  const [evaluated, setEvaluated] = useState(false);

  const evalMutation = useMutation({
    mutationFn: () => apiClient.post(`/v1/loop/evaluate/${candidateId}`),
    onSuccess: (res) => {
      const data = res.data as {
        decision: string; reason: string; confidence: number;
        completeness_score: number; profit_margin: number;
        risk_flags?: string[]; listing_task_id?: number; approval_id?: number;
      };
      setResult(data);
      if (data.listing_task_id) setListingTaskId(data.listing_task_id);
      if (data.approval_id) setApprovalId(data.approval_id);
      setEvaluated(true);
      message.success('上架建议已生成');
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : '评估失败';
      message.error(msg);
    },
  });

  const [result, setResult] = useState<{
    decision: string; reason: string; confidence: number;
    completeness_score: number; profit_margin: number;
    risk_flags?: string[]; listing_task_id?: number; approval_id?: number;
  } | null>(null);

  if (!evaluated) {
    return (
      <div>
        <Card>
          <p>点击下方按钮，系统将：</p>
          <ul>
            <li>检查商品资料完整度</li>
            <li>基于利润证据卡评估商品可行性</li>
            <li>如果评估通过，创建一个<strong>待审批的沙箱上架任务</strong></li>
          </ul>
          <p style={{ color: '#888', fontSize: 13 }}>此操作不会真实发布任何商品。</p>
        </Card>
        <div style={{ marginTop: 16, display: 'flex', gap: 8 }}>
          <Button onClick={goBack}>上一步</Button>
          <Button type="primary" loading={evalMutation.isPending}
            onClick={() => evalMutation.mutate()}>
            生成建议并创建沙箱审批任务
          </Button>
        </div>
      </div>
    );
  }

  const dc = decisionConfig[result?.decision || 'blocked'];

  return (
    <div>
      <Result
        status={result?.decision === 'list' ? 'success' : 'warning'}
        title={<><Tag color={dc.color}>{dc.label}</Tag> 信心值: {((result?.confidence ?? 0) * 100).toFixed(0)}%</>}
        subTitle={result?.reason}
      />

      {result?.risk_flags && result.risk_flags.length > 0 && (
        <Card size="small" title="风险标记" style={{ marginBottom: 16 }}>
          {result.risk_flags.map((f, i) => <Tag key={i} color="red">{f}</Tag>)}
        </Card>
      )}

      <Card size="small" style={{ marginBottom: 16 }}>
        <p>完整度评分: {result?.completeness_score?.toFixed(0)}% | 利润率: {result?.profit_margin?.toFixed(2)}%</p>
      </Card>

      <div style={{ display: 'flex', gap: 8 }}>
        <Button onClick={goBack}>返回修改</Button>
        {result?.decision === 'list' && (
          <Button type="primary" onClick={goNext}>提交审批</Button>
        )}
        {result?.decision === 'cautious' && (
          <>
            <Button onClick={() => setEvaluated(false)}>返回补齐字段</Button>
            <Button type="primary" onClick={goNext}>仍然提交审批</Button>
          </>
        )}
      </div>
    </div>
  );
}
