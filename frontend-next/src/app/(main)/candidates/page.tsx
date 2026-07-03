'use client';

import { useState, useEffect } from 'react';
import {
  Alert, Badge, Button, Card, Input, Modal, message, Space, Table, Tag, Tabs, Typography, Select
} from 'antd';
import { DatabaseOutlined, PlayCircleOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import dayjs from 'dayjs';

// ── Types ──────────────────────────────────────────────────────────

interface CandidateProduct {
  id: number;
  title: string;
  description: string;
  main_image: string;
  purchase_price: number;
  purchase_currency: string;
  package_weight_kg: number;
  target_sale_price: number;
  target_currency: string;
  completeness_status: string;
  source_platform: string;
  source_url: string;
  target_platform_id: number | null;
  destination_country: string;
  hs_code: string;
  origin_country: string;
  status: string;
  is_seed_data: boolean;
  created_by: string;
  created_at: string;
}

interface CollectLead {
  id: number;
  title: string;
  price_range: string;
  detail_url: string;
  image_url: string;
  shop_hint: string;
  source_page_url: string;
  status: string;
  collected_at: string;
  created_at: string;
}

type EvaluateResult = {
  product_id: number;
  title: string;
  completeness_score: number;
  completeness_status: string;
  missing_items: string[];
  profit_margin: number;
  estimated_profit: number;
  profit_status: string;
  decision: 'list' | 'cautious' | 'skip';
  confidence: number;
  reason: string;
  risk_flags: string[];
  listing_task_id?: number | null;
};

// ── Maps ────────────────────────────────────────────────────────────

const statusColorMap: Record<string, string> = {
  draft: 'default', in_review: 'processing', approved: 'success', rejected: 'error',
};
const statusLabelMap: Record<string, string> = {
  draft: '草稿', in_review: '审核中', approved: '已通过', rejected: '已拒绝',
};
const completenessColorMap: Record<string, string> = {
  collected: 'default', incomplete: 'warning', ready_for_profit_check: 'success', rejected: 'error',
};
const completenessLabelMap: Record<string, string> = {
  collected: '已采集', incomplete: '资料不完整', ready_for_profit_check: '可测算利润', rejected: '已拒绝',
};
const platformLabelMap: Record<string, string> = {
  '1': 'Ozon', '2': 'Shopee', '3': 'Lazada',
};
const leadStatusLabelMap: Record<string, string> = {
  pending_detail_collect: '待采集详情', collecting: '采集中', collected: '已采集', skipped: '已跳过',
};

// ── Components ─────────────────────────────────────────────────────

/** CollectLead list tab */
function CollectLeadTable() {
  const [items, setItems] = useState<CollectLead[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState<string | undefined>();
  const [detail, setDetail] = useState<CollectLead | null>(null);

  const doFetch = async (p: number, ps: number, st?: string) => {
    setLoading(true);
    try {
      const params: Record<string, string> = { page: String(p), size: String(ps) };
      if (st) params.status = st;
      const res = await apiClient.get('/v1/candidates/collect-leads', params);
      const body = res as unknown as { data: CollectLead[]; total: number };
      setItems(body.data || []);
      setTotal(body.total || 0);
    } catch { message.error('加载采集线索失败'); }
    finally { setLoading(false); }
  };

  // eslint-disable-next-line -- standard data-fetching pattern, setState on async resolve
  useEffect(() => { doFetch(page, pageSize, statusFilter); }, [page, pageSize, statusFilter]);

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '标题', dataIndex: 'title', ellipsis: true, render: (v: string) => v || '-' },
    { title: '价格', dataIndex: 'price_range', width: 110, render: (v: string) => v || '-' },
    {
      title: '来源',
      width: 160, ellipsis: true,
      render: (_: unknown, r: CollectLead) => (
        <a href={r.source_page_url} target="_blank" rel="noopener noreferrer" onClick={e => e.stopPropagation()}>
          {r.source_page_url ? '打开列表页' : '-'}
        </a>
      ),
    },
    {
      title: '详情页',
      width: 160, ellipsis: true,
      render: (_: unknown, r: CollectLead) => (
        <a href={r.detail_url} target="_blank" rel="noopener noreferrer" onClick={e => e.stopPropagation()}>
          {r.detail_url ? '打开详情' : '-'}
        </a>
      ),
    },
    {
      title: '状态', dataIndex: 'status', width: 120,
      render: (s: string) => <Tag>{leadStatusLabelMap[s] || s}</Tag>,
    },
    {
      title: '采集时间', dataIndex: 'collected_at', width: 160,
      render: (v: string) => v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-',
    },
  ];

  return (
    <>
      <Card size="small" style={{ marginBottom: 'var(--space-lg)' }} styles={{ body: { padding: '12px 20px', display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' } }}>
        <Select allowClear placeholder="线索状态" style={{ width: 150 }} value={statusFilter} onChange={v => { setStatusFilter(v); setPage(1); }}
          options={[
            { value: 'pending_detail_collect', label: '待采集详情' },
            { value: 'collecting', label: '采集中' },
            { value: 'collected', label: '已采集' },
            { value: 'skipped', label: '已跳过' },
          ]} />
      </Card>
      <Card size="small" styles={{ body: { padding: 0 } }}>
        <Table<CollectLead>
          rowKey="id" columns={columns} dataSource={items} loading={loading}
          onRow={(r) => ({ onClick: () => setDetail(r), style: { cursor: 'pointer' } })}
          pagination={{
            current: page, pageSize, total, showSizeChanger: true, showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps); },
          }}
          scroll={{ x: 800 }}
        />
      </Card>
      <Modal title={detail ? `采集线索 #${detail.id}` : ''} open={!!detail} onCancel={() => setDetail(null)} footer={null} width={560}>
        {detail && (
          <div style={{ lineHeight: 2 }}>
            <div><strong>标题：</strong>{detail.title || '-'}</div>
            <div><strong>价格：</strong>{detail.price_range || '-'}</div>
            <div><strong>店铺：</strong>{detail.shop_hint || '-'}</div>
            {detail.detail_url && <div><strong>详情页：</strong><a href={detail.detail_url} target="_blank" rel="noopener noreferrer">打开</a></div>}
            {detail.source_page_url && <div><strong>列表页：</strong><a href={detail.source_page_url} target="_blank" rel="noopener noreferrer">打开</a></div>}
            <div><strong>状态：</strong>{leadStatusLabelMap[detail.status] || detail.status}</div>
            <div><strong>采集时间：</strong>{detail.collected_at ? dayjs(detail.collected_at).format('YYYY-MM-DD HH:mm') : '-'}</div>
          </div>
        )}
      </Modal>
    </>
  );
}

/** CandidateProduct list tab */
function CandidateProductTable() {
  const router = useRouter();
  const [data, setData] = useState<CandidateProduct[]>([]);
  const [loading, setLoading] = useState(false);
  const [evaluating, setEvaluating] = useState<number | null>(null);
  const [lastEvaluation, setLastEvaluation] = useState<EvaluateResult | null>(null);
  const [seeding, setSeeding] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [detailModal, setDetailModal] = useState<CandidateProduct | null>(null);
  const [search, setSearch] = useState('');
  const [csFilter, setCsFilter] = useState<string | undefined>();
  const [spFilter, setSpFilter] = useState<string | undefined>();

  const doFetchCandidates = async (p: number, ps: number) => {
    setLoading(true);
    try {
      const params: Record<string, string> = { page: String(p), size: String(ps) };
      if (csFilter) params.completeness_status = csFilter;
      if (spFilter) params.source_platform = spFilter;
      if (search) params.search = search;
      const res = await apiClient.get('/v1/candidates', params);
      const body = res as unknown as { data: CandidateProduct[]; total: number };
      setData(body.data || []);
      setTotal(body.total || 0);
    } catch { message.error('加载候选商品列表失败'); }
    finally { setLoading(false); }
  };

  // eslint-disable-next-line -- standard data-fetching pattern
  useEffect(() => { doFetchCandidates(page, pageSize); }, [page, pageSize, csFilter, spFilter, search]);

  const handleEvaluate = async (productId: number) => {
    setEvaluating(productId);
    setLastEvaluation(null);
    try {
      const res = await apiClient.post<EvaluateResult>(`/v1/loop/evaluate/${productId}`);
      if (res.data) { setLastEvaluation(res.data); message.success('评估完成'); }
      else { message.error(res.message || '评估失败'); }
    } catch { message.error('评估请求失败'); }
    finally { setEvaluating(null); }
  };

  const handleSeed = async () => {
    setSeeding(true);
    try {
      const res = await apiClient.post('/v1/candidates/seed');
      if (res.code === 0) { message.success('种子数据生成成功'); await doFetchCandidates(page, pageSize); }
      else { message.error(res.message || '种子数据生成失败'); }
    } catch { message.error('种子数据请求失败'); }
    finally { setSeeding(false); }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '标题', dataIndex: 'title', ellipsis: true },
    {
      title: '来源平台', dataIndex: 'source_platform', width: 90,
      render: (v: string) => v ? <Tag>{v}</Tag> : '-',
    },
    { title: '采购价', dataIndex: 'purchase_price', width: 90, render: (p: number) => p != null ? `¥${p.toFixed(2)}` : '-' },
    { title: '目标售价', dataIndex: 'target_sale_price', width: 100, render: (p: number) => p != null ? `$${p.toFixed(2)}` : '-' },
    {
      title: '目标平台', dataIndex: 'target_platform_id', width: 90,
      render: (id: number | null) => id != null ? platformLabelMap[String(id)] || `#${id}` : '-',
    },
    {
      title: '完整度', dataIndex: 'completeness_status', width: 110,
      render: (s: string) => s ? (
        <Badge status={(completenessColorMap[s] || 'default') as 'success' | 'error' | 'processing' | 'warning' | 'default'} text={completenessLabelMap[s] || s} />
      ) : '-',
    },
    {
      title: '状态', dataIndex: 'status', width: 80,
      render: (s: string) => (
        <Badge status={(statusColorMap[s] || 'default') as 'success' | 'error' | 'processing' | 'warning' | 'default'} text={statusLabelMap[s] || s} />
      ),
    },
    {
      title: '操作', width: 140,
      render: (_: unknown, r: CandidateProduct) => (
        <Space size="small">
          <Button type="link" size="small" onClick={(e) => { e.stopPropagation(); setDetailModal(r); }}>详情</Button>
          <Button size="small" icon={<PlayCircleOutlined />} loading={evaluating === r.id}
            onClick={(e) => { e.stopPropagation(); handleEvaluate(r.id); }}>评估</Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      {/* Toolbar */}
      <Card size="small" style={{ marginBottom: 'var(--space-lg)' }} styles={{ body: { padding: '12px 20px', display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' } }}>
        <Button type="primary" icon={<DatabaseOutlined />} onClick={handleSeed} loading={seeding}>生成种子数据</Button>
        <Input.Search allowClear placeholder="搜索标题/描述" style={{ width: 220 }} value={search}
          onChange={e => setSearch(e.target.value)}
          onSearch={v => { setSearch(v); setPage(1); }} />
        <Select allowClear placeholder="完整度" style={{ width: 140 }} value={csFilter} onChange={v => { setCsFilter(v); setPage(1); }}
          options={[
            { value: 'collected', label: '已采集' },
            { value: 'incomplete', label: '资料不完整' },
            { value: 'ready_for_profit_check', label: '可测算利润' },
          ]} />
        <Select allowClear placeholder="来源平台" style={{ width: 110 }} value={spFilter} onChange={v => { setSpFilter(v); setPage(1); }}
          options={[
            { value: '1688', label: '1688' },
            { value: 'chrome_extension', label: 'Chrome扩展' },
          ]} />
      </Card>

      {/* Evaluation result */}
      {lastEvaluation && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Alert
            type={lastEvaluation.decision === 'list' ? 'success' : lastEvaluation.decision === 'cautious' ? 'warning' : 'error'}
            message={lastEvaluation.decision === 'list' ? '系统建议上架，但仍需 Owner 审批' : lastEvaluation.decision === 'cautious' ? '系统建议谨慎处理' : '系统不建议上架'}
            description={lastEvaluation.reason} showIcon
            style={{ marginBottom: lastEvaluation.listing_task_id ? 12 : 0 }}
          />
          {lastEvaluation.listing_task_id && (
            <Space style={{ marginTop: 'var(--space-md)' }}>
              <Tag color="orange">待审批</Tag>
              <Typography.Text>已生成刊登任务 #{lastEvaluation.listing_task_id}，审批通过前不会执行发布。</Typography.Text>
              <Button type="primary" onClick={() => router.push('/approval')}>去审批</Button>
              <Button onClick={() => router.push(`/listing-tasks/${lastEvaluation.listing_task_id}`)}>查看任务</Button>
            </Space>
          )}
        </Card>
      )}

      {/* Table */}
      <Card size="small" styles={{ body: { padding: 0 } }}>
        <Table<CandidateProduct> rowKey="id" columns={columns} dataSource={data} loading={loading}
          onRow={(r) => ({ onClick: () => setDetailModal(r), style: { cursor: 'pointer' } })}
          pagination={{ current: page, pageSize, total, showSizeChanger: true, showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps); } }}
          scroll={{ x: 900 }}
        />
      </Card>

      {/* Detail Modal */}
      <Modal title={detailModal ? `候选商品 #${detailModal.id}` : ''} open={!!detailModal}
        onCancel={() => setDetailModal(null)} footer={null} width={640}>
        {detailModal && (
          <div style={{ lineHeight: 2 }}>
            <Typography.Title level={5} style={{ marginTop: 0 }}>{detailModal.title}</Typography.Title>
            <div>
              <strong>来源平台：</strong>{detailModal.source_platform || '-'}
              {detailModal.source_url && <> · <a href={detailModal.source_url} target="_blank" rel="noopener noreferrer">打开原始来源</a></>}
            </div>
            <div><strong>完整度：</strong>
              <Badge status={(completenessColorMap[detailModal.completeness_status] || 'default') as 'success' | 'error' | 'processing' | 'warning' | 'default'}
                text={completenessLabelMap[detailModal.completeness_status] || detailModal.completeness_status} />
            </div>
            <div><strong>采购价：</strong>¥{detailModal.purchase_price?.toFixed(2) || '-'}</div>
            <div><strong>包装：</strong>{detailModal.package_weight_kg ? `${detailModal.package_weight_kg.toFixed(2)}kg` : '-'}</div>
            {detailModal.source_url && <div><strong>来源：</strong><a href={detailModal.source_url} target="_blank" rel="noopener noreferrer">{detailModal.source_url}</a></div>}
            <div><strong>状态：</strong><Tag color={statusColorMap[detailModal.status] || 'default'}>{statusLabelMap[detailModal.status] || detailModal.status}</Tag>
              {detailModal.is_seed_data && <Tag color="orange" style={{ marginLeft: 8 }}>种子数据</Tag>}
            </div>
            <div><strong>创建时间：</strong>{detailModal.created_at ? dayjs(detailModal.created_at).format('YYYY-MM-DD HH:mm:ss') : '-'}</div>
            <Button type="primary" icon={<ThunderboltOutlined />} style={{ marginTop: 12 }}
              onClick={() => { handleEvaluate(detailModal.id); setDetailModal(null); }}>
              执行完整度+利润评估
            </Button>
          </div>
        )}
      </Modal>
    </>
  );
}

// ── Page ────────────────────────────────────────────────────────────

export default function CandidatesPage() {
  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', color: 'var(--t1)', margin: '0 0 16px 0' }}>
        候选商品
      </h1>
      <Tabs
        defaultActiveKey="candidates"
        items={[
          { key: 'leads', label: '采集线索', children: <CollectLeadTable /> },
          { key: 'candidates', label: '候选商品', children: <CandidateProductTable /> },
        ]}
      />
    </div>
  );
}
