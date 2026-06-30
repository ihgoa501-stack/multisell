'use client';

import { useState } from 'react';
import { Card, DatePicker, Descriptions, Empty, Select, Spin, Statistic, Table, Tabs } from 'antd';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';

const { RangePicker } = DatePicker;

type TabKey = 'sales' | 'profit' | 'inventory' | 'settlement' | 'platform-fee';

const TABS: { key: TabKey; label: string; path: string; hasRange: boolean }[] = [
  { key: 'sales', label: '销售报表', path: '/v1/report/sales', hasRange: true },
  { key: 'profit', label: '利润报表', path: '/v1/report/profit', hasRange: true },
  { key: 'inventory', label: '库存报表', path: '/v1/report/inventory', hasRange: false },
  { key: 'settlement', label: '结算报表', path: '/v1/report/settlement', hasRange: true },
  { key: 'platform-fee', label: '平台费用', path: '/v1/report/platform-fee', hasRange: true },
];

interface SummaryData {
  summary?: Record<string, number | string>;
  rows?: Record<string, unknown>[];
}

interface ProfitSummaryResult {
  total_profit: number;
  avg_margin: number;
  loss_sku_count: number;
  period_count: number;
  total_revenue: number;
  total_cost: number;
}

interface ProfitRankingItem {
  id: number;
  order_id: number;
  sku_id: number;
  revenue: number;
  net_profit: number;
  profit_margin: number;
  platform_fee: number;
  logistics_fee: number;
  advertising_cost: number;
  purchase_cost: number;
  other_cost: number;
  calculated_at: string;
  period_start: string;
  period_end: string;
}

function formatCurrency(v: number): string {
  return `¥${v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

function formatPercent(v: number): string {
  return `${(v * 100).toFixed(1)}%`;
}

const PROFIT_RANKING_COLUMNS = [
  { title: 'SKU ID', dataIndex: 'sku_id', render: (v: number) => String(v) },
  { title: '收入', dataIndex: 'revenue', render: (v: number) => formatCurrency(v) },
  { title: '利润', dataIndex: 'net_profit', render: (v: number) => formatCurrency(v) },
  { title: '利润率', dataIndex: 'profit_margin', render: (v: number) => formatPercent(v) },
];

export default function ReportsPage() {
  const [activeTab, setActiveTab] = useState<TabKey>('sales');
  const [range, setRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(29, 'day'),
    dayjs(),
  ]);
  const [platformId, setPlatformId] = useState<string>('');

  const current = TABS.find((t) => t.key === activeTab)!;

  // Generic report query for non-profit tabs
  const { data, isLoading: genericLoading } = useQuery({
    queryKey: ['report', activeTab, range?.[0]?.format('YYYY-MM-DD'), range?.[1]?.format('YYYY-MM-DD'), platformId],
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (current.hasRange && range) {
        params.from = range[0].format('YYYY-MM-DD');
        params.to = range[1].format('YYYY-MM-DD');
      }
      if (platformId) params.platform_id = platformId;
      const res = await apiClient.get<SummaryData>(current.path, params);
      return res.data;
    },
    enabled: activeTab !== 'profit',
    retry: false,
  });

  // Profit summary query
  const { data: profitSummary, isLoading: profitSummaryLoading } = useQuery({
    queryKey: ['profit-summary', range?.[0]?.format('YYYY-MM-DD'), range?.[1]?.format('YYYY-MM-DD')],
    queryFn: async () => {
      const params: Record<string, string> = {
        since: range[0].format('YYYY-MM-DD'),
        until: range[1].format('YYYY-MM-DD'),
      };
      const res = await apiClient.get<ProfitSummaryResult>('/v1/finance/profit/summary', params);
      return res.data;
    },
    enabled: activeTab === 'profit',
    retry: false,
  });

  // Profit ranking query
  const { data: profitRanking, isLoading: rankingLoading } = useQuery({
    queryKey: ['profit-ranking', range?.[0]?.format('YYYY-MM-DD')],
    queryFn: async () => {
      const params: Record<string, string> = {
        since: range[0].format('YYYY-MM-DD'),
        limit: '10',
      };
      const res = await apiClient.get<ProfitRankingItem[]>('/v1/finance/profit/ranking', params);
      return res.data;
    },
    enabled: activeTab === 'profit',
    retry: false,
  });

  const summaryEntries = Object.entries(data?.summary ?? []);
  const rows = data?.rows ?? [];

  const isProfitTab = activeTab === 'profit';
  const isProfitLoading = profitSummaryLoading || rankingLoading;

  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', color: 'var(--t1)' }}>报表</h1>

      <Tabs
        activeKey={activeTab}
        onChange={(k) => setActiveTab(k as TabKey)}
        items={TABS.map((t) => ({ key: t.key, label: t.label }))}
      />

      <Card style={{ marginBottom: 16, background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'center' }}>
          {current.hasRange && (
            <RangePicker
              value={range}
              onChange={(v) => v && setRange(v as [dayjs.Dayjs, dayjs.Dayjs])}
            />
          )}
          <Select
            placeholder="选择平台（可选）"
            allowClear
            style={{ width: 200 }}
            value={platformId || undefined}
            onChange={(v) => setPlatformId(v ?? '')}
            options={[
              { label: '全部', value: '' },
              { label: '淘宝', value: 'taobao' },
              { label: '京东', value: 'jd' },
              { label: '拼多多', value: 'pdd' },
              { label: '抖音', value: 'douyin' },
            ]}
          />
        </div>
      </Card>

      {isProfitTab ? (
        isProfitLoading ? (
          <Card style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <div style={{ textAlign: 'center', padding: 48 }}>
              <Spin tip="利润报表加载中..." />
            </div>
          </Card>
        ) : (
          <>
            {/* 利润汇总 */}
            <Card
              title="利润汇总"
              style={{ marginBottom: 16, background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}
            >
              {!profitSummary ? (
                <Empty description="暂无汇总数据" />
              ) : (
                <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap' }}>
                  <Statistic title="总收入" value={profitSummary.total_revenue} prefix="¥" precision={2} />
                  <Statistic title="总成本" value={profitSummary.total_cost} prefix="¥" precision={2} />
                  <Statistic
                    title="毛利润"
                    value={profitSummary.total_profit}
                    prefix="¥"
                    precision={2}
                    valueStyle={{ color: profitSummary.total_profit >= 0 ? '#3f8600' : '#cf1322' }}
                  />
                  <Statistic
                    title="毛利率"
                    value={profitSummary.avg_margin * 100}
                    suffix="%"
                    precision={1}
                    valueStyle={{ color: profitSummary.avg_margin >= 0 ? '#3f8600' : '#cf1322' }}
                  />
                </div>
              )}
            </Card>

            {/* SKU利润排名 */}
            <Card
              title="SKU利润排名"
              style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}
            >
              {!profitRanking || profitRanking.length === 0 ? (
                <Empty description="暂无SKU利润排名数据" />
              ) : (
                <Table
                  rowKey="id"
                  dataSource={profitRanking}
                  pagination={false}
                  columns={PROFIT_RANKING_COLUMNS}
                />
              )}
            </Card>
          </>
        )
      ) : (
        <>
          {genericLoading ? (
            <Card style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
              <div style={{ textAlign: 'center', padding: 48 }}>
                <Spin tip="报表加载中..." />
              </div>
            </Card>
          ) : (
            <>
              <Card title="汇总" style={{ marginBottom: 16, background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
                {summaryEntries.length === 0 ? (
                  <Empty description="暂无汇总数据" />
                ) : (
                  <Descriptions column={3} bordered size="small">
                    {summaryEntries.map(([k, v]) => (
                      <Descriptions.Item key={k} label={k}>
                        {String(v)}
                      </Descriptions.Item>
                    ))}
                  </Descriptions>
                )}
              </Card>

              <Card title="明细" style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
                {rows.length === 0 ? (
                  <Empty description="暂无明细数据" />
                ) : (
                  <Table
                    rowKey={(r, i) => String(r.id ?? i)}
                    dataSource={rows}
                    pagination={{ pageSize: 10 }}
                    scroll={{ x: 'max-content' }}
                    columns={Object.keys(rows[0] ?? {}).map((key) => ({
                      title: key,
                      dataIndex: key,
                      render: (v: unknown) => (v === null || v === undefined ? '-' : String(v)),
                    }))}
                  />
                )}
              </Card>
            </>
          )}
        </>
      )}
    </div>
  );
}
