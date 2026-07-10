'use client';

import { Card, Tag, Table, Alert, Descriptions } from 'antd';
import { CheckCircleOutlined, WarningOutlined, CloseCircleOutlined } from '@ant-design/icons';

interface CostItem {
  category: string;
  label: string;
  amount: number;
  rate: number;
  calculation_type: string;
  data_source: string;
  source_note: string;
  required: boolean;
}

interface DataField {
  field_name: string;
  label: string;
  value: string;
  source: string;
}

export interface EvidenceCardData {
  product_id: number;
  title: string;
  currency: string;
  revenue: { amount: number; label: string };
  cost_items: CostItem[];
  total_fixed_cost: number;
  total_variable_fee_rate: number;
  estimated_variable_fee: number;
  total_cost_at_target_price: number;
  estimated_profit: number;
  profit_margin: number;
  status: string;
  confidence_level: string;
  can_evaluate: boolean;
  confirmed_items: DataField[];
  estimated_items: DataField[];
  missing_items: string[];
  blocking_reasons: string[];
  break_even_price: number;
  recommended_min_price: number;
  target_margin: number;
  buffer_rate: number;
}

const sourceIcon = (source: string) => {
  switch (source) {
    case 'confirmed': return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
    case 'estimated': return <WarningOutlined style={{ color: '#faad14' }} />;
    case 'template_default': return <WarningOutlined style={{ color: '#faad14' }} />;
    default: return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
  }
};

const sourceColor = (source: string) => {
  switch (source) {
    case 'confirmed': return 'green';
    case 'estimated': return 'orange';
    case 'template_default': return 'orange';
    default: return 'red';
  }
};

const statusColor = (status: string) => {
  switch (status) {
    case 'profitable': return 'green';
    case 'marginal': return 'orange';
    case 'unprofitable': return 'red';
    default: return 'default';
  }
};

export default function EvidenceCard({ data }: { data: EvidenceCardData }) {
  const costColumns = [
    { title: '项目', dataIndex: 'label', key: 'label' },
    {
      title: '金额', dataIndex: 'amount', key: 'amount',
      render: (v: number, r: CostItem) => r.calculation_type === 'percent_of_revenue'
        ? `${(r.rate * 100).toFixed(1)}% ($${v.toFixed(2)})`
        : `$${v.toFixed(2)}`,
    },
    {
      title: '数据源', dataIndex: 'data_source', key: 'data_source',
      render: (v: string) => <Tag color={sourceColor(v)}>{v}</Tag>,
    },
    { title: '备注', dataIndex: 'source_note', key: 'source_note' },
  ];

  return (
    <div>
      {/* Conclusion bar */}
      <Alert
        type={data.can_evaluate ? 'info' : 'error'}
        message={
          <strong>
            利润信心等级: {data.confidence_level}
            {data.can_evaluate
              ? ` | 预估利润: $${data.estimated_profit.toFixed(2)} (${data.profit_margin.toFixed(1)}%)`
              : ' | 数据不足，无法评估'}
          </strong>
        }
        style={{ marginBottom: 16 }}
      />

      {/* Revenue */}
      <Card size="small" title="收入" style={{ marginBottom: 12 }}>
        <Descriptions column={1} size="small">
          <Descriptions.Item label={data.revenue.label}>
            ${data.revenue.amount.toFixed(2)} {data.currency}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Cost breakdown */}
      <Card size="small" title="成本明细" style={{ marginBottom: 12 }}>
        <Table
          dataSource={data.cost_items}
          columns={costColumns}
          rowKey="category"
          pagination={false}
          size="small"
          summary={() => (
            <Table.Summary.Row>
              <Table.Summary.Cell index={0}><strong>总固定成本</strong></Table.Summary.Cell>
              <Table.Summary.Cell index={1}><strong>${data.total_fixed_cost.toFixed(2)}</strong></Table.Summary.Cell>
              <Table.Summary.Cell index={2} />
              <Table.Summary.Cell index={3} />
            </Table.Summary.Row>
          )}
        />
      </Card>

      {/* Price recommendations */}
      {data.can_evaluate && (
        <Card size="small" title="达标售价测算" style={{ marginBottom: 12 }}>
          <Descriptions column={1} size="small">
            <Descriptions.Item label="盈亏平衡价">${data.break_even_price.toFixed(2)}</Descriptions.Item>
            <Descriptions.Item label="建议最低售价">${data.recommended_min_price.toFixed(2)}</Descriptions.Item>
            <Descriptions.Item label="目标利润率">{(data.target_margin * 100).toFixed(0)}%</Descriptions.Item>
            <Descriptions.Item label="缓冲率">{(data.buffer_rate * 100).toFixed(0)}%</Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {/* Data quality */}
      <Card size="small" title="数据质量">
        {data.confirmed_items.length > 0 && (
          <div>✅ 已确认: {data.confirmed_items.map(i => i.label).join('、')}</div>
        )}
        {data.estimated_items.length > 0 && (
          <div>⚠️ 估算: {data.estimated_items.map(i => i.label).join('、')}</div>
        )}
        {data.missing_items.length > 0 && (
          <div>❌ 缺失: {data.missing_items.map(i => i).join('、')}</div>
        )}
        {data.blocking_reasons.length > 0 && (
          <Alert type="error" message={data.blocking_reasons.join('；')} style={{ marginTop: 8 }} />
        )}
      </Card>
    </div>
  );
}
