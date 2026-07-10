'use client';

import { useState, useCallback } from 'react';
import { Card, Button, Descriptions, Tag, message, Spin } from 'antd';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useSandboxListingStore } from '../store';
import HighRiskConfirmDialog from '@/components/ui/HighRiskConfirmDialog';
import apiClient from '@/lib/api-client';

export default function StepApproval() {
  const { approvalId, candidateId, listingTaskId, goNext, goBack } = useSandboxListingStore();
  const [showConfirm, setShowConfirm] = useState(false);

  const { data: approval, isLoading } = useQuery({
    queryKey: ['approval', approvalId],
    queryFn: () => apiClient.get(`/v1/approval/${approvalId}`).then(r => r.data as { risk_level?: string }),
    enabled: !!approvalId,
  });

  const { data: candidate } = useQuery({
    queryKey: ['candidate', candidateId],
    queryFn: () => apiClient.get(`/v1/candidates/${candidateId}`).then(r => r.data as { title?: string; target_sale_price?: number }),
    enabled: !!candidateId,
  });

  const approveMutation = useMutation({
    mutationFn: (reviewNote: string) =>
      apiClient.put(`/v1/approval/${approvalId}/review`, {
        action: 'approve',
        review_note: reviewNote,
      }),
    onSuccess: () => {
      message.success('审批通过');
      setShowConfirm(false);
      goNext();
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : '审批失败';
      message.error(msg);
    },
  });

  const handleApprove = useCallback((note: string) => {
    approveMutation.mutate(note);
  }, [approveMutation]);

  if (isLoading) return <Spin />;

  return (
    <div>
      <Card title="审批摘要">
        <Descriptions column={1} size="small">
          <Descriptions.Item label="商品">
            {candidate?.title || `ID: ${candidateId}`}
          </Descriptions.Item>
          <Descriptions.Item label="目标售价">
            ${candidate?.target_sale_price?.toFixed(2)}
          </Descriptions.Item>
          <Descriptions.Item label="风险等级">
            <Tag color="orange">{approval?.risk_level || 'high'}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="执行模式">
            <Tag color="orange">Sandbox</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="不会真实发布">
            ✅ 本操作仅创建沙箱任务，不会发布真实 listing
          </Descriptions.Item>
          <Descriptions.Item label="审计记录">
            approval #{approvalId} 关联 listing task #{listingTaskId}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <div style={{ marginTop: 16, display: 'flex', gap: 8 }}>
        <Button onClick={goBack}>上一步</Button>
        <Button type="primary" onClick={() => setShowConfirm(true)}>
          确认审批沙箱上架任务
        </Button>
      </div>

      <HighRiskConfirmDialog
        open={showConfirm}
        actionName="审批沙箱上架任务"
        riskLevel="high"
        detail={{ targetLabel: candidate?.title || `#${candidateId}` }}
        environmentMode="sandbox"
        showReason
        confirmLoading={approveMutation.isPending}
        onConfirm={(note) => handleApprove(note || '')}
        onCancel={() => setShowConfirm(false)}
      />
    </div>
  );
}
