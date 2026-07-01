'use client';

import { Divider, Drawer, Empty, Space, Table, Tag, Typography, Descriptions } from 'antd';
import type { WorkItemDetail } from './types';

const { Text } = Typography;

const riskColor = (v: string): string => {
  if (v === 'high' || v === 'critical') return 'red';
  if (v === 'medium') return 'orange';
  if (v === 'low') return 'green';
  return 'blue';
};

const statusColor = (v: string): string => {
  if (v === 'suggested') return 'blue';
  if (v === 'approved' || v === 'executed') return 'green';
  if (v === 'executing') return 'cyan';
  if (v === 'rejected' || v === 'failed') return 'red';
  if (v === 'reviewed') return 'default';
  return 'blue';
};

interface Props {
  open: boolean;
  detail?: WorkItemDetail;
  loading?: boolean;
  onClose: () => void;
  width?: number;
}

export default function WorkItemDrawer({
  open, detail, loading, onClose, width = 640,
}: Props) {
  return (
    <Drawer
      title={detail?.title ?? '工作项详情'}
      open={open}
      onClose={onClose}
      width={width}
      loading={loading}
    >
      {detail ? (
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Descriptions size="small" column={2} bordered>
            <Descriptions.Item label="ID">{detail.id}</Descriptions.Item>
            <Descriptions.Item label="Agent">{detail.agent_id}</Descriptions.Item>
            <Descriptions.Item label="Squad">{detail.squad_id}</Descriptions.Item>
            <Descriptions.Item label="风险">
              <Tag color={riskColor(detail.risk_level)}>{detail.risk_level}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusColor(detail.status)}>{detail.status}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="置信度">
              {detail.confidence !== null ? `${(detail.confidence * 100).toFixed(0)}%` : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="决策点">{detail.decision_point}</Descriptions.Item>
            <Descriptions.Item label="提议时间">{detail.proposed_at || '-'}</Descriptions.Item>
            <Descriptions.Item label="Trace ID" span={2}>{detail.trace_id ?? '-'}</Descriptions.Item>
          </Descriptions>

          <CodeBlock label="决策理由" content={detail.reason} />
          <CodeBlock label="输入摘要" content={detail.input_summary} />
          <CodeBlock label="输出摘要" content={detail.output_summary} />

          <Divider>实体信息</Divider>
          <Descriptions size="small" column={2} bordered>
            <Descriptions.Item label="实体类型">{detail.entity_type}</Descriptions.Item>
            <Descriptions.Item label="实体 ID">{detail.entity_id ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="实体状态">{detail.entity_status}</Descriptions.Item>
          </Descriptions>

          {detail.approval && (
            <>
              <Divider>审批信息</Divider>
              <Descriptions size="small" column={2} bordered>
                <Descriptions.Item label="审批 ID">{detail.approval.id}</Descriptions.Item>
                <Descriptions.Item label="审批状态">
                  <Tag color={statusColor(detail.approval.status)}>{detail.approval.status}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="审批风险">
                  <Tag color={riskColor(detail.approval.risk_level)}>{detail.approval.risk_level}</Tag>
                </Descriptions.Item>
              </Descriptions>
            </>
          )}

          <RelatedItemsTable label="上游工作项" items={detail.upstream_items ?? []} />
          <RelatedItemsTable label="下游工作项" items={detail.downstream_items ?? []} />
          <AuditLogTable logs={detail.audit_logs ?? []} />
        </Space>
      ) : (
        <Empty description={loading ? '加载中...' : '无法加载工作项详情'} />
      )}
    </Drawer>
  );
}

function CodeBlock({ label, content }: { label: string; content?: string | null }) {
  if (!content) return null;
  return (
    <>
      <Divider>{label}</Divider>
      <div style={{ background: 'var(--bg)', padding: 12, borderRadius: 6, whiteSpace: 'pre-wrap', fontSize: '0.8rem' }}>
        {content}
      </div>
    </>
  );
}

function RelatedItemsTable({ label, items }: { label: string; items: Array<{ id: number; type: string; title: string; status: string }> }) {
  if (items.length === 0) return null;
  return (
    <>
      <Divider>{label} ({items.length})</Divider>
      <Table
        rowKey="id"
        dataSource={items}
        size="small"
        pagination={false}
        columns={[
          { title: 'ID', dataIndex: 'id', width: 60 },
          { title: '类型', dataIndex: 'type', width: 80 },
          { title: '标题', dataIndex: 'title', ellipsis: true },
          { title: '状态', dataIndex: 'status', width: 100, render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag> },
        ]}
      />
    </>
  );
}

function AuditLogTable({ logs }: { logs: Array<{ id: number; action: string; content: string; operator: string; created_at: string }> }) {
  if (logs.length === 0) return null;
  return (
    <>
      <Divider>审计日志</Divider>
      <Table
        rowKey="id"
        dataSource={logs}
        size="small"
        pagination={false}
        columns={[
          { title: '操作', dataIndex: 'action', width: 100 },
          { title: '内容', dataIndex: 'content', ellipsis: true },
          { title: '操作人', dataIndex: 'operator', width: 80 },
          { title: '时间', dataIndex: 'created_at', width: 150 },
        ]}
      />
    </>
  );
}
