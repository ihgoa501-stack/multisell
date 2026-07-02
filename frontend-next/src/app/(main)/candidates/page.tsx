'use client';

import { useState, useEffect } from 'react';
import { Alert, Badge, Button, Card, Modal, message, Space, Table, Tag, Typography } from 'antd';
import { DatabaseOutlined, PlayCircleOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import dayjs from 'dayjs';

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
  target_platform_id: number | null;
  destination_country: string;
  hs_code: string;
  origin_country: string;
  status: string;
  is_seed_data: boolean;
  created_by: string;
  created_at: string;
}

const statusColorMap: Record<string, string> = {
  draft: 'default',
  in_review: 'processing',
  approved: 'success',
  rejected: 'error',
};

const statusLabelMap: Record<string, string> = {
  draft: '草稿',
  in_review: '审核中',
  approved: '已通过',
  rejected: '已拒绝',
};

const platformLabelMap: Record<string, string> = {
  '1': 'Ozon',
  '2': 'Shopee',
  '3': 'Lazada',
};

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

export default function CandidatesPage() {
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

  const fetchCandidates = async (p: number, ps: number) => {
    setLoading(true);
    try {
      const res = await apiClient.get<CandidateProduct[]>('/v1/candidates', {
        page: String(p),
        size: String(ps),
      });
      const body = res as unknown as { data: CandidateProduct[]; total: number };
      setData(body.data || []);
      setTotal(body.total || 0);
    } catch {
      message.error('加载候选商品列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCandidates(page, pageSize);
  }, [page, pageSize]);

  const handleEvaluate = async (productId: number) => {
    setEvaluating(productId);
    setLastEvaluation(null);
    try {
      const res = await apiClient.post<EvaluateResult>(`/v1/loop/evaluate/${productId}`);
      if (res.data) {
        setLastEvaluation(res.data);
        message.success('评估完成');
      } else {
        message.error(res.message || '评估失败');
      }
    } catch {
      message.error('评估请求失败');
    } finally {
      setEvaluating(null);
    }
  };

  const handleSeed = async () => {
    setSeeding(true);
    try {
      const res = await apiClient.post('/v1/candidates/seed');
      if (res.code === 0) {
        message.success('种子数据生成成功');
        await fetchCandidates(page, pageSize);
      } else {
        message.error(res.message || '种子数据生成失败');
      }
    } catch {
      message.error('种子数据请求失败');
    } finally {
      setSeeding(false);
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
    },
    {
      title: '采购价',
      dataIndex: 'purchase_price',
      width: 100,
      render: (price: number) => (price != null ? `¥${price.toFixed(2)}` : '-'),
    },
    {
      title: '目标售价',
      dataIndex: 'target_sale_price',
      width: 110,
      render: (price: number) => (price != null ? `$${price.toFixed(2)}` : '-'),
    },
    {
      title: '目标平台',
      dataIndex: 'target_platform_id',
      width: 100,
      render: (id: number) =>
        id ? platformLabelMap[String(id)] || `平台 #${id}` : '-',
    },
    {
      title: '目的国',
      dataIndex: 'destination_country',
      width: 80,
    },
    {
      title: '重量',
      dataIndex: 'package_weight_kg',
      width: 80,
      render: (w: number) => (w != null ? `${w.toFixed(2)}kg` : '-'),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => (
        <Badge
          status={(statusColorMap[status] || 'default') as 'success' | 'error' | 'processing' | 'warning' | 'default'}
          text={statusLabelMap[status] || status}
        />
      ),
    },
    {
      title: '操作',
      width: 140,
      render: (_: unknown, record: CandidateProduct) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            onClick={(e) => {
              e.stopPropagation();
              setDetailModal(record);
            }}
          >
            详情
          </Button>
          <Button
            size="small"
            icon={<PlayCircleOutlined />}
            loading={evaluating === record.id}
            onClick={(e) => {
              e.stopPropagation();
              handleEvaluate(record.id);
            }}
          >
            评估
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div
      style={{
        padding: '16px 20px',
        background: 'var(--bg)',
        minHeight: '100%',
      }}
    >
      <h1
        style={{
          fontFamily: 'var(--ds)',
          fontWeight: 700,
          fontSize: 'var(--text-h1)',
          color: 'var(--t1)',
          margin: '0 0 16px 0',
        }}
      >
        候选商品
      </h1>

      {/* Toolbar */}
      <Card
        size="small"
        style={{ marginBottom: 'var(--space-lg)' }}
        styles={{
          body: {
            padding: '12px 20px',
            display: 'flex',
            alignItems: 'center',
            gap: 12,
          },
        }}
      >
        <Button
          type="primary"
          icon={<DatabaseOutlined />}
          onClick={handleSeed}
          loading={seeding}
        >
          生成种子数据
        </Button>
      </Card>

      {/* Evaluation result */}
      {lastEvaluation && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Alert
            type={lastEvaluation.decision === 'list' ? 'success' : lastEvaluation.decision === 'cautious' ? 'warning' : 'error'}
            message={
              lastEvaluation.decision === 'list'
                ? '系统建议上架，但仍需 Owner 审批'
                : lastEvaluation.decision === 'cautious'
                  ? '系统建议谨慎处理'
                  : '系统不建议上架'
            }
            description={lastEvaluation.reason}
            showIcon
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
        <Table<CandidateProduct>
          rowKey="id"
          columns={columns}
          dataSource={data}
          loading={loading}
          onRow={(record) => ({
            onClick: () => setDetailModal(record),
            style: { cursor: 'pointer' },
          })}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
          scroll={{ x: 850 }}
        />
      </Card>

      {/* Detail Modal */}
      <Modal
        title={detailModal ? `候选商品 #${detailModal.id}` : ''}
        open={!!detailModal}
        onCancel={() => setDetailModal(null)}
        footer={null}
        width={600}
      >
        {detailModal && (
          <div>
            <Typography.Title level={5} style={{ marginTop: 0 }}>
              {detailModal.title}
            </Typography.Title>

            <div
              style={{
                marginBottom: 'var(--space-lg)',
                color: 'var(--t2)',
                lineHeight: 1.8,
              }}
            >
              <div>采购价：¥{detailModal.purchase_price?.toFixed(2)}</div>
              <div>
                目标售价：${detailModal.target_sale_price?.toFixed(2)} {detailModal.target_currency}
              </div>
              <div>
                目标平台：
                {detailModal.target_platform_id
                  ? platformLabelMap[String(detailModal.target_platform_id)] || `平台 #${detailModal.target_platform_id}`
                  : '-'}
                {' → '}
                目的国：{detailModal.destination_country || '-'}
              </div>
              <div>
                包装：{detailModal.package_weight_kg?.toFixed(2)}kg |
                HS编码：{detailModal.hs_code || '-'}
              </div>
              <div>
                状态：<Tag color={statusColorMap[detailModal.status] || 'default'}>
                  {statusLabelMap[detailModal.status] || detailModal.status}
                </Tag>
                {detailModal.is_seed_data && (
                  <Tag color="orange" style={{ marginLeft: 8 }}>种子数据</Tag>
                )}
              </div>
              <div>来源：{detailModal.created_by || '-'}</div>
              <div>
                创建时间：
                {detailModal.created_at
                  ? dayjs(detailModal.created_at).format('YYYY-MM-DD HH:mm:ss')
                  : '-'}
              </div>
            </div>

            <Button
              type="primary"
              icon={<ThunderboltOutlined />}
              onClick={() => {
                handleEvaluate(detailModal.id);
                setDetailModal(null);
              }}
            >
              执行完整度+利润评估
            </Button>
          </div>
        )}
      </Modal>
    </div>
  );
}
