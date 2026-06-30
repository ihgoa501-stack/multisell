'use client';

import { useMemo, useState } from 'react';
import { Button, Descriptions, Form, Input, Modal, Space, Table, message } from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  ReloadOutlined as RecalcOutlined,
  EditOutlined,
  DeleteOutlined,
  BarChartOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import FilterBar from '@/components/ui/FilterBar';
import ConfirmDialog from '@/components/ui/ConfirmDialog';

// ── Types ──────────────────────────────────────────────────────────

interface Supplier {
  id: number;
  name: string;
  contact_person: string;
  contact_phone: string;
  email: string;
  address: string;
  status: number;
  remark: string;
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
  last_order_date: string | null;
}

// ── Color helpers ──────────────────────────────────────────────────

function scoreColor(score: number): string {
  if (score >= 80) return '#52c41a';
  if (score >= 60) return '#faad14';
  if (score >= 40) return '#fa8c16';
  return '#f5222d';
}

function fmtDate(v: unknown): string {
  if (!v) return '-';
  const s = String(v);
  const d = dayjs(s);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm') : s;
}

// ── Score Detail Modal ─────────────────────────────────────────────

function ScoreDetailModal({
  score,
  supplierName,
  onClose,
}: {
  score: SupplierScore | null;
  supplierName: string;
  onClose: () => void;
}) {
  return (
    <Modal
      title={`${supplierName} - 信用评分`}
      open={!!score}
      onCancel={onClose}
      footer={<Button onClick={onClose}>关闭</Button>}
      width={560}
    >
      {score && (
        <Descriptions column={2} size="small" bordered>
          <Descriptions.Item label="综合可靠度" span={2}>
            <span style={{ color: scoreColor(score.reliability_score), fontWeight: 700, fontSize: 18 }}>
              {score.reliability_score.toFixed(1)}
            </span>
          </Descriptions.Item>
          <Descriptions.Item label="准时交付率">
            {score.on_time_delivery_rate.toFixed(1)}%
          </Descriptions.Item>
          <Descriptions.Item label="质量合格率">
            {score.quality_pass_rate.toFixed(1)}%
          </Descriptions.Item>
          <Descriptions.Item label="沟通评分">
            {score.communication_score.toFixed(1)}
          </Descriptions.Item>
          <Descriptions.Item label="订单履行率">
            {score.order_fulfillment_pct.toFixed(1)}%
          </Descriptions.Item>
          <Descriptions.Item label="平均交货天数" span={2}>
            {score.avg_lead_time_days.toFixed(1)} 天
          </Descriptions.Item>
          <Descriptions.Item label="最近订单日期" span={2}>
            {score.last_order_date ? dayjs(score.last_order_date).format('YYYY-MM-DD') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="数据新鲜度" span={2}>
            {score.data_freshness} 天前更新
          </Descriptions.Item>
        </Descriptions>
      )}
    </Modal>
  );
}

// ── Main Page ──────────────────────────────────────────────────────

export default function SuppliersPage() {
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(10);
  const [search, setSearch] = useState('');

  // Form modal
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Supplier | null>(null);
  const [form] = Form.useForm();

  // Delete confirmation
  const [deleteTarget, setDeleteTarget] = useState<Supplier | null>(null);

  // Score detail modal
  const [scoreModalData, setScoreModalData] = useState<{ supplierName: string; score: SupplierScore } | null>(null);

  // ── Queries ──────────────────────────────────────────────────────

  const listKey = useMemo(
    () => ['crud', '/suppliers', page, size, search],
    [page, size, search],
  );

  const { data, isLoading, refetch } = useQuery({
    queryKey: listKey,
    queryFn: async () => {
      const params: Record<string, string> = { page: String(page), size: String(size) };
      if (search) params.search = String(search);
      return apiClient.getPage<Supplier>('/v1/suppliers', params);
    },
  });

  // Scoreboard — all supplier scores
  const { data: scoreboard } = useQuery({
    queryKey: ['supplier-scoreboard'],
    queryFn: async () => {
      const res = await apiClient.get('/v1/suppliers/scoreboard');
      return (res.data || []) as SupplierScore[];
    },
    staleTime: 30_000,
  });

  // Build lookup: supplier_id -> SupplierScore
  const scoreMap = useMemo(() => {
    const m = new Map<number, SupplierScore>();
    (scoreboard || []).forEach((s) => m.set(s.supplier_id, s));
    return m;
  }, [scoreboard]);

  // ── Mutations ────────────────────────────────────────────────────

  const createMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) => apiClient.post('/v1/suppliers', values),
    onSuccess: () => {
      message.success('已创建');
      setModalOpen(false);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['crud', '/suppliers'] });
    },
    onError: (e: Error) => message.error(`创建失败: ${e.message}`),
  });

  const updateMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) => apiClient.put(`/v1/suppliers/${editing?.id}`, values),
    onSuccess: () => {
      message.success('已更新');
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['crud', '/suppliers'] });
    },
    onError: (e: Error) => message.error(`更新失败: ${e.message}`),
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => apiClient.delete(`/v1/suppliers/${id}`),
    onSuccess: () => {
      message.success('已删除');
      setDeleteTarget(null);
      qc.invalidateQueries({ queryKey: ['crud', '/suppliers'] });
    },
    onError: (e: Error) => message.error(`删除失败: ${e.message}`),
  });

  const recalcMutation = useMutation({
    mutationFn: async (id: number) => apiClient.post(`/v1/suppliers/${id}/recalculate`, {}),
    onSuccess: () => {
      message.success('评分已重新计算');
      qc.invalidateQueries({ queryKey: ['crud', '/suppliers'] });
      qc.invalidateQueries({ queryKey: ['supplier-scoreboard'] });
    },
    onError: (e: Error) => message.error(`计算失败: ${e.message}`),
  });

  // ── Columns ──────────────────────────────────────────────────────

  const columns = useMemo(() => [
    { title: 'ID', dataIndex: 'id', width: 70, key: 'id' },
    { title: '供应商名称', dataIndex: 'name', width: 200, key: 'name' },
    { title: '联系人', dataIndex: 'contact_person', width: 120, key: 'contact_person' },
    { title: '电话', dataIndex: 'contact_phone', width: 140, key: 'contact_phone' },
    { title: '邮箱', dataIndex: 'email', width: 180, key: 'email' },
    { title: '地址', dataIndex: 'address', width: 220, key: 'address' },
    { title: '备注', dataIndex: 'remark', width: 180, key: 'remark' },
    { title: '状态', dataIndex: 'status', width: 80, key: 'status' },
    {
      title: '信用评分',
      key: 'reliability_score',
      width: 110,
      render: (_: unknown, record: Supplier) => {
        const score = scoreMap.get(record.id);
        if (!score) return <span style={{ color: '#999' }}>-</span>;
        return (
          <a
            style={{ color: scoreColor(score.reliability_score), fontWeight: 700 }}
            onClick={() => setScoreModalData({ supplierName: record.name, score })}
          >
            {score.reliability_score.toFixed(1)}
          </a>
        );
      },
    },
    { title: '创建时间', dataIndex: 'created_at', width: 160, key: 'created_at', render: fmtDate },
    {
      title: '操作',
      key: '__actions__',
      width: 260,
      fixed: 'right' as const,
      render: (_: unknown, record: Supplier) => (
        <Space size="small">
          <Button
            size="small"
            type="link"
            icon={<BarChartOutlined />}
            onClick={async () => {
              try {
                const res = await apiClient.get(`/v1/suppliers/${record.id}/score`);
                setScoreModalData({ supplierName: record.name, score: res.data as SupplierScore });
              } catch {
                message.info('暂无评分数据，请先点击"重新计算"');
              }
            }}
          >
            评分详情
          </Button>
          <Button
            size="small"
            type="link"
            icon={<RecalcOutlined />}
            loading={recalcMutation.isPending && recalcMutation.variables === record.id}
            onClick={() => recalcMutation.mutate(record.id)}
          >
            重新计算
          </Button>
          <Button
            size="small"
            type="link"
            icon={<EditOutlined />}
            onClick={() => {
              setEditing(record);
              form.setFieldsValue(record);
              setModalOpen(true);
            }}
          >
            编辑
          </Button>
          <Button
            size="small"
            type="link"
            danger
            icon={<DeleteOutlined />}
            onClick={() => setDeleteTarget(record)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ], [scoreMap, recalcMutation, form]);

  // ── Submit handler ───────────────────────────────────────────────

  const handleSubmit = async () => {
    const values = await form.validateFields();
    if (editing) {
      updateMutation.mutate(values);
    } else {
      createMutation.mutate(values);
    }
  };

  // ── Render ───────────────────────────────────────────────────────

  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      {/* Header */}
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 12,
      }}>
        <h1 style={{
          fontFamily: 'var(--ds)',
          fontWeight: 700,
          fontSize: 'var(--text-h1)',
          color: 'var(--t1)',
          margin: 0,
        }}>供应商</h1>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => refetch()} style={{ fontFamily: 'var(--body)', fontSize: '0.8rem' }}>
            刷新
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { setEditing(null); form.resetFields(); setModalOpen(true); }}
            style={{ fontFamily: 'var(--body)', fontSize: '0.8rem' }}
          >
            新建供应商
          </Button>
        </Space>
      </div>

      {/* Search */}
      <FilterBar
        search={search}
        onSearch={(v) => { setSearch(v); setPage(1); }}
        searchPlaceholder="搜索供应商名称 / 联系人 / 电话..."
      />

      {/* Table */}
      <Table
        rowKey="id"
        loading={isLoading}
        dataSource={data?.data}
        columns={columns}
        scroll={{ x: 'max-content' }}
        pagination={{
          current: data?.page ?? page,
          pageSize: data?.size ?? size,
          total: data?.total ?? 0,
          showSizeChanger: true,
          showTotal: (t: number) => `共 ${t} 条`,
          onChange: (p: number, s: number) => { setPage(p); setSize(s); },
        }}
      />

      {/* Create / Edit modal */}
      <Modal
        title={editing ? '编辑供应商' : '新建供应商'}
        open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditing(null); form.resetFields(); }}
        onOk={handleSubmit}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        width={560}
        destroyOnClose
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item name="name" label="供应商名称" rules={[{ required: true, message: '请输入供应商名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="contact_person" label="联系人">
            <Input />
          </Form.Item>
          <Form.Item name="contact_phone" label="电话">
            <Input />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input />
          </Form.Item>
          <Form.Item name="address" label="地址">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue={1}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>

      {/* Delete confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        title="删除供应商"
        content="确定要删除此供应商吗？此操作不可撤销。"
        okType="danger"
        okText="确认删除"
        confirmLoading={deleteMutation.isPending}
        risk="high"
        onOk={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        onCancel={() => setDeleteTarget(null)}
      />

      {/* Score detail modal */}
      <ScoreDetailModal
        score={scoreModalData?.score ?? null}
        supplierName={scoreModalData?.supplierName ?? ''}
        onClose={() => setScoreModalData(null)}
      />
    </div>
  );
}
