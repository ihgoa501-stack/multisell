'use client';

import { useState } from 'react';
import { Alert, Card, DatePicker, Descriptions, Empty, Select, Spin, Statistic, Table, Tabs } from 'antd';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';

const { RangePicker } = DatePicker;

type TabKey = 'sales' | 'profit' | 'inventory' | 'settlement' | 'platform-fee' | 'daily' | 'weekly';

const TABS: { key: TabKey; label: string; path: string; hasRange: boolean }[] = [
  { key: 'sales', label: '销售报表', path: '/v1/report/sales', hasRange: true },
  { key: 'inventory', label: '库存报表', path: '/v1/report/inventory', hasRange: false },
  { key: 'platform-fee', label: '平台费用', path: '/v1/report/platform-fee', hasRange: true },
  { key: 'daily', label: '日报', path: '/v1/report/daily', hasRange: false },
  { key: 'weekly', label: '周报', path: '/v1/report/weekly', hasRange: false },
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

interface DailyReport {
  date: string;
  sales: number;
  orders: number;
  new_listings: number;
  anomalies: number;
  approvals: number;
  agent_proposals: number;
  llm_cost: number;
}

interface WeeklyReport {
  week_start: string;
  week_end: string;
  daily_reports: DailyReport[];
  sales_total: number;
  orders_total: number;
  anomalies_total: number;
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

const WEEKLY_DAILY_COLUMNS = [
  { title: '日期', dataIndex: 'date', key: 'date' },
  { title: '销售额', dataIndex: 'sales', key: 'sales', render: (v: number) => formatCurrency(v) },
  { title: '订单数', dataIndex: 'orders', key: 'orders' },
  { title: '上新数', dataIndex: 'new_listings', key: 'new_listings' },
  { title: '异常', dataIndex: 'anomalies', key: 'anomalies' },
  { title: '审批数', dataIndex: 'approvals', key: 'approvals' },
  { title: 'Agent提案', dataIndex: 'agent_proposals', key: 'agent_proposals' },
  { title: 'LLM费用($)', dataIndex: 'llm_cost', key: 'llm_cost', render: (v: number) => `$${v.toFixed(2)}` },
];

export default function ReportsPage() {
  const [activeTab, setActiveTab] = useState<TabKey>('sales');
  const [range, setRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(29, 'day'),
    dayjs(),
  ]);
  const [platformId, setPlatformId] = useState<string>('');
  const [dailyDate, setDailyDate] = useState<dayjs.Dayjs>(dayjs());
  const [weeklyStart, setWeeklyStart] = useState<dayjs.Dayjs>(dayjs().startOf('week').add(1, 'day')); // Monday

  const current = TABS.find((t) => t.key === activeTab)!;

  // Generic report query for non-profit, non-daily/weekly tabs
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
    enabled: activeTab !== 'profit' && activeTab !== 'daily' && activeTab !== 'weekly',
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

  // Daily report query
  const { data: dailyData, isLoading: dailyLoading } = useQuery({
    queryKey: ['report-daily', dailyDate?.format('YYYY-MM-DD')],
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (dailyDate) params.date = dailyDate.format('YYYY-MM-DD');
      const res = await apiClient.get<DailyReport>('/v1/report/daily', params);
      return res.data;
    },
    enabled: activeTab === 'daily',
    retry: false,
  });

  // Weekly report query
  const { data: weeklyData, isLoading: weeklyLoading } = useQuery({
    queryKey: ['report-weekly', weeklyStart?.format('YYYY-MM-DD')],
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (weeklyStart) params.week_start = weeklyStart.format('YYYY-MM-DD');
      const res = await apiClient.get<WeeklyReport>('/v1/report/weekly', params);
      return res.data;
    },
    enabled: activeTab === 'weekly',
    retry: false,
  });

  const summaryEntries = Object.entries(data?.summary ?? []);
  const rows = data?.rows ?? [];

  const isProfitTab = activeTab === 'profit';
  const isProfitLoading = profitSummaryLoading || rankingLoading;
  const isDailyTab = activeTab === 'daily';
  const isWeeklyTab = activeTab === 'weekly';
  const isGenericTab = !isProfitTab && !isDailyTab && !isWeeklyTab;

  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', color: 'var(--t1)' }}>报表</h1>

      <Alert
		type="warning"
		showIcon
		message="此页不是最终利润或现金对账依据"
		description="历史聚合只用于趋势参考。订单终局请使用小Q的“订单经营事实”以及权威利润、结算和财务页面。"
		style={{ marginBottom: 16 }}
	  />

      <Tabs
        activeKey={activeTab}
        onChange={(k) => setActiveTab(k as TabKey)}
        items={TABS.map((t) => ({ key: t.key, label: t.label }))}
      />

      <Card style={{ marginBottom: 16, background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'center' }}>
          {activeTab === 'daily' && (
            <DatePicker
              value={dailyDate}
              onChange={(v) => v && setDailyDate(v)}
              allowClear={false}
            />
          )}
          {activeTab === 'weekly' && (
            <DatePicker
              value={weeklyStart}
              onChange={(v) => v && setWeeklyStart(v)}
              allowClear={false}
              picker="week"
            />
          )}
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

      {isProfitTab && (
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
      )}

      {isDailyTab && (
        dailyLoading ? (
          <Card style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <div style={{ textAlign: 'center', padding: 48 }}>
              <Spin tip="日报加载中..." />
            </div>
          </Card>
        ) : !dailyData ? (
          <Card style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <Empty description="暂无日报数据" />
          </Card>
        ) : (
          <Card title={`日报 ${dailyData.date}`} style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap' }}>
              <Statistic title="销售额" value={dailyData.sales} prefix="¥" precision={2} />
              <Statistic title="订单数" value={dailyData.orders} />
              <Statistic title="上新数" value={dailyData.new_listings} />
              <Statistic title="异常" value={dailyData.anomalies}
                valueStyle={{ color: dailyData.anomalies > 0 ? '#cf1322' : undefined }} />
              <Statistic title="审批数" value={dailyData.approvals} />
              <Statistic title="Agent提案" value={dailyData.agent_proposals} />
              <Statistic title="LLM费用" value={dailyData.llm_cost} precision={2} prefix="$" />
            </div>
          </Card>
        )
      )}

      {isWeeklyTab && (
        weeklyLoading ? (
          <Card style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <div style={{ textAlign: 'center', padding: 48 }}>
              <Spin tip="周报加载中..." />
            </div>
          </Card>
        ) : !weeklyData ? (
          <Card style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <Empty description="暂无周报数据" />
          </Card>
        ) : (
          <>
            <Card title={`周报 ${weeklyData.week_start} ~ ${weeklyData.week_end}`} style={{ marginBottom: 16, background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
              <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap' }}>
                <Statistic title="销售总额" value={weeklyData.sales_total} prefix="¥" precision={2} />
                <Statistic title="订单总数" value={weeklyData.orders_total} />
                <Statistic title="异常总数" value={weeklyData.anomalies_total}
                  valueStyle={{ color: weeklyData.anomalies_total > 0 ? '#cf1322' : undefined }} />
              </div>
            </Card>
            <Card title="每日明细" style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
              <Table
                rowKey="date"
                dataSource={weeklyData.daily_reports}
                pagination={false}
                columns={WEEKLY_DAILY_COLUMNS}
              />
            </Card>
          </>
        )
      )}

      {isGenericTab && (
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
