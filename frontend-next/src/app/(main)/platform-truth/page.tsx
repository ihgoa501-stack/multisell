'use client';

import { useMemo, useState } from 'react';
import { Alert, Card, Col, Descriptions, Input, Row, Select, Skeleton, Space, Statistic, Table, Tag, Typography } from 'antd';
import { useQuery } from '@tanstack/react-query';
import PageHeader from '@/components/ui/PageHeader';
import { getPlatformTruth } from '@/features/platformtruth/api';

export default function PlatformTruthPage() {
  const [system, setSystem] = useState<string>();
  const [search, setSearch] = useState('');
  const query = useQuery({ queryKey: ['platform-truth'], queryFn: getPlatformTruth, retry: 1 });
  const rows = useMemo(() => (query.data?.domain_dispositions ?? []).filter((item) => {
    const text = `${item.id} ${item.name} ${item.reason}`.toLowerCase();
    return (!system || item.system === system) && (!search.trim() || text.includes(search.trim().toLowerCase()));
  }), [query.data, search, system]);
  return (
    <div style={{ padding: '16px 20px', minHeight: '100%' }}>
      <PageHeader title="平台真相" subtitle="当前唯一开发路线、事实等级和现有模块处置合同" />
      {query.isLoading && <Skeleton active />}
      {query.error && <Alert type="error" showIcon title="平台真相加载失败" description={query.error instanceof Error ? query.error.message : '未知错误'} />}
      {query.data && <Space orientation="vertical" size={16} style={{ width: '100%' }}>
        <Alert type="info" showIcon title={query.data.direction} description={`合同版本：${query.data.version}`} />
        <Row gutter={[12, 12]}>
          <Col xs={12} md={6}><Card><Statistic title="已分类领域" value={query.data.domain_dispositions.length} /></Card></Col>
          <Col xs={12} md={6}><Card><Statistic title="经营事实" value={query.data.domain_dispositions.filter((i) => i.system === 'fact').length} /></Card></Col>
          <Col xs={12} md={6}><Card><Statistic title="经营决策" value={query.data.domain_dispositions.filter((i) => i.system === 'decision').length} /></Card></Col>
          <Col xs={12} md={6}><Card><Statistic title="冻结/删除" value={query.data.domain_dispositions.filter((i) => i.system === 'frozen').length} /></Card></Col>
        </Row>
        <Card title="系统边界">
          <Descriptions column={{ xs: 1, lg: 2 }} bordered size="small">
            {query.data.system_boundaries.map((item) => <Descriptions.Item key={item.code} label={item.name}>{item.responsibility}<br /><Typography.Text type="danger">不得：{item.must_not}</Typography.Text></Descriptions.Item>)}
          </Descriptions>
        </Card>
        <Card title="事实等级">
          <Descriptions column={{ xs: 1, md: 2 }} bordered size="small">
            {query.data.truth_levels.map((item) => <Descriptions.Item key={item.code} label={<Tag>{item.code}</Tag>}>{item.meaning}</Descriptions.Item>)}
          </Descriptions>
        </Card>
        <Card title="工程声明等级">
          <Descriptions column={{ xs: 1, md: 2 }} bordered size="small">
            {query.data.claim_levels.map((item) => <Descriptions.Item key={item.code} label={<Tag>{item.code}</Tag>}>{item.meaning}</Descriptions.Item>)}
          </Descriptions>
        </Card>
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}><Card title="对象身份规则"><ul>{query.data.object_identity_rules.map((item) => <li key={item.code}>{item.rule}</li>)}</ul></Card></Col>
          <Col xs={24} lg={12}><Card title="来源规则"><ul>{query.data.source_rules.map((item) => <li key={item.code}>{item.rule}</li>)}</ul></Card></Col>
        </Row>
        <Card title="领域处置清单">
          <Space wrap style={{ marginBottom: 12 }}>
            <Input aria-label="搜索领域" allowClear placeholder="搜索领域或处置原因" value={search} onChange={(event) => setSearch(event.target.value)} style={{ width: 260 }} />
            <Select aria-label="系统筛选" allowClear placeholder="全部系统" value={system} onChange={setSystem} style={{ width: 180 }} options={query.data.system_boundaries.map((item) => ({ value: item.code, label: item.name }))} />
          </Space>
          <Table rowKey="id" pagination={{ pageSize: 20, showSizeChanger: false }} scroll={{ x: 1100 }} dataSource={rows} columns={[
            { title: '领域', dataIndex: 'name', width: 180 },
            { title: '系统', dataIndex: 'system', width: 110, render: (v: string) => <Tag>{v}</Tag> },
            { title: '处置', dataIndex: 'disposition', width: 110, render: (v: string) => <Tag color={v === 'freeze' ? 'default' : 'blue'}>{v}</Tag> },
            { title: '证据', dataIndex: 'evidence', width: 130 },
            { title: '小Q', dataIndex: 'xiao_q_support', width: 120 },
            { title: '风险', dataIndex: 'risk', width: 90 },
            { title: '原因', dataIndex: 'reason' },
          ]} />
        </Card>
        <Card title="不可越过的边界"><ul>{query.data.boundary_rules.map((item) => <li key={item}>{item}</li>)}</ul></Card>
        <Card title="仍然未知"><Typography.Text type="secondary"><ul>{query.data.unknowns.map((item) => <li key={item}>{item}</li>)}</ul></Typography.Text></Card>
      </Space>}
    </div>
  );
}
