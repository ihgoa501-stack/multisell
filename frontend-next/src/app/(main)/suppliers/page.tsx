'use client';

import { useState } from 'react';
import { Button, Collapse, Form, Input, InputNumber, Modal, Space, Table, Tag, message } from 'antd';
import { ReloadOutlined, TrophyOutlined, HistoryOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';

interface Supplier {
  id: number;
  name: string;
  contact_person: string;
  contact_phone: string;
  email: string;
  status: number;
  kpi_score: number;
  created_at: string;
}

interface SupplierScore {
  supplier_id: number;
  on_time_delivery_rate: number;
  quality_pass_rate: number;
  communication_score: number;
  order_fulfillment_pct: number;
  avg_lead_time_days: number;
  reliability_score: number;
  data_freshness: number;
}

interface ScoreHistory {
  id: number;
  supplier_id: number;
  reliability_score: number;
  trigger: string;
  created_at: string;
}

function scoreColor(score: number): string {
  if (score >= 80) return '#52c41a';
  if (score >= 60) return '#faad14';
  return '#ff4d4f';
}

export default function SuppliersPage() {
  const qc = useQueryClient();
  const [search, setSearch] = useState('');
  const [scoreModalOpen, setScoreModalOpen] = useState(false);
  const [scoreSupplier, setScoreSupplier] = useState<Supplier | null>(null);
  const [scoreValue, setScoreValue] = useState(0);
  const [historyModalOpen, setHistoryModalOpen] = useState(false);
  const [historySupplierId, setHistorySupplierId] = useState<number | null>(null);

  const { data: listRes, isLoading } = useQuery({
    queryKey: ['suppliers', search],
    queryFn: async () => { const res = await apiClient.getPage<Supplier>('/v1/suppliers', { search, page: '1', size: '50' }); return res; },
  });

  const { data: scoreboardRes } = useQuery({
    queryKey: ['supplier-scoreboard'],
    queryFn: async () => { const res = await apiClient.get<SupplierScore[]>('/v1/suppliers/scoreboard'); return res.data; },
  });

  const suppliers: Supplier[] = listRes?.data ?? [];
  const scoreboard: SupplierScore[] = scoreboardRes ?? [];

  const updateScoreMut = useMutation({
    mutationFn: (params: { id: number; kpi_score: number }) =>
      apiClient.put(`/v1/suppliers/${params.id}/kpi-score`, { kpi_score: params.kpi_score }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['suppliers'] }); qc.invalidateQueries({ queryKey: ['supplier-scoreboard'] }); message.success('评分已更新'); setScoreModalOpen(false); },
  });

  const recalcMut = useMutation({
    mutationFn: (id: number) => apiClient.post(`/v1/suppliers/${id}/recalculate`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['suppliers'] }); qc.invalidateQueries({ queryKey: ['supplier-scoreboard'] }); message.success('评分已重新计算'); },
  });

  const { data: historyRes } = useQuery({
    queryKey: ['supplier-score-history', historySupplierId],
    queryFn: async () => { const id = historySupplierId; if (!id) return []; const res = await apiClient.get<ScoreHistory[]>('/v1/suppliers/'+id+'/score-history'); return res.data; },
    enabled: !!historySupplierId,
  });

  const scoreHistory: ScoreHistory[] = historyRes ?? [];

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '联系人', dataIndex: 'contact_person', key: 'contact_person' },
    { title: '电话', dataIndex: 'contact_phone', key: 'contact_phone' },
    { title: '邮箱', dataIndex: 'email', key: 'email' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (v: number) => <Tag color={v === 1 ? 'green' : 'default'}>{v === 1 ? '启用' : '禁用'}</Tag> },
    { title: 'KPI评分', dataIndex: 'kpi_score', key: 'kpi_score', render: (v: number) => <span style={{ color: scoreColor(v), fontWeight: 600 }}>{v.toFixed(1)}</span> },
    { title: '操作', key: 'actions', render: (_: unknown, r: Supplier) => (
        <Space>
          <Button size="small" icon={<TrophyOutlined />} onClick={() => { setScoreSupplier(r); setScoreValue(r.kpi_score); setScoreModalOpen(true); }}>评分</Button>
          <Button size="small" icon={<ReloadOutlined />} onClick={() => recalcMut.mutate(r.id)} loading={recalcMut.isPending}>重新计算</Button>
          <Button size="small" icon={<HistoryOutlined />} onClick={() => { setHistorySupplierId(r.id); setHistoryModalOpen(true); }}>历史</Button>
        </Space>
    )},
  ];

  const scoreboardColumns = [
    { title: '供应商ID', dataIndex: 'supplier_id', key: 'supplier_id' },
    { title: '准时交付率', dataIndex: 'on_time_delivery_rate', key: 'on_time_delivery_rate', render: (v: number) => `${v.toFixed(1)}%` },
    { title: '质量通过率', dataIndex: 'quality_pass_rate', key: 'quality_pass_rate', render: (v: number) => `${v.toFixed(1)}%` },
    { title: '沟通评分', dataIndex: 'communication_score', key: 'communication_score', render: (v: number) => v.toFixed(1) },
    { title: '订单履约率', dataIndex: 'order_fulfillment_pct', key: 'order_fulfillment_pct', render: (v: number) => `${v.toFixed(1)}%` },
    { title: '平均交期(天)', dataIndex: 'avg_lead_time_days', key: 'avg_lead_time_days', render: (v: number) => v.toFixed(1) },
    { title: '综合可靠性', dataIndex: 'reliability_score', key: 'reliability_score', render: (v: number) => <span style={{ color: scoreColor(v), fontWeight: 600 }}>{v.toFixed(1)}</span> },
  ];

  return (
    <PageContainer title="供应商管理" subtitle="P3: 供应商评分 (#197)">
      <Collapse defaultActiveKey={['suppliers']} items={[
        { key: 'suppliers', label: '供应商列表', children: (
          <>
            <Input.Search placeholder="搜索供应商..." value={search} onChange={e => setSearch(e.target.value)} style={{ width: 300, marginBottom: 16 }} allowClear />
            <Table dataSource={suppliers} columns={columns} rowKey="id" loading={isLoading} pagination={false} size="small" />
          </>
        )},
        { key: 'scoreboard', label: '评分排行榜', children: (
          <Table dataSource={scoreboard} columns={scoreboardColumns} rowKey="supplier_id" pagination={false} size="small" />
        )},
      ]} />

      <Modal title={`手动评分 - ${scoreSupplier?.name}`} open={scoreModalOpen} onOk={() => scoreSupplier && updateScoreMut.mutate({ id: scoreSupplier.id, kpi_score: scoreValue })} onCancel={() => setScoreModalOpen(false)} confirmLoading={updateScoreMut.isPending}>
        <Form layout="vertical">
          <Form.Item label="KPI评分 (0-100)">
            <InputNumber min={0} max={100} value={scoreValue} onChange={v => setScoreValue(v ?? 0)} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="评分历史" open={historyModalOpen} onCancel={() => setHistoryModalOpen(false)} footer={null} width={700}>
        <Table dataSource={scoreHistory} rowKey="id" pagination={false} size="small"
          columns={[
            { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
            { title: '可靠性评分', dataIndex: 'reliability_score', key: 'reliability_score', render: (v: number) => <span style={{ color: scoreColor(v) }}>{v.toFixed(1)}</span> },
            { title: '触发方式', dataIndex: 'trigger', key: 'trigger', render: (v: string) => <Tag>{v === 'auto' ? '自动' : '手动'}</Tag> },
            { title: '时间', dataIndex: 'created_at', key: 'created_at', render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm') },
          ]}
        />
      </Modal>
    </PageContainer>
  );
}
