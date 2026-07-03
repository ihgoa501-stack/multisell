'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  message,
  Progress,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import {
  DatabaseOutlined,
  PlayCircleOutlined,
  ThunderboltOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import dayjs from 'dayjs';
import PageContainer from '@/components/ui/PageContainer';

const { Text } = Typography;

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

interface CompletenessDimension {
  dimension: string;
  label: string;
  score: number;
  weight: number;
  complete: boolean;
  reason: string;
}

interface CompletenessCheckResult {
  product_id: number;
  score: number;
  status: string;
  dimensions: CompletenessDimension[];
  missing_items: string[];
}

interface EvaluateResult {
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
}

const SUGGESTIONS: Record<string, string> = {
  '商品标题': '添加包含核心卖点和搜索关键词的标题，至少10个字符。',
  '商品描述': '编写包含功能、材质、尺寸信息的描述，至少20个字符。',
  '主图': '上传高清白底主图，清晰展示商品正面。',
  '多图': '添加多角度图片，含细节特写和使用场景。',
  '类目': '选择准确的商品类目，有助于买家搜索。',
  '品牌': '填写品牌信息，无品牌可填 OEM。',
  '规格参数（颜色/尺寸/重量）': '填写完整规格参数，包括颜色、尺寸、重量。',
  '采购成本': '填写准确的采购成本用于利润计算。',
  '包装信息（重量/尺寸）': '填写包装重量和尺寸用于物流成本核算。',
  'HS编码': '填写HS编码用于关税计算和海关申报。',
  '目标售价': '设置覆盖成本和利润的目标售价。',
  '原产地': '填写商品原产地信息。',
};

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
  const [detailProduct, setDetailProduct] = useState<CandidateProduct | null>(null);
  const [completenessResult, setCompletenessResult] = useState<CompletenessCheckResult | null>(null);
  const [completenessLoading, setCompletenessLoading] = useState(false);

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

  const handleOpenDetail = useCallback(async (product: CandidateProduct) => {
    setDetailProduct(product);
    setCompletenessResult(null);
    setCompletenessLoading(true);
    try {
      const res = await apiClient.post<CompletenessCheckResult>(
        `/v1/completeness/check/${product.id}`,
      );
      if (res.data) {
        setCompletenessResult(res.data);
      }
    } catch {
      // completeness endpoint may not be reachable; degrade to product-info-only view
    } finally {
      setCompletenessLoading(false);
    }
  }, []);

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
          status={
            (statusColorMap[status] || 'default') as
              | 'success'
              | 'error'
              | 'processing'
              | 'warning'
              | 'default'
          }
          text={statusLabelMap[status] || status}
        />
      ),
    },
    {
      title: '操作',
      width: 190,
      render: (_: unknown, record: CandidateProduct) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            onClick={(e) => {
              e.stopPropagation();
              handleOpenDetail(record);
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
    <PageContainer
      title="候选商品"
      extra={
        <Button
          type="primary"
          icon={<DatabaseOutlined />}
          onClick={handleSeed}
          loading={seeding}
        >
          生成种子数据
        </Button>
      }
    >
      {/* Evaluation result banner */}
      {lastEvaluation && (
        <Card size="small" style={{ marginBottom: 'var(--space-lg)' }}>
          <Alert
            type={
              lastEvaluation.decision === 'list'
                ? 'success'
                : lastEvaluation.decision === 'cautious'
                  ? 'warning'
                  : 'error'
            }
            message={
              lastEvaluation.decision === 'list'
                ? '系统建议上架，但仍需 Owner 审批'
                : lastEvaluation.decision === 'cautious'
                  ? '系统建议谨慎处理'
                  : '系统不建议上架'
            }
            description={lastEvaluation.reason}
            showIcon
          />
          {lastEvaluation.listing_task_id && (
            <Space style={{ marginTop: 'var(--space-md)' }}>
              <Tag color="orange">待审批</Tag>
              <Text>
                已生成刊登任务 #{lastEvaluation.listing_task_id}
                ，审批通过前不会执行发布。
              </Text>
              <Button
                type="primary"
                size="small"
                onClick={() => router.push('/approval')}
              >
                去审批
              </Button>
              <Button
                size="small"
                onClick={() =>
                  router.push(
                    `/listing-tasks/${lastEvaluation.listing_task_id}`,
                  )
                }
              >
                查看任务
              </Button>
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
            onClick: () => handleOpenDetail(record),
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

      {/* Detail card */}
      {detailProduct && (
        <Card
          size="small"
          style={{ marginTop: 'var(--space-lg)' }}
          title={
            <Space>
              <Text strong>候选商品 #{detailProduct.id} 详情</Text>
              {completenessLoading && (
                <Text type="secondary" style={{ fontSize: '0.85rem' }}>
                  加载完整度数据...
                </Text>
              )}
            </Space>
          }
          extra={
            <Space size="small">
              <Button
                type="primary"
                size="small"
                icon={<ThunderboltOutlined />}
                loading={evaluating === detailProduct.id}
                onClick={() => handleEvaluate(detailProduct.id)}
              >
                执行完整度+利润评估
              </Button>
              <Button
                size="small"
                icon={<CloseOutlined />}
                onClick={() => setDetailProduct(null)}
              />
            </Space>
          }
        >
          {/* Product info */}
          <div
            style={{
              marginBottom: 'var(--space-md)',
              color: 'var(--t2)',
              lineHeight: 1.8,
            }}
          >
            <Text
              strong
              style={{
                fontSize: 'var(--text-body-b)',
                color: 'var(--t1)',
              }}
            >
              {detailProduct.title}
            </Text>
            <div style={{ marginTop: 'var(--space-sm)' }}>
              采购价：¥{detailProduct.purchase_price?.toFixed(2)}
              <span style={{ marginLeft: 20 }}>
                目标售价：${detailProduct.target_sale_price?.toFixed(2)}
              </span>
              <span style={{ marginLeft: 20 }}>
                目标平台：
                {detailProduct.target_platform_id
                  ? platformLabelMap[
                      String(detailProduct.target_platform_id)
                    ] || `平台 #${detailProduct.target_platform_id}`
                  : '-'}
              </span>
            </div>
            <div style={{ marginTop: 'var(--space-xs)' }}>
              包装：{detailProduct.package_weight_kg?.toFixed(2)}kg
              <span style={{ marginLeft: 20 }}>
                HS编码：{detailProduct.hs_code || '-'}
              </span>
              <span style={{ marginLeft: 20 }}>
                状态：
                <Tag
                  color={
                    statusColorMap[detailProduct.status] || 'default'
                  }
                >
                  {statusLabelMap[detailProduct.status] ||
                    detailProduct.status}
                </Tag>
                {detailProduct.is_seed_data && (
                  <Tag color="orange" style={{ marginLeft: 4 }}>
                    种子数据
                  </Tag>
                )}
              </span>
            </div>
            <div style={{ marginTop: 'var(--space-xs)' }}>
              来源：{detailProduct.created_by || '-'}
              <span style={{ marginLeft: 20 }}>
                创建时间：
                {detailProduct.created_at
                  ? dayjs(detailProduct.created_at).format(
                      'YYYY-MM-DD HH:mm:ss',
                    )
                  : '-'}
              </span>
            </div>
          </div>

          {/* Completeness breakdown */}
          {completenessResult && (
            <>
              {/* Score header */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'baseline',
                  gap: 8,
                  marginBottom: 'var(--space-md)',
                }}
              >
                <Text strong>完整度评分：</Text>
                <Text
                  style={{
                    fontSize: 'var(--text-h2)',
                    fontWeight: 700,
                    color:
                      completenessResult.score >= 80
                        ? 'var(--g4)'
                        : completenessResult.score >= 50
                          ? 'var(--y4)'
                          : 'var(--r4)',
                  }}
                >
                  {completenessResult.score.toFixed(0)}
                </Text>
                <Text type="secondary">/ 100</Text>
                <Tag
                  color={
                    completenessResult.status === 'complete'
                      ? 'green'
                      : 'orange'
                  }
                >
                  {completenessResult.status === 'complete'
                    ? '完整'
                    : '不完整'}
                </Tag>
              </div>

              {/* Missing items with suggestions */}
              {completenessResult.missing_items.length > 0 && (
                <Card
                  size="small"
                  type="inner"
                  title="缺失项与改进建议"
                  style={{ marginBottom: 'var(--space-md)' }}
                >
                  {completenessResult.missing_items.map((item) => (
                    <div
                      key={item}
                      style={{ marginBottom: 'var(--space-sm)' }}
                    >
                      <Tag color="error" style={{ marginRight: 8 }}>
                        {item}
                      </Tag>
                      <Text type="secondary">
                        {SUGGESTIONS[item] || '请补充此项信息'}
                      </Text>
                    </div>
                  ))}
                </Card>
              )}

              {/* Dimension breakdown */}
              <Card size="small" type="inner" title="各维度评分明细">
                {completenessResult.dimensions.map((dim) => (
                  <div
                    key={dim.dimension}
                    style={{ marginBottom: dim.reason ? 12 : 8 }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 12,
                      }}
                    >
                      <div
                        style={{
                          width: 150,
                          flexShrink: 0,
                          fontSize: '0.85rem',
                          color: 'var(--t2)',
                        }}
                      >
                        {dim.label}
                      </div>
                      <Progress
                        percent={Math.round(dim.score)}
                        size="small"
                        style={{ flex: 1, marginBottom: 0 }}
                        strokeColor={
                          dim.complete ? 'var(--g4)' : 'var(--y4)'
                        }
                      />
                      <Tag
                        color={dim.complete ? 'green' : 'orange'}
                        style={{
                          flexShrink: 0,
                          margin: 0,
                          fontSize: '0.72rem',
                        }}
                      >
                        {dim.complete ? 'OK' : '缺'}
                      </Tag>
                    </div>
                    {!dim.complete && dim.reason && (
                      <div
                        style={{
                          paddingLeft: 162,
                          fontSize: '0.78rem',
                          color: 'var(--t3)',
                          lineHeight: 1.4,
                          marginTop: 2,
                        }}
                      >
                        {dim.reason}
                      </div>
                    )}
                  </div>
                ))}
              </Card>
            </>
          )}

          {/* Last evaluation result inline when it matches the selected product */}
          {lastEvaluation &&
            lastEvaluation.product_id === detailProduct.id && (
              <div style={{ marginTop: 'var(--space-md)' }}>
                <Card size="small" type="inner" title="评估结果">
                  <div style={{ color: 'var(--t2)', lineHeight: 1.8 }}>
                    <div>
                      决策：
                      <Tag
                        color={
                          lastEvaluation.decision === 'list'
                            ? 'green'
                            : lastEvaluation.decision === 'cautious'
                              ? 'orange'
                              : 'red'
                        }
                      >
                        {lastEvaluation.decision === 'list'
                          ? '建议上架'
                          : lastEvaluation.decision === 'cautious'
                            ? '谨慎处理'
                            : '不建议上架'}
                      </Tag>
                      <span style={{ marginLeft: 20 }}>
                        置信度：
                        {(lastEvaluation.confidence * 100).toFixed(0)}%
                      </span>
                    </div>
                    <div>
                      利润率：{lastEvaluation.profit_margin.toFixed(2)}%
                      <span style={{ marginLeft: 20 }}>
                        预估利润：$
                        {lastEvaluation.estimated_profit.toFixed(2)}
                      </span>
                      <span style={{ marginLeft: 20 }}>
                        利润状态：
                        <Tag
                          color={
                            lastEvaluation.profit_status === 'profitable'
                              ? 'green'
                              : 'orange'
                          }
                        >
                          {lastEvaluation.profit_status}
                        </Tag>
                      </span>
                    </div>
                    <div>
                      评估理由：{lastEvaluation.reason}
                    </div>
                    {lastEvaluation.risk_flags.length > 0 && (
                      <div>
                        风险标记：
                        {lastEvaluation.risk_flags.map((flag) => (
                          <Tag key={flag} color="red" style={{ marginTop: 4 }}>
                            {flag}
                          </Tag>
                        ))}
                      </div>
                    )}
                  </div>
                </Card>
              </div>
            )}
        </Card>
      )}
    </PageContainer>
  );
}
