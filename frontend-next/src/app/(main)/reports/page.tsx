'use client';

import { useState } from 'react';
import { Card, DatePicker, Descriptions, Empty, Select, Spin, Table, Tabs } from 'antd';
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

export default function ReportsPage() {
  const [activeTab, setActiveTab] = useState<TabKey>('sales');
  const [range, setRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(29, 'day'),
    dayjs(),
  ]);
  const [platformId, setPlatformId] = useState<string>('');

  const current = TABS.find((t) => t.key === activeTab)!;

  const { data, isLoading } = useQuery({
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
    retry: false,
  });

  const summaryEntries = Object.entries(data?.summary ?? []);
  const rows = data?.rows ?? [];

  return (
    <div style={{ padding: 24 }}>
      <h1 style={{ fontSize: 24, fontWeight: 600, marginBottom: 16 }}>报表</h1>

      <Tabs
        activeKey={activeTab}
        onChange={(k) => setActiveTab(k as TabKey)}
        items={TABS.map((t) => ({ key: t.key, label: t.label }))}
      />

      <Card style={{ marginBottom: 16 }}>
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

      {isLoading ? (
        <Card>
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin tip="报表加载中..." />
          </div>
        </Card>
      ) : (
        <>
          <Card title="汇总" style={{ marginBottom: 16 }}>
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

          <Card title="明细">
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
    </div>
  );
}
