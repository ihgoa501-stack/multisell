'use client';

import { useState } from 'react';
import { Card, Col, Row, Table, Tag, Statistic } from 'antd';
import { RetweetOutlined, ArrowUpOutlined, ArrowDownOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';

dayjs.extend(relativeTime);

interface MetabolismLog {
  id: number;
  source: string;
  total_score: number;
  impact_score: number;
  ref_score: number;
  freshness_score: number;
  semantic_score: number;
  sem_skipped: boolean;
  excretable: boolean;
  reason: string;
  created_at: string;
}

export default function WasteRecyclingPage() {
  const [page, setPage] = useState(1);

  // Fetch excretable metabolism logs (potential waste)
  const { data, isLoading } = useQuery({
    queryKey: ['metabolism-waste', page],
    queryFn: async () => {
      const res = await apiClient.get<{ data: MetabolismLog[]; total: number }>('/v1/metabolism', {
        page: String(page),
        page_size: '20',
      });
      return res.data;
    },
    refetchInterval: 30_000,
  });

  const logs = data?.data?.filter((l) => l.excretable) ?? [];
  const total = data?.total ?? 0;

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 100,
      render: (s: string) => <Tag>{s}</Tag>,
    },
    {
      title: '总分',
      dataIndex: 'total_score',
      key: 'total_score',
      width: 80,
      render: (v: number) => (
        <span style={{ color: v >= 0.7 ? '#ff4d4f' : '#faad14', fontWeight: 600 }}>
          {v.toFixed(2)}
        </span>
      ),
    },
    {
      title: '原因',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
    },
    {
      title: '状态',
      key: 'status',
      width: 80,
      render: () => <Tag color="orange">待回收</Tag>,
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 120,
      render: (v: string) => dayjs(v).fromNow(),
    },
  ];

  return (
    <PageContainer title="废物集市" subtitle="跨 Agent 废物回收中心">
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="待回收废物"
              value={logs.length}
              prefix={<RetweetOutlined />}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="已回收"
              value={0}
              prefix={<ArrowUpOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="回收率"
              value="0"
              suffix="%"
              prefix={<ArrowDownOutlined />}
            />
          </Card>
        </Col>
      </Row>
      <Card>
        <Table
          dataSource={logs}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={{
            current: page,
            pageSize: 20,
            total: logs.length,
            onChange: setPage,
            showSizeChanger: false,
          }}
          size="small"
        />
      </Card>
    </PageContainer>
  );
}
