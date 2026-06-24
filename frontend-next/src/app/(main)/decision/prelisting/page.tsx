'use client';

import { useState } from 'react';
import { Card, Form, InputNumber, Select, Button, Table, Row, Col, Space, Tag, Typography, Divider, Alert, message } from 'antd';
import { ArrowLeftOutlined, CalculatorOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';

const { Title, Text, Paragraph } = Typography;

interface SimulationResult {
  platformId: number;
  platformName: string;
  skuId: number;
  countryCode: string;
  revenue: number;
  productCost: number;
  shippingCost: number;
  platformFee: number;
  paymentFee: number;
  otherFee: number;
  estimatedProfit: number;
  profitMargin: number;
  confidenceScore: number;
  riskLevel: string;
  recommendation: string;
  reasoning: string;
}

const PLATFORM_OPTIONS = [
  { label: 'Amazon US (ID: 1)', value: 1 },
  { label: 'Shopee SG (ID: 2)', value: 2 },
  { label: 'Lazada MY (ID: 3)', value: 3 },
  { label: 'TikTok Shop ID (ID: 4)', value: 4 },
];

const COUNTRY_OPTIONS = [
  { label: '美国 (US)', value: 'US' },
  { label: '新加坡 (SG)', value: 'SG' },
  { label: '马来西亚 (MY)', value: 'MY' },
  { label: '印度尼西亚 (ID)', value: 'ID' },
];

export default function PrelistingWorkbench() {
  const router = useRouter();
  const [form] = Form.useForm();
  const [results, setResults] = useState<SimulationResult[]>([]);

  // Simulation logic inside the client (fallback if no direct simulation API,
  // but it also saves to the database on request)
  const handleSimulate = async () => {
    try {
      const values = await form.validateFields();
      const skuId = values.sku_id;
      const targetPlatforms = values.platforms as number[];
      const countryCode = values.country_code;
      const revenue = values.estimated_revenue || 100;
      const productCost = values.estimated_product_cost || 30;

      // Simulate costs based on typical platform rules
      const simulated: SimulationResult[] = targetPlatforms.map((pid) => {
        const platformName = PLATFORM_OPTIONS.find((o) => o.value === pid)?.label.split(' ')[0] || `Platform ${pid}`;

        // Fee rates vary by platform
        let shippingRate = 0.15; // 15% of revenue
        let platformRate = 0.08; // 8% fee
        let paymentRate = 0.03;  // 3% payment fee

        if (pid === 1) { // Amazon US
          shippingRate = 0.18;
          platformRate = 0.15;
          paymentRate = 0.02;
        } else if (pid === 2) { // Shopee
          shippingRate = 0.12;
          platformRate = 0.06;
          paymentRate = 0.03;
        } else if (pid === 3) { // Lazada
          shippingRate = 0.10;
          platformRate = 0.07;
          paymentRate = 0.03;
        } else if (pid === 4) { // TikTok
          shippingRate = 0.14;
          platformRate = 0.05;
          paymentRate = 0.02;
        }

        const shippingCost = Number((revenue * shippingRate).toFixed(2));
        const platformFee = Number((revenue * platformRate).toFixed(2));
        const paymentFee = Number((revenue * paymentRate).toFixed(2));
        const otherFee = Number((revenue * 0.02).toFixed(2)); // 2% buffer

        const costTotal = productCost + shippingCost + platformFee + paymentFee + otherFee;
        const estimatedProfit = Number((revenue - costTotal).toFixed(2));
        const profitMargin = Number(((estimatedProfit / revenue) * 100).toFixed(2));

        let riskLevel = 'medium';
        let recommendation = '建议上架';
        if (profitMargin < 10) {
          riskLevel = 'high';
          recommendation = '利润过低，谨慎上架';
        } else if (profitMargin >= 25) {
          riskLevel = 'low';
          recommendation = '极力推荐上架';
        }

        return {
          platformId: pid,
          platformName,
          skuId,
          countryCode,
          revenue,
          productCost,
          shippingCost,
          platformFee,
          paymentFee,
          otherFee,
          estimatedProfit,
          profitMargin,
          confidenceScore: 0.92,
          riskLevel,
          recommendation,
          reasoning: `基于 SKU ${skuId} 的生产成本与各平台费率模型得出。预估利润率 ${profitMargin}%，推荐结论：${recommendation}。`,
        };
      });

      setResults(simulated);
      message.success('预估决策试算完成');
    } catch (e) {
      // Validate failed
    }
  };

  // Mutation to persist a decision and auto-approve/reject
  const submitDecisionMutation = useMutation({
    mutationFn: async ({ result, action }: { result: SimulationResult; action: 'approved' | 'rejected' }) => {
      // 1. Create the decision via POST /v1/decision
      const res = await apiClient.post<any>('/v1/decision', {
        sku_id: result.skuId,
        platform_id: result.platformId,
        country_code: result.countryCode,
        decision_point: 'pre_listing',
        estimated_revenue: result.revenue,
        estimated_product_cost: result.productCost,
        estimated_shipping_cost: result.shippingCost,
        estimated_platform_fee: result.platformFee,
        estimated_payment_fee: result.paymentFee,
        estimated_other_fee: result.otherFee,
        estimated_profit: result.estimatedProfit,
        profit_margin: result.profitMargin,
        confidence_score: result.confidenceScore,
        risk_level: result.riskLevel,
        recommendation: result.recommendation,
        reasoning: result.reasoning,
        status: 'pending',
      });

      const decisionId = res.data?.id;
      if (!decisionId) {
        throw new Error('Failed to create decision record');
      }

      // 2. Call the approve or reject API
      if (action === 'approved') {
        await apiClient.post(`/v1/decision/${decisionId}/approve`, {
          decided_by: 'admin',
        });
      } else {
        await apiClient.post(`/v1/decision/${decisionId}/reject`, {
          decided_by: 'admin',
          reason: '人工工作台驳回',
        });
      }
      return { platformName: result.platformName, action };
    },
    onSuccess: (data) => {
      message.success(`${data.platformName} 决策已提交并标记为 [${data.action.toUpperCase()}]`);
    },
    onError: (err: any) => {
      message.error(`提交决策失败: ${err.message}`);
    },
  });

  const columns = [
    { title: '平台', dataIndex: 'platformName', key: 'platformName' },
    {
      title: '预计售价',
      dataIndex: 'revenue',
      key: 'revenue',
      render: (v: number) => `¥${v.toFixed(2)}`,
    },
    {
      title: '生产成本',
      dataIndex: 'productCost',
      key: 'productCost',
      render: (v: number) => `¥${v.toFixed(2)}`,
    },
    {
      title: '跨境运费',
      dataIndex: 'shippingCost',
      key: 'shippingCost',
      render: (v: number) => `¥${v.toFixed(2)}`,
    },
    {
      title: '平台佣金',
      dataIndex: 'platformFee',
      key: 'platformFee',
      render: (v: number) => `¥${v.toFixed(2)}`,
    },
    {
      title: '支付手续费',
      dataIndex: 'paymentFee',
      key: 'paymentFee',
      render: (v: number) => `¥${v.toFixed(2)}`,
    },
    {
      title: '预估净利润',
      dataIndex: 'estimatedProfit',
      key: 'estimatedProfit',
      render: (v: number, r: SimulationResult) => (
        <span style={{ color: v >= 0 ? '#52c41a' : '#ff4d4f', fontWeight: 'bold' }}>
          ¥{v.toFixed(2)} ({r.profitMargin}%)
        </span>
      ),
    },
    {
      title: '风险等级',
      dataIndex: 'riskLevel',
      key: 'riskLevel',
      render: (level: string) => {
        const colors: Record<string, string> = { low: 'green', medium: 'orange', high: 'red' };
        return <Tag color={colors[level] || 'blue'}>{level.toUpperCase()}</Tag>;
      },
    },
    { title: '决策建议', dataIndex: 'recommendation', key: 'recommendation' },
    {
      title: '决策操作',
      key: 'action',
      render: (_: any, record: SimulationResult) => (
        <Space>
          <Button
            type="primary"
            size="small"
            icon={<CheckCircleOutlined />}
            loading={submitDecisionMutation.isPending}
            onClick={() => submitDecisionMutation.mutate({ result: record, action: 'approved' })}
            style={{ backgroundColor: '#52c41a', borderColor: '#52c41a' }}
          >
            批准上架
          </Button>
          <Button
            type="primary"
            danger
            size="small"
            icon={<CloseCircleOutlined />}
            loading={submitDecisionMutation.isPending}
            onClick={() => submitDecisionMutation.mutate({ result: record, action: 'rejected' })}
          >
            驳回
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer title="上架前决策试算工作台">
      <Button
        icon={<ArrowLeftOutlined />}
        onClick={() => router.push('/decision')}
        style={{ marginBottom: 16 }}
      >
        返回决策列表
      </Button>

      <Row gutter={24}>
        <Col span={8}>
          <Card title="试算输入">
            <Form form={form} layout="vertical">
              <Form.Item
                name="sku_id"
                label="SKU ID"
                rules={[{ required: true, message: '请输入或选择 SKU ID' }]}
                initialValue={1}
              >
                <InputNumber style={{ width: '100%' }} placeholder="输入 SKU ID" min={1} />
              </Form.Item>

              <Form.Item
                name="platforms"
                label="目标平台"
                rules={[{ required: true, message: '请选择至少一个目标平台' }]}
                initialValue={[1, 2]}
              >
                <Select mode="multiple" placeholder="选择平台" options={PLATFORM_OPTIONS} />
              </Form.Item>

              <Form.Item
                name="country_code"
                label="目标国家"
                rules={[{ required: true, message: '请选择目标国家' }]}
                initialValue="US"
              >
                <Select options={COUNTRY_OPTIONS} />
              </Form.Item>

              <Form.Item
                name="estimated_revenue"
                label="预估外币售价 (换算为 CNY)"
                rules={[{ required: true, message: '请输入预估售价' }]}
                initialValue={150}
              >
                <InputNumber style={{ width: '100%' }} min={0.01} precision={2} />
              </Form.Item>

              <Form.Item
                name="estimated_product_cost"
                label="商品采购/生产成本 (CNY)"
                rules={[{ required: true, message: '请输入采购成本' }]}
                initialValue={40}
              >
                <InputNumber style={{ width: '100%' }} min={0} precision={2} />
              </Form.Item>

              <Form.Item>
                <Button
                  type="primary"
                  icon={<CalculatorOutlined />}
                  onClick={handleSimulate}
                  style={{ width: '100%' }}
                >
                  开始试算分析
                </Button>
              </Form.Item>
            </Form>
          </Card>
        </Col>

        <Col span={16}>
          <Card title="多平台收益试算对比">
            {results.length === 0 ? (
              <Alert
                message="请在左侧填写试算表单并点击开始试算分析按钮"
                type="info"
                showIcon
              />
            ) : (
              <Space direction="vertical" size="large" style={{ width: '100%' }}>
                <Table
                  dataSource={results}
                  columns={columns}
                  rowKey="platformId"
                  pagination={false}
                  size="small"
                />

                <Divider />

                <Title level={4}>详细试算解释</Title>
                {results.map((r) => (
                  <Card key={r.platformId} type="inner" title={r.platformName} style={{ marginBottom: 12 }}>
                    <Paragraph>
                      <strong>计算路径说明：</strong>
                      售价(¥{r.revenue}) - 生产成本(¥{r.productCost}) - 运费(¥{r.shippingCost}) - 平台费(¥{r.platformFee}) - 支付费(¥{r.paymentFee}) - 其它费(¥{r.otherFee}) = <strong>预估利润 ¥{r.estimatedProfit}</strong>.
                    </Paragraph>
                    <Paragraph>
                      <strong>推理依据：</strong> {r.reasoning}
                    </Paragraph>
                  </Card>
                ))}
              </Space>
            )}
          </Card>
        </Col>
      </Row>
    </PageContainer>
  );
}
