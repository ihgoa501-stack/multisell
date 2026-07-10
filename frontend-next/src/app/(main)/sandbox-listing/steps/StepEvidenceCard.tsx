'use client';

import { Spin, Button, Alert } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { useSandboxListingStore } from '../store';
import EvidenceCard, { type EvidenceCardData } from '@/components/profit/EvidenceCard';
import apiClient from '@/lib/api-client';

export default function StepEvidenceCardPage() {
  const { candidateId, goNext, goBack } = useSandboxListingStore();

  const { data, isLoading, error } = useQuery<EvidenceCardData | undefined>({
    queryKey: ['evidence-card', candidateId],
    queryFn: () => apiClient.get(`/v1/profit/evidence-card/${candidateId}`).then(r => r.data as EvidenceCardData | undefined),
    enabled: !!candidateId,
  });

  if (isLoading) return <Spin />;
  if (error) return <Alert type="error" message="加载利润证据卡失败" />;
  if (!data) return null;

  return (
    <div>
      <EvidenceCard data={data} />
      <div style={{ marginTop: 24, display: 'flex', gap: 8 }}>
        <Button onClick={goBack}>上一步</Button>
        <Button type="primary" onClick={goNext}>继续到上架建议</Button>
      </div>
    </div>
  );
}
