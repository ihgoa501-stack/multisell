'use client';

import { useState, useEffect } from 'react';
import {
  Card, Table, Input, Button, Space, Tag, Spin, message, Badge,
  Select, Typography, Row, Col, Divider, Tooltip, Alert,
} from 'antd';
import {
  SearchOutlined, LinkOutlined, BulbOutlined, ThunderboltOutlined,
  ExperimentOutlined, ArrowRightOutlined,
  DatabaseOutlined,
} from '@ant-design/icons';
import apiClient from '@/lib/api-client';

const { Text } = Typography;

// ── Recommendation (existing) ─────────────────────────────────────────────

interface Recommendation {
  id: number;
  source_url: string;
  title: string;
  supplier_name: string;
  price: number;
  score: number;
  status: string;
  product_id_1688: string;
  image_url: string;
  recommend_reason: string;
  created_at: string;
}

interface PageData {
  code: number;
  message: string;
  data: Recommendation[];
  total: number;
  page: number;
  size: number;
}

const statusColorMap: Record<string, string> = {
  recommended: 'green', pending: 'gold', low_quality: 'red',
  imported: 'blue', rejected: 'default',
};
const statusLabelMap: Record<string, string> = {
  recommended: '推荐', pending: '待处理', low_quality: '低质量',
  imported: '已导入', rejected: '已拒绝',
};
const scoreColor = (score: number): string => {
  if (score >= 7) return 'green';
  if (score >= 4) return 'gold';
  return 'red';
};

// ── Research types (Phase 2) ──────────────────────────────────────────────

interface Direction {
  name: string;
  why: string;
  target_price_band: string;
  risk_notes?: string[];
  keywords?: string[];
  data_needed?: string[];
  confidence: number;
}

interface ResearchResult {
  status: string;
  category: string;
  target_market: string;
  target_platform: string;
  constraints_used?: string;
  recommended_directions: Direction[];
  data_needed: string[];
  warnings?: string[];
}

const MARKET_OPTIONS = [
  { value: 'US', label: '🇺🇸 美国' },
  { value: 'RU', label: '🇷🇺 俄罗斯' },
  { value: 'JP', label: '🇯🇵 日本' },
  { value: 'EU', label: '🇪🇺 欧洲' },
  { value: 'BR', label: '🇧🇷 巴西' },
  { value: 'SEA', label: '🌏 东南亚' },
];

const PLATFORM_OPTIONS = [
  { value: 'Amazon', label: 'Amazon' },
  { value: 'Ozon', label: 'Ozon' },
  { value: 'Shopee', label: 'Shopee' },
  { value: 'Lazada', label: 'Lazada' },
  { value: 'Wildberries', label: 'Wildberries' },
  { value: 'eBay', label: 'eBay' },
];

// ── Confidence display ────────────────────────────────────────────────────

const confidenceColor = (c: number): string => {
  if (c >= 0.7) return 'green';
  if (c >= 0.4) return 'orange';
  return 'red';
};

// ── Page ──────────────────────────────────────────────────────────────────

export default function SourcingPage() {
  // Research state
  const [researching, setResearching] = useState(false);
  const [researchResult, setResearchResult] = useState<ResearchResult | null>(null);
  const [category, setCategory] = useState('');
  const [targetMarket, setTargetMarket] = useState<string>('US');
  const [targetPlatform, setTargetPlatform] = useState<string>('Amazon');

  // Sourcing state (existing)
  const [url, setUrl] = useState('');
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [data, setData] = useState<Recommendation[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // ── Research: call A1 agent ───────────────────────────────────────────

  const handleResearch = async () => {
    if (!category.trim()) {
      message.warning('请输入商品类目');
      return;
    }
    setResearching(true);
    setResearchResult(null);
    try {
      const res = await apiClient.post('/v1/ai/run', {
        agent_id: 'A1',
        decision_point: 'product_research',
        context: {
          category: category.trim(),
          target_market: targetMarket,
          target_platform: targetPlatform,
        },
      });
      if (res.code === 0 && res.data?.output) {
        setResearchResult(res.data.output as ResearchResult);
        message.success('调研完成');
      } else {
        message.error(res.message || '调研失败');
      }
    } catch {
      message.error('调研请求失败');
    } finally {
      setResearching(false);
    }
  };

  // ── Sourcing (existing) ────────────────────────────────────────────────

  const loadRecommendations = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get<PageData>('/v1/sourcing/recommendations', {
        page: String(page), size: String(pageSize),
      });
      if (res.data) {
        setData(res.data.data || []);
        setTotal(res.data.total || 0);
      }
    } catch {
      message.error('加载推荐列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRecommendations();
  }, [page, pageSize]);

  const handleFetch = async () => {
    if (!url.trim()) {
      message.warning('请输入 1688 商品链接');
      return;
    }
    setFetching(true);
    try {
      const res = await apiClient.post<{
        product_id: number; title: string; price: number; score: number; status: string;
      }>('/v1/sourcing/fetch', { url: url.trim() });
      if (res.code === 0) {
        message.success(`采集成功！评分：${res.data?.score}/10`);
        setUrl('');
        await loadRecommendations();
      } else {
        message.error(res.message || '采集失败');
      }
    } catch {
      message.error('采集请求失败，请检查链接是否正确');
    } finally {
      setFetching(false);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    {
      title: '商品标题', dataIndex: 'title', ellipsis: true,
      render: (title: string, record: Recommendation) => (
        <a href={record.source_url} target="_blank" rel="noopener noreferrer">
          {title || record.source_url.slice(0, 50) + '...'}
        </a>
      ),
    },
    { title: '供应商', dataIndex: 'supplier_name', width: 140, ellipsis: true },
    {
      title: '价格 (CNY)', dataIndex: 'price', width: 120,
      render: (price: number) => `¥${price.toFixed(2)}`,
    },
    {
      title: '评分', dataIndex: 'score', width: 80,
      render: (score: number) => (
        <Badge count={score} showZero color={scoreColor(score)} style={{ fontSize: 12, fontWeight: 600 }} overflowCount={10} />
      ),
    },
    {
      title: '状态', dataIndex: 'status', width: 100,
      render: (status: string) => (
        <Tag color={statusColorMap[status] || 'default'}>{statusLabelMap[status] || status}</Tag>
      ),
    },
    { title: '采集时间', dataIndex: 'created_at', width: 160 },
    {
      title: '操作', width: 80,
      render: (_: unknown, record: Recommendation) => (
        <Button type="link" size="small" icon={<LinkOutlined />} href={record.source_url} target="_blank">查看</Button>
      ),
    },
  ];

  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', color: 'var(--t1)', margin: '0 0 16px 0' }}>
        AI 选品
      </h1>

      {/* ══ Phase 2: 选品调研 ══ */}
      <Card
        size="small"
        title={
          <Space>
            <ExperimentOutlined />
            <span>选品调研</span>
          </Space>
        }
        style={{ marginBottom: 16 }}
        styles={{ body: { padding: '16px 20px' } }}
      >
        <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
          输入目标品类和市场，AI 自动生成调研方向、关键词建议和风险提示
        </Text>

        <Space.Compact style={{ width: '100%', marginBottom: 12 }}>
          <Input
            placeholder="商品类目（如 家居、宠物用品）"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            onPressEnter={handleResearch}
            prefix={<BulbOutlined />}
            size="large"
            style={{ width: '35%' }}
            disabled={researching}
          />
          <Select
            value={targetMarket}
            onChange={setTargetMarket}
            options={MARKET_OPTIONS}
            size="large"
            style={{ width: '20%' }}
            disabled={researching}
          />
          <Select
            value={targetPlatform}
            onChange={setTargetPlatform}
            options={PLATFORM_OPTIONS}
            size="large"
            style={{ width: '20%' }}
            disabled={researching}
          />
          <Button
            type="primary"
            icon={researching ? <Spin size="small" /> : <ThunderboltOutlined />}
            onClick={handleResearch}
            loading={researching}
            size="large"
            style={{ width: '25%' }}
          >
            开始调研
          </Button>
        </Space.Compact>

        {/* Research results */}
        {researching && (
          <div style={{ textAlign: 'center', padding: '24px 0' }}>
            <Spin tip="AI 正在分析市场…" />
          </div>
        )}

        {researchResult && !researching && (
          <>
            <Divider style={{ margin: '12px 0' }} />

            {/* Warnings */}
            {researchResult.warnings?.map((w, i) => (
              <Alert key={i} type="warning" message={w} style={{ marginBottom: 8, fontSize: 12 }} showIcon />
            ))}

            {/* Data needed badge */}
            <div style={{ marginBottom: 12 }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                <DatabaseOutlined /> 待采集数据：{researchResult.data_needed?.join('、') || '无'}
              </Text>
            </div>

            {/* Direction cards */}
            <Row gutter={[12, 12]}>
              {researchResult.recommended_directions?.map((dir, i) => (
                <Col key={i} xs={24} sm={12} lg={8}>
                  <Card
                    size="small"
                    title={
                      <Space>
                        <ArrowRightOutlined />
                        <Text strong>{dir.name}</Text>
                      </Space>
                    }
                    styles={{ body: { padding: '12px 16px' } }}
                  >
                    <Space direction="vertical" size={4} style={{ width: '100%' }}>
                      <Text style={{ fontSize: 13 }}>{dir.why}</Text>

                      <Space style={{ marginTop: 8 }}>
                        <Tag color="blue">售价 {dir.target_price_band}</Tag>
                        <Tooltip title="置信度">
                          <Tag color={confidenceColor(dir.confidence)}>
                            {Math.round(dir.confidence * 100)}%
                          </Tag>
                        </Tooltip>
                      </Space>

                      {dir.keywords && dir.keywords.length > 0 && (
                        <div>
                          <Text type="secondary" style={{ fontSize: 12 }}>关键词：</Text>
                          <Space size={4} wrap>
                            {dir.keywords.map((kw, j) => (
                              <Tag key={j} style={{ fontSize: 11 }}>{kw}</Tag>
                            ))}
                          </Space>
                        </div>
                      )}

                      {dir.risk_notes && dir.risk_notes.length > 0 && (
                        <div>
                          <Text type="warning" style={{ fontSize: 12 }}>风险提示：</Text>
                          <ul style={{ margin: '4px 0', paddingLeft: 16, fontSize: 12 }}>
                            {dir.risk_notes.map((rn, j) => (
                              <li key={j} style={{ color: 'var(--t3)' }}>{rn}</li>
                            ))}
                          </ul>
                        </div>
                      )}
                    </Space>
                  </Card>
                </Col>
              ))}
            </Row>

            {/* Next action */}
            <div style={{ marginTop: 16, padding: '12px', background: 'var(--bg2)', borderRadius: 6 }}>
              <Text type="secondary">
                <BulbOutlined /> 下一步：在下方「1688 采集」粘贴商品链接开始单品分析，或使用 Chrome 扩展采集搜索列表
              </Text>
            </div>
          </>
        )}
      </Card>

      {/* ══ 1688 采集 (existing) ══ */}
      <Card
        size="small"
        title={<Space><SearchOutlined /><span>1688 采集</span></Space>}
        style={{ marginBottom: 16 }}
        styles={{ body: { padding: '16px 20px' } }}
      >
        <Space.Compact style={{ width: '100%' }}>
          <Input
            placeholder="粘贴 1688 商品链接，例如 https://detail.1688.com/offer/xxx.html"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            onPressEnter={handleFetch}
            prefix={<LinkOutlined />}
            size="large"
            disabled={fetching}
          />
          <Button
            type="primary"
            icon={fetching ? <Spin size="small" /> : <SearchOutlined />}
            onClick={handleFetch}
            loading={fetching}
            size="large"
          >
            采集分析
          </Button>
        </Space.Compact>
      </Card>

      {/* ══ 推荐列表 (existing) ══ */}
      <Card
        size="small"
        title={`推荐列表 (${total})`}
        styles={{ body: { padding: 0 } }}
      >
        <Table<Recommendation>
          rowKey="id"
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={{
            current: page, pageSize, total, showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps); },
          }}
          scroll={{ x: 900 }}
        />
      </Card>
    </div>
  );
}
