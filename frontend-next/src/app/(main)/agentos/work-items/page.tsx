'use client';

import { Button, Card, Popconfirm, Space, Spin, Table, Tag, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';
import { fmtDate } from '@/components/crud/CrudListPage';

interface WorkItem {
  id: string | number;
  title?: string;
  description?: string;
  risk_level?: 'high' | 'medium' | 'low' | 'critical' | string;
  status?: 'suggested' | 'approved' | 'executing' | 'executed' | 'rejected' | 'failed' | string;
  agent_id?: string;
  created_at?: string;
}

const RISK_COLORS: Record<string, string> = {
  high: 'red',
  medium: 'orange',
  low: 'green',
  critical: 'red',
};

const STATUS_COLORS: Record<string, string> = {
  suggested: 'blue',
  approved: 'green',
  executing: 'cyan',
  executed: 'green',
  rejected: 'red',
  failed: 'red',
  reviewed: 'default',
};

export default function AgentosWorkItemsPage() {
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['agentos', 'work-items'],
    queryFn: async () => {
      const res = await apiClient.get<WorkItem[]>('/v1/agentos/work-items');
      return res.data ?? [];
    },
    retry: false,
  });

  const act = useMutation({
    mutationFn: async ({ id, action }: { id: string | number; action: 'approve' | 'reject' | 'execute' }) =>
      apiClient.post(`/v1/agentos/work-items/${id}/${action}`),
    onSuccess: (_d, _v, ctx) => {
      message.success('操作成功');
      qc.invalidateQueries({ queryKey: ['agentos', 'work-items'] });
      return ctx;
    },
    onError: (e: Error) => message.error(`操作失败: ${e.message}`),
  });

  const rowBg = (risk: string | undefined) => {
    if (risk === 'high' || risk === 'critical') return '#fff1f0';
    return undefined;
  };

  return (
    <PageContainer title="AgentOS 工作项">
      <Card>
        {isLoading ? (
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin tip="加载中..." />
          </div>
        ) : (
          <Table
            rowKey="id"
            dataSource={data ?? []}
            pagination={{ pageSize: 10 }}
            scroll={{ x: 'max-content' }}
            rowClassName={(r) => {
              const item = r as WorkItem;
              return rowBg(item.risk_level) ? 'row-risk-high' : '';
            }}
            columns={[
              { title: 'ID', dataIndex: 'id', width: 70 },
              { title: '标题', dataIndex: 'title' },
              { title: '描述', dataIndex: 'description' },
              {
                title: '风险',
                dataIndex: 'risk_level',
                width: 100,
                render: (v: string) => (
                  <Tag color={RISK_COLORS[v] ?? 'default'}>{v ?? '-'}</Tag>
                ),
              },
              {
                title: '状态',
                dataIndex: 'status',
                width: 120,
                render: (v: string) => (
                  <Tag color={STATUS_COLORS[v] ?? 'default'}>{v ?? '-'}</Tag>
                ),
              },
              { title: 'Agent', dataIndex: 'agent_id', width: 140 },
              { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
              {
                title: '操作',
                key: '__actions__',
                width: 240,
                fixed: 'right',
                render: (_: unknown, record: WorkItem) => (
                  <Space size="small">
                    <Popconfirm
                      title="确认批准该工作项？"
                      onConfirm={() => act.mutate({ id: record.id, action: 'approve' })}
                    >
                      <Button size="small" type="link">
                        批准
                      </Button>
                    </Popconfirm>
                    <Popconfirm
                      title="确认拒绝该工作项？"
                      onConfirm={() => act.mutate({ id: record.id, action: 'reject' })}
                    >
                      <Button size="small" type="link" danger>
                        拒绝
                      </Button>
                    </Popconfirm>
                    <Popconfirm
                      title="确认执行该工作项？"
                      onConfirm={() => act.mutate({ id: record.id, action: 'execute' })}
                    >
                      <Button size="small" type="link">
                        执行
                      </Button>
                    </Popconfirm>
                  </Space>
                ),
              },
            ]}
          />
        )}
      </Card>

      <style>{`
        .row-risk-high { background: #fff1f0 !important; }
        .row-risk-high:hover > td { background: #ffe7e5 !important; }
      `}</style>
    </PageContainer>
  );
}
