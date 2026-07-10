'use client';

import { useState } from 'react';
import { Card, Button, message, Spin, Input, Tag } from 'antd';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useSandboxListingStore } from '../store';
import apiClient from '@/lib/api-client';

export default function StepFieldCompletion() {
  const { candidateId, goNext, goBack } = useSandboxListingStore();
  const queryClient = useQueryClient();

  const { data: candidate, isLoading } = useQuery({
    queryKey: ['candidate', candidateId],
    queryFn: () => apiClient.get(`/v1/candidates/${candidateId}`).then(r => r.data),
    enabled: !!candidateId,
  });

  const { data: completeness, isLoading: completenessLoading } = useQuery({
    queryKey: ['completeness', candidateId],
    queryFn: () => apiClient.post(`/v1/completeness/check/${candidateId}`).then(r => r.data as { missing_items: string[]; score: number }),
    enabled: !!candidateId,
  });

  const completenessData = completeness as { missing_items: string[]; score: number } | undefined;

  const fillMutation = useMutation({
    mutationFn: (fields: Array<{ field: string; value: unknown }>) =>
      apiClient.put(`/v1/candidates/${candidateId}/fields`, { fields }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['candidate', candidateId] });
      queryClient.invalidateQueries({ queryKey: ['completeness', candidateId] });
      message.success('字段已更新');
    },
  });

  const [editValues, setEditValues] = useState<Record<string, string>>({});

  if (isLoading) return <Spin />;

  const missingItems: string[] = completenessData?.missing_items || [];
  const score = completenessData?.score || 0;

  const handleFill = (field: string) => {
    const val = editValues[field];
    if (!val) return;
    fillMutation.mutate([{ field, value: val }]);
  };

  const handleSkip = (field: string) => {
    fillMutation.mutate([{ field, value: '__skipped__' }]);
  };

  return (
    <div>
      <Card size="small" style={{ marginBottom: 16 }}>
        资料完整度: <Tag color={score >= 80 ? 'green' : score >= 50 ? 'orange' : 'red'}>{score}%</Tag>
      </Card>

      {missingItems.length === 0 ? (
        <p>所有字段已完成，可以进入下一步。</p>
      ) : (
        missingItems.map((field) => (
          <Card key={field} size="small" style={{ marginBottom: 8 }}>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <span style={{ minWidth: 160 }}>{field}</span>
              <Input
                size="small"
                style={{ width: 200 }}
                value={editValues[field] || ''}
                onChange={(e) => setEditValues({ ...editValues, [field]: e.target.value })}
              />
              <Button size="small" type="primary" onClick={() => handleFill(field)} disabled={!editValues[field]}>
                填写
              </Button>
              <Button size="small" onClick={() => handleSkip(field)}>暂缺</Button>
            </div>
          </Card>
        ))
      )}

      <div style={{ marginTop: 24, display: 'flex', gap: 8 }}>
        <Button onClick={goBack}>上一步</Button>
        <Button type="primary" onClick={goNext}>继续到利润证据卡</Button>
      </div>
    </div>
  );
}
