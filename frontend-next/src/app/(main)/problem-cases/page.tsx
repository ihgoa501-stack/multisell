'use client';

import { Alert, Button, Card, Space, Table, Tag, Typography, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import type { ProblemCase, ReviewedProblemBatchOutcome } from '@/types/problem-case';

const { Text } = Typography;
const meta: Record<string, { label: string; color: string }> = {
  lead: { label: '问题线索', color: 'blue' },
  evidence_missing: { label: '证据不足', color: 'orange' },
  survives_falsification: { label: '反证后存活', color: 'green' },
  rejected: { label: '已淘汰', color: 'red' },
};

function useImportBatch(path: string, successText: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => (await apiClient.post<ReviewedProblemBatchOutcome>(path, {})).data,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['problem-cases'] });
      message.success(successText);
    },
    onError: (error: Error) => message.error(error.message),
  });
}

export default function ProblemCasesPage() {
  const router = useRouter();
  const query = useQuery({
    queryKey: ['problem-cases'],
    queryFn: async () => (await apiClient.get<ProblemCase[]>('/v1/problem-cases')).data ?? [],
  });
  const broadBatch = useImportBatch('/v1/problem-cases/research/reviewed-problem-batch', '已导入首轮问题研究：2 个证据不足，2 个淘汰');
  const hoopaBatch = useImportBatch('/v1/problem-cases/research/reviewed-wildfire-event-batch', '已导入 Hoopa 事件审阅结果：淘汰，不进入商品或渠道');

  return <PageContainer title="具体问题研究" subtitle="先判断问题、责任主体和消费品可解性；问题存活后才选择销售渠道。" loading={query.isLoading} error={query.isError} errorMsg={(query.error as Error | undefined)?.message} onRetry={() => void query.refetch()}>
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Alert type="info" showIcon title="这里不是商品或平台列表" description="房东、雇主、医疗或公共服务负主要责任的问题会被淘汰；没有独立反证不能进入渠道比较。" />
      <Space wrap>
        <Button loading={broadBatch.isPending} onClick={() => broadBatch.mutate()}>导入首轮已审阅问题研究</Button>
        <Button type="primary" loading={hoopaBatch.isPending} onClick={() => hoopaBatch.mutate()}>导入 Hoopa 2021 事件审阅结果</Button>
      </Space>
      <Card styles={{ body: { padding: 0 } }}>
        <Table rowKey="id" pagination={false} dataSource={query.data ?? []} onRow={(row) => ({ onClick: () => router.push(`/problem-cases/${row.id}`), style: { cursor: 'pointer' } })} columns={[
          { title: '地区与人群', render: (_, row) => <><Text strong>{row.region}</Text><br /><Text type="secondary">{row.observable_population}</Text></> },
          { title: '具体问题', dataIndex: 'problem_scenario' },
          { title: '责任主体', dataIndex: 'responsibility' },
          { title: '重复残余障碍', dataIndex: 'residual_barrier_status' },
          { title: '裁决', dataIndex: 'status', render: (value: string) => { const item = meta[value] ?? { label: value, color: 'default' }; return <Tag color={item.color}>{item.label}</Tag>; } },
        ]} />
      </Card>
    </Space>
  </PageContainer>;
}
