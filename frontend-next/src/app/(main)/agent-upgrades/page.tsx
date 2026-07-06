'use client';

import { Card, Collapse, Descriptions, Tag, Typography } from 'antd';
import PageContainer from '@/components/ui/PageContainer';

const { Text, Paragraph } = Typography;

export default function AgentUpgradesPage() {
  return (
    <PageContainer title="Agent 升级" subtitle="P3: 智能体升级 (#193-196)">
      <Collapse defaultActiveKey={['prism', 'adpilot', 'listing', 'support']} items={[
        {
          key: 'prism',
          label: <span>Prism 商品图生成 <Tag color="blue">#193</Tag></span>,
          children: (
            <Card size="small">
              <Descriptions column={1} size="small" bordered>
                <Descriptions.Item label="状态"><Tag color="green">已集成</Tag></Descriptions.Item>
                <Descriptions.Item label="触发端点">POST /api/v1/product-analysis/trigger-prism</Descriptions.Item>
                <Descriptions.Item label="Prism 服务">已配置，通过 prismadapter 客户端调用</Descriptions.Item>
                <Descriptions.Item label="生成参数">
                  <div>image_url: 商品原图URL</div>
                  <div>platform: 目标平台 (ozon/shopee/lazada/amazon)</div>
                  <div>product_id: 商品ID</div>
                </Descriptions.Item>
                <Descriptions.Item label="输出">
                  <div>job_id: 生成任务ID</div>
                  <div>output_url: 生成后图片URL</div>
                  <div>compliance_report: 合规检测报告</div>
                  <div>risk_score: 风险评估</div>
                </Descriptions.Item>
              </Descriptions>
            </Card>
          ),
        },
        {
          key: 'adpilot',
          label: <span>Ad Pilot Agent (A3) <Tag color="blue">#194</Tag></span>,
          children: (
            <Card size="small">
              <Paragraph>
                <Text strong>A3 Ad Pilot</Text> 新增决策点 - 广告策略、关键词出价、ACOS 优化、预算分配
              </Paragraph>
              <Descriptions column={1} size="small" bordered>
                <Descriptions.Item label="ad_strategy">
                  <div>根据产品生命周期阶段 (launch/growth/mature) 推荐广告策略</div>
                  <div>输出: 策略组合 + 预算分配比例</div>
                </Descriptions.Item>
                <Descriptions.Item label="keyword_bidding">
                  <div>为每个关键词推荐 CPC 出价</div>
                  <div>输出: suggested_bid + action (increase/reduce/maintain)</div>
                </Descriptions.Item>
                <Descriptions.Item label="acos_optimization">
                  <div>批量分析多个广告活动的 ACOS</div>
                  <div>输出: 每个活动的优化建议</div>
                </Descriptions.Item>
                <Descriptions.Item label="budget_allocation">
                  <div>基于各活动表现智能分配预算</div>
                  <div>输出: allocated_budget + 分配理由</div>
                </Descriptions.Item>
              </Descriptions>
            </Card>
          ),
        },
        {
          key: 'listing',
          label: <span>Listing Genius Agent (A2) <Tag color="blue">#195</Tag></span>,
          children: (
            <Card size="small">
              <Paragraph>
                <Text strong>A2 Listing Genius</Text> 新增决策点 - 标题优化、关键词优化、搜索趋势分析、多平台 SEO
              </Paragraph>
              <Descriptions column={1} size="small" bordered>
                <Descriptions.Item label="title_optimization">
                  <div>根据市场平台优化商品标题</div>
                  <div>考虑字符限制、关键词前置、品类关键词</div>
                </Descriptions.Item>
                <Descriptions.Item label="keyword_optimization">
                  <div>分析关键词的搜索量和竞争度</div>
                  <div>输出: 高/低优先级关键词分类</div>
                </Descriptions.Item>
                <Descriptions.Item label="search_trend_analysis">
                  <div>分析关键词搜索趋势方向</div>
                  <div>输出: trend_direction + 季节性高峰 + 推荐</div>
                </Descriptions.Item>
                <Descriptions.Item label="multi_platform_seo">
                  <div>为多平台生成 SEO 建议</div>
                  <div>支持: Amazon, Ozon, Shopee, Lazada, eBay, Walmart</div>
                </Descriptions.Item>
              </Descriptions>
            </Card>
          ),
        },
        {
          key: 'support',
          label: <span>Support Mate Agent (A4) <Tag color="blue">#196</Tag></span>,
          children: (
            <Card size="small">
              <Paragraph>
                <Text strong>A4 Support Mate</Text> 新增功能 - 多语言回复、工单系统
              </Paragraph>
              <Descriptions column={1} size="small" bordered>
                <Descriptions.Item label="multi_language_reply">
                  <div>多语言自动回复模板</div>
                  <div>支持: 中文/English/Русский/日本語/한국어/ภาษาไทย/Tiếng Việt/Español</div>
                </Descriptions.Item>
                <Descriptions.Item label="ticket_actions">
                  <div>工单管理系统</div>
                  <div>操作: create / update / escalate / resolve / close</div>
                  <div>优先级: low / medium / high / urgent</div>
                </Descriptions.Item>
              </Descriptions>
            </Card>
          ),
        },
      ]} />
    </PageContainer>
  );
}
