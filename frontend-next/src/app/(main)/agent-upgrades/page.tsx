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
          label: <span>Prism 商品图生成 <Tag>历史能力</Tag></span>,
          children: (
            <Card size="small">
              <Descriptions column={1} size="small" bordered>
                <Descriptions.Item label="状态"><Tag color="orange">已停止入口，迁移中</Tag></Descriptions.Item>
                <Descriptions.Item label="原因">旧入口接受任意图片 URL，已停止注册，避免服务器代抓取造成安全风险。</Descriptions.Item>
                <Descriptions.Item label="替代入口">商品图片工作室 /product-images</Descriptions.Item>
                <Descriptions.Item label="边界">历史记录保留只读；新任务统一由 Image Service 处理，图片处理结果不会自动批准或发布。</Descriptions.Item>
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
