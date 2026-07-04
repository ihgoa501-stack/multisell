'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  Input,
  InputNumber,
  message,
  Progress,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  CheckOutlined,
  CloseOutlined,
  DatabaseOutlined,
  EditOutlined,
  PlayCircleOutlined,
  StopOutlined,
  ThunderboltOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import dayjs from 'dayjs';
import PageContainer from '@/components/ui/PageContainer';

const { Text } = Typography;

const completenessColorMap: Record<string, string> = {
  incomplete: 'default',
  needs_review: 'warning',
  research_ready: 'processing',
  listing_ready: 'success',
};

const completenessLabelMap: Record<string, string> = {
  incomplete: '不完整',
  needs_review: '待补充',
  research_ready: '可调研',
  listing_ready: '可上架',
};

const completenessHintMap: Record<string, string> = {
  incomplete: '缺少核心信息（标题、采购价、主图），补充后才能继续',
  needs_review: '已有关键信息，补充供应商和包装信息后可进入调研',
  research_ready: '信息基本完整，可以执行利润分析和选品调研',
  listing_ready: '所有信息完整，可以准备上架草稿',
};

// Field label map: internal field name -> Chinese label
// ponytail: mirrors Go backend's completenessFieldNames (candidate/model.go).
// Both must be updated together when adding a new completable field.
const FIELD_LABELS: Record<string, string> = {
  title: '标题',
  purchase_price: '采购价',
  main_image: '主图',
  supplier_id: '供应商',
  package_weight_kg: '包装重量',
  package_length_cm: '包装长度',
  package_width_cm: '包装宽度',
  package_height_cm: '包装高度',
  hs_code: 'HS编码',
  target_sale_price: '目标售价',
  origin_country: '原产地',
};

// Field type map: internal field name -> input type
const FIELD_TYPES: Record<string, 'string' | 'number'> = {
  title: 'string',
  purchase_price: 'number',
  main_image: 'string',
  supplier_id: 'number',
  package_weight_kg: 'number',
  package_length_cm: 'number',
  package_width_cm: 'number',
  package_height_cm: 'number',
  hs_code: 'string',
  target_sale_price: 'number',
  origin_country: 'string',
};

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
  completeness_status: string;
  is_seed_data: boolean;
  created_by: string;
  created_at: string;
}

interface CandidateDetail extends CandidateProduct {
  missing_fields: string[];
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
  const [detailMissingFields, setDetailMissingFields] = useState<string[]>([]);
  const [completenessResult, setCompletenessResult] = useState<CompletenessCheckResult | null>(null);
  const [completenessLoading, setCompletenessLoading] = useState(false);
  const [completenessFilter, setCompletenessFilter] = useState<string>('');
  const [fillingField, setFillingField] = useState<string | null>(null);
  const [fillValues, setFillValues] = useState<Record<string, string>>({});
  const [actionLoading, setActionLoading] = useState<Record<string, boolean>>({});

  const fetchCandidates = useCallback(async (p: number, ps: number) => {
    setLoading(true);
    try {
      const params: Record<string, string> = { page: String(p), size: String(ps) };
      if (completenessFilter) params.completeness_status = completenessFilter;
      const res = await apiClient.get<CandidateProduct[]>('/v1/candidates', params);
      const body = res as unknown as { data: CandidateProduct[]; total: number };
      setData(body.data || []);
      setTotal(body.total || 0);
    } catch {
      message.error('加载候选商品列表失败');
    } finally {
      setLoading(false);
    }
  }, [completenessFilter]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void fetchCandidates(page, pageSize);
    }, 0);

    return () => window.clearTimeout(timer);
  }, [fetchCandidates, page, pageSize]);

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
      if (res.code === 0 && res.data) {
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

  const refreshDetail = useCallback(async (productId: number) => {
    try {
      const [detailRes, completenessRes] = await Promise.all([
        apiClient.get<CandidateDetail>(`/v1/candidates/${productId}`),
        apiClient.post<CompletenessCheckResult>(`/v1/completeness/check/${productId}`),
      ]);
      if (detailRes.data) {
        setDetailProduct(detailRes.data);
        setDetailMissingFields(detailRes.data.missing_fields || []);
      }
      if (completenessRes.data) {
        setCompletenessResult(completenessRes.data);
      }
    } catch {
      // partial failure — degrade gracefully
    }
  }, []);

  const handleOpenDetail = useCallback(async (product: CandidateProduct) => {
    setDetailProduct(product);
    setDetailMissingFields([]);
    setCompletenessResult(null);
    setCompletenessLoading(true);
    setFillingField(null);
    setFillValues({});
    try {
      const [detailRes, completenessRes] = await Promise.all([
        apiClient.get<CandidateDetail>(`/v1/candidates/${product.id}`),
        apiClient.post<CompletenessCheckResult>(`/v1/completeness/check/${product.id}`),
      ]);
      if (detailRes.data) {
        setDetailProduct(detailRes.data);
        setDetailMissingFields(detailRes.data.missing_fields || []);
      }
      if (completenessRes.data) {
        setCompletenessResult(completenessRes.data);
      }
    } catch {
      message.error('加载详情失败');
      setDetailProduct(null);
      setDetailMissingFields([]);
      setCompletenessResult(null);
    } finally {
      setCompletenessLoading(false);
    }
  }, []);

  const handleFillField = useCallback(async (productId: number, field: string) => {
    const value = fillValues[field];
    if (!value) {
      message.warning('请输入值');
      return;
    }
    setActionLoading((prev) => ({ ...prev, [field]: true }));
    try {
      const parsedValue = FIELD_TYPES[field] === 'number' ? Number(value) : value;
      const res = await apiClient.put(`/v1/candidates/${productId}/fields`, {
        fields: [{ field, value: parsedValue }],
      });
      if (res.code === 0 && res.data) {
        message.success(`"${FIELD_LABELS[field] || field}" 已更新`);
        setDetailProduct(res.data as unknown as CandidateProduct);
        setDetailMissingFields((res.data as any).missing_fields || []);
        setFillingField(null);
        setFillValues((prev) => ({ ...prev, [field]: '' }));
      } else {
        message.error(res.message || '更新失败');
      }
    } catch {
      message.error('更新请求失败');
    } finally {
      setActionLoading((prev) => ({ ...prev, [field]: false }));
    }
  }, [fillValues]);

  const handleSkipField = useCallback(async (productId: number, field: string) => {
    setActionLoading((prev) => ({ ...prev, [field]: true }));
    try {
      const res = await apiClient.post(`/v1/candidates/${productId}/skip-field`, { field });
      if (res.code === 0 && res.data) {
        message.success(`"${FIELD_LABELS[field] || field}" 已标记为无法补齐`);
        setDetailProduct(res.data as unknown as CandidateProduct);
        setDetailMissingFields((res.data as any).missing_fields || []);
      } else {
        message.error(res.message || '操作失败');
      }
    } catch {
      message.error('操作请求失败');
    } finally {
      setActionLoading((prev) => ({ ...prev, [field]: false }));
    }
  }, []);

  const handleRescrape = useCallback(async (productId: number) => {
    setActionLoading((prev) => ({ ...prev, _rescrape: true }));
    try {
      const res = await apiClient.post(`/v1/candidates/${productId}/rescrape`);
      if (res.code === 0 && res.data) {
        message.success('已尝试重新采集');
        setDetailProduct(res.data as unknown as CandidateProduct);
        setDetailMissingFields((res.data as any).missing_fields || []);
      } else {
        message.error(res.message || '重新采集失败');
      }
    } catch {
      message.error('重新采集请求失败');
    } finally {
      setActionLoading((prev) => ({ ...prev, _rescrape: false }));
    }
  }, []);

  const handleStartFill = (field: string) => {
    setFillingField(field);
    setFillValues((prev) => ({ ...prev, [field]: '' }));
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
      title: '完整度',
      dataIndex: 'completeness_status',
      width: 100,
      render: (s: string) =>
        s ? (
          <Tag color={completenessColorMap[s] || 'default'}>
            {completenessLabelMap[s] || s}
          </Tag>
        ) : (
          <Tag>未检查</Tag>
        ),
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
      <Card
        size="small"
        styles={{ body: { padding: '8px 20px', display: 'flex', alignItems: 'center', gap: 12 } }}
        style={{ marginBottom: 'var(--space-sm)' }}
      >
        <div style={{ flex: 1 }} />
        <Select
          allowClear
          placeholder="按完整度筛选"
          style={{ width: 160 }}
          value={completenessFilter || undefined}
          onChange={(val) => {
            setCompletenessFilter(val || '');
            setPage(1);
          }}
          options={[
            { value: 'incomplete', label: '不完整' },
            { value: 'needs_review', label: '待补充' },
            { value: 'research_ready', label: '可调研' },
            { value: 'listing_ready', label: '可上架' },
          ]}
        />
      </Card>
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

              {/* Missing fields with action buttons */}
              {detailMissingFields.length > 0 && (
                <Card
                  size="small"
                  type="inner"
                  title="缺失字段补全"
                  style={{ marginBottom: 'var(--space-md)' }}
                >
                  {detailMissingFields.map((field) => (
                    <div
                      key={field}
                      style={{
                        marginBottom: 'var(--space-sm)',
                        padding: 'var(--space-sm)',
                        background: 'var(--bg-component)',
                        borderRadius: 6,
                      }}
                    >
                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 8,
                          flexWrap: 'wrap',
                        }}
                      >
                        <Tag color="error" style={{ marginRight: 4 }}>
                          {FIELD_LABELS[field] || field}
                        </Tag>
                        <Text type="secondary" style={{ fontSize: '0.78rem', flex: 1 }}>
                          {SUGGESTIONS[FIELD_LABELS[field] || field] || '请补充此项信息'}
                        </Text>

                        {/* Rescrape button */}
                        <Tooltip title="重新从来源采集此字段">
                          <Button
                            size="small"
                            type="link"
                            icon={<ReloadOutlined />}
                            loading={actionLoading[field] || actionLoading['_rescrape']}
                            onClick={() => handleRescrape(detailProduct.id)}
                          >
                            重新采集
                          </Button>
                        </Tooltip>

                        {/* Manual fill button / inline editor */}
                        {fillingField === field ? (
                          <Space size="small">
                            {FIELD_TYPES[field] === 'number' ? (
                              <InputNumber
                                size="small"
                                style={{ width: 140 }}
                                value={fillValues[field] as unknown as number}
                                onChange={(v) =>
                                  setFillValues((prev) => ({
                                    ...prev,
                                    [field]: v != null ? String(v) : '',
                                  }))
                                }
                                onPressEnter={() => handleFillField(detailProduct.id, field)}
                              />
                            ) : (
                              <Input
                                size="small"
                                style={{ width: 160 }}
                                value={fillValues[field]}
                                placeholder={`输入${FIELD_LABELS[field] || field}`}
                                onChange={(e) =>
                                  setFillValues((prev) => ({
                                    ...prev,
                                    [field]: e.target.value,
                                  }))
                                }
                                onPressEnter={() => handleFillField(detailProduct.id, field)}
                              />
                            )}
                            <Button
                              size="small"
                              type="primary"
                              icon={<CheckOutlined />}
                              loading={actionLoading[field]}
                              onClick={() => handleFillField(detailProduct.id, field)}
                            />
                            <Button
                              size="small"
                              icon={<CloseOutlined />}
                              onClick={() => {
                                setFillingField(null);
                                setFillValues((prev) => ({ ...prev, [field]: '' }));
                              }}
                            />
                          </Space>
                        ) : (
                          <Button
                            size="small"
                            type="link"
                            icon={<EditOutlined />}
                            onClick={() => handleStartFill(field)}
                          >
                            补录
                          </Button>
                        )}

                        {/* Skip button */}
                        <Tooltip title="标记为此字段无法补齐，不再提示">
                          <Button
                            size="small"
                            type="link"
                            danger
                            icon={<StopOutlined />}
                            loading={actionLoading[field]}
                            onClick={() => handleSkipField(detailProduct.id, field)}
                          >
                            无法补齐
                          </Button>
                        </Tooltip>
                      </div>
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

              {/* Next step suggestion */}
              <Card size="small" type="inner" title="下一步建议" style={{ marginTop: 'var(--space-md)' }}>
                <Text>
                  {completenessHintMap[detailProduct.completeness_status] ||
                    (completenessResult.score >= 80
                      ? '所有信息完整，可以准备上架草稿。'
                      : completenessResult.score >= 60
                        ? '信息基本完整，可以执行利润分析和选品调研。'
                        : completenessResult.score >= 40
                          ? '已有关键信息，补充供应商和包装信息后可进入调研。'
                          : '缺少核心信息（标题、采购价、主图），补充后才能继续。')}
                </Text>
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
