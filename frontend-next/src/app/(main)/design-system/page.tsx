'use client';

import { Col, Row, Space, Tag, Typography, Divider } from 'antd';
import {
  ThunderboltOutlined, AlertOutlined, TeamOutlined, SafetyCertificateOutlined,
} from '@ant-design/icons';
import StatCard from '@/components/ui/StatCard';
import SectionCard from '@/components/ui/SectionCard';
import { CardSkeleton, TableSkeleton, StatRowSkeleton } from '@/components/ui/PageSkeleton';
import { ErrorBoundary } from '@/components/ui/ErrorBoundary';

const { Text } = Typography;

function Usage({ code, children }: { code: string; children?: React.ReactNode }) {
  return (
    <div style={{ marginBottom: 'var(--space-xl)' }}>
      <div style={{ marginBottom: 'var(--space-lg)' }}>{children}</div>
      <pre style={{
        background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 6,
        padding: 'var(--space-md)', fontSize: '0.72rem', overflow: 'auto',
        fontFamily: 'var(--mono)', color: 'var(--t2)',
      }}>{code}</pre>
    </div>
  );
}

export default function DesignSystemPage() {
  return (
    <div style={{ padding: 'var(--space-lg) var(--space-xl)', background: 'var(--bg)', minHeight: '100%' }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', color: 'var(--t1)', marginBottom: 'var(--space-xl)' }}>
        设计系统 · 组件目录
      </h1>
      <Text type="secondary" style={{ display: 'block', marginBottom: 'var(--space-xl)' }}>
        所有新页面应优先使用以下组件。每新增一个组件先在这里登记。
      </Text>

      {/* ── StatCard ── */}
      <SectionCard title={<Space><ThunderboltOutlined /> StatCard — 统计卡片</Space>}>
        <Text style={{ display: 'block', color: 'var(--t3)', marginBottom: 'var(--space-lg)' }}>
          替代手动 div + Statistic 模式。支持 prefix（图标/符号）、valueStyle（着色）、loading、trend。
        </Text>
        <Row gutter={16}>
          <Col xs={12} md={6}>
            <StatCard title="今日订单" value={128} prefix={<ThunderboltOutlined />} />
          </Col>
          <Col xs={12} md={6}>
            <StatCard title="异常告警" value={3} prefix={<AlertOutlined />} valueStyle={{ color: 'var(--r4)' }} />
          </Col>
          <Col xs={12} md={6}>
            <StatCard title="活跃 Agent" value={12} prefix={<TeamOutlined />} loading />
          </Col>
          <Col xs={12} md={6}>
            <StatCard title="平均置信度" value="87%" prefix={<SafetyCertificateOutlined />} valueStyle={{ color: 'var(--g4)' }} />
          </Col>
        </Row>
        <Usage code={'<StatCard title="今日订单" value={128}\n  prefix={<ThunderboltOutlined />} />\n\n<StatCard title="异常告警" value={3}\n  prefix={<AlertOutlined />}\n  valueStyle={{ color: \'var(--r4)\' }} />\n\n<StatCard title="活跃 Agent" value={12}\n  prefix={<TeamOutlined />} loading />\n\n<StatCard title="平均置信度" value="87%"\n  prefix={<SafetyCertificateOutlined />}\n  valueStyle={{ color: \'var(--g4)\' }} />'} />
      </SectionCard>

      {/* ── SectionCard ── */}
      <SectionCard title={<Space><SafetyCertificateOutlined /> SectionCard — 区域容器</Space>} style={{ marginTop: 'var(--space-lg)' }}>
        <Text style={{ display: 'block', color: 'var(--t3)', marginBottom: 'var(--space-lg)' }}>
          替代手动 div + header + content 模式。支持 title（字符串或 JSX）、actions、noPadding。
        </Text>
        <Row gutter={16}>
          <Col span={12}>
            <SectionCard title="基本信息">
              <Text>带标题和 padding 的内容区域。</Text>
            </SectionCard>
          </Col>
          <Col span={12}>
            <SectionCard title={<Space>标题含图标 <Tag color="blue">标签</Tag></Space>}>
              <Text>title 支持 ReactNode，可嵌入图标和 Tag。</Text>
            </SectionCard>
          </Col>
        </Row>
        <div style={{ marginTop: 'var(--space-md)' }}>
          <SectionCard title="noPadding 模式" noPadding>
            <div style={{ padding: 'var(--space-lg)' }}>
              设置 noPadding 后自行控制内部 padding（用于内嵌全宽 Table 等）。
            </div>
          </SectionCard>
        </div>
        <Usage code={'<SectionCard title="基本信息">\n  <Text>内容</Text>\n</SectionCard>\n\n<SectionCard title={<Space>标题 <Tag>标签</Tag></Space>}>\n  <Text>内容</Text>\n</SectionCard>\n\n<SectionCard title="无 padding" noPadding>\n  <Table ... />\n</SectionCard>'} />
      </SectionCard>

      {/* ── PageContainer ── */}
      <SectionCard title="PageContainer — 页面容器" style={{ marginTop: 'var(--space-lg)' }}>
        <Text style={{ display: 'block', color: 'var(--t3)', marginBottom: 'var(--space-lg)' }}>
          统一页面标题、padding、loading/empty/error 状态。新建页面应优先使用。
        </Text>
        <Usage code={'<PageContainer title="商品详情" subtitle="ID: 123"\n  extra={<Button>操作</Button>}\n  loading={isLoading}\n  error={isError}\n  onRetry={refetch}\n  empty={isEmpty}\n  emptyDesc="暂无商品">\n  <Table ... />\n</PageContainer>'} />
      </SectionCard>

      {/* ── Skeleton ── */}
      <SectionCard title={<Space><AlertOutlined /> PageSkeleton — 骨架屏</Space>} style={{ marginTop: 'var(--space-lg)' }}>
        <Text style={{ display: 'block', color: 'var(--t3)', marginBottom: 'var(--space-lg)' }}>
          替代裸 Spin。三种变体：StatRowSkeleton（统计行）、CardSkeleton（卡片）、TableSkeleton（表格）。
        </Text>
        <StatRowSkeleton count={4} />
        <div style={{ marginTop: 'var(--space-lg)' }} />
        <CardSkeleton />
        <div style={{ marginTop: 'var(--space-lg)' }} />
        <TableSkeleton rows={3} cols={4} />
        <Usage code={'<StatRowSkeleton count={4} />   // 替代统计行 loading\n<CardSkeleton rows={2} />    // 替代卡片 loading\n<TableSkeleton rows={5} cols={4} /> // 替代表格 loading'} />
      </SectionCard>

      {/* ── ErrorBoundary ── */}
      <SectionCard title={<Space>⚠️ ErrorBoundary — 错误边界</Space>} style={{ marginTop: 'var(--space-lg)' }}>
        <Text style={{ display: 'block', color: 'var(--t3)', marginBottom: 'var(--space-lg)' }}>
          包裹页面或组件，捕获渲染异常并显示友好错误信息 + 重试按钮。
        </Text>
        <div style={{
          background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8,
          padding: 'var(--space-lg)', marginBottom: 'var(--space-md)',
        }}>
          <ErrorBoundary>
            <Text>正常内容区域 — ErrorBoundary 不干扰正常运行。</Text>
          </ErrorBoundary>
        </div>
        <Usage code={'<ErrorBoundary>\n  <YourComponent />\n</ErrorBoundary>\n\n<ErrorBoundary fallback={<CustomFallback />}>\n  <YourComponent />\n</ErrorBoundary>'} />
      </SectionCard>

      {/* ── 设计原则 ── */}
      <Divider />

      <SectionCard title="设计规范摘要">
        <Space direction="vertical" style={{ width: '100%' }} size="small">
          <div><Tag color="blue">规则 1</Tag><Text> 颜色必须用 CSS 变量 <Tag>var(--*)</Tag>，禁止十六进制</Text></div>
          <div><Tag color="blue">规则 2</Tag><Text> 间距必须用 <Tag>var(--space-*)</Tag> scale（8px 基准）</Text></div>
          <div><Tag color="blue">规则 3</Tag><Text> 加载态用 <Tag>PageSkeleton</Tag> 组件，禁止裸 Spin</Text></div>
          <div><Tag color="blue">规则 4</Tag><Text> 页面布局用 <Tag>PageContainer</Tag>，手动写 div 前先确认不能用它</Text></div>
          <div><Tag color="blue">规则 5</Tag><Text> 卡片容器用 <Tag>SectionCard</Tag>，统计数字用 <Tag>StatCard</Tag></Text></div>
        </Space>
      </SectionCard>
    </div>
  );
}
