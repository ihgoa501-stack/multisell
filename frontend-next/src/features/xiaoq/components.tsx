'use client';

import { Alert, Button, Card, Descriptions, Empty, Space, Tag, Typography } from 'antd';
import type {
  XiaoQCapability,
  XiaoQIdentity,
  XiaoQMessageResponse,
  XiaoQMode,
  XiaoQTruthStatus,
} from './types';

const { Paragraph, Text } = Typography;

const truthLabels: Record<XiaoQTruthStatus, string> = {
  actual: '已核验事实',
  quoted: '来源说法',
  estimated: '估算',
  inferred: '推断',
  unknown: '未知',
  mock: '模拟',
};

const truthColors: Record<XiaoQTruthStatus, string> = {
  actual: 'green', quoted: 'blue', estimated: 'gold', inferred: 'orange', unknown: 'default', mock: 'magenta',
};

export function TruthStatusTag({ status }: { status: XiaoQTruthStatus }) {
  return <Tag color={truthColors[status]}>{truthLabels[status]}</Tag>;
}

export function ModeTag({ mode }: { mode: XiaoQMode }) {
  const readOnly = mode === 'read_only' || mode === 'read_only_v1';
  return <Tag color={readOnly ? 'blue' : 'purple'}>{readOnly ? '只读模式' : '建议模式'}</Tag>;
}

export function XiaoQBoundaryBanner({ identity }: { identity?: XiaoQIdentity }) {
  return (
    <Alert
      type="info"
      showIcon
      message={<Space wrap><strong>{identity?.name ?? '小Q'}</strong><ModeTag mode={identity?.mode ?? 'read_only'} /></Space>}
      description="小Q可以读取系统信息、解释证据并提出建议，但不会直接执行发布、采购、价格、库存、订单或资金操作。模拟、未知和推断不会被当作已核验事实。"
    />
  );
}

export function XiaoQCapabilities({ capabilities }: { capabilities: XiaoQCapability[] }) {
  if (capabilities.length === 0) return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂未登记可用能力" />;
  return (
    <Space direction="vertical" size={8} style={{ width: '100%' }}>
      {capabilities.map((capability) => (
        <Card key={capability.code} size="small">
          <Space wrap>
            <Text strong>{capability.name}</Text>
            <ModeTag mode={capability.mode} />
            <Tag color={capability.available ? 'green' : 'default'}>{capability.available ? '可用' : '不可用'}</Tag>
            {capability.truth_status && <TruthStatusTag status={capability.truth_status} />}
          </Space>
          {capability.description && <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>{capability.description}</Paragraph>}
          {(capability.required_permission || capability.status || capability.approval_required !== undefined || capability.approval) && (
            <Space wrap style={{ marginTop: 8 }}>
              {capability.required_permission && <Text type="secondary">权限：<Text code>{capability.required_permission}</Text></Text>}
              {capability.status && <Tag>{capability.status}</Tag>}
              {capability.approval_required !== undefined && <Tag color={capability.approval_required ? 'gold' : 'green'}>{capability.approval_required ? '需要审批' : '无需审批'}</Tag>}
              {capability.approval && <Text type="secondary">审批：{capability.approval}</Text>}
            </Space>
          )}
          {!capability.available && capability.unavailable_reason && <Text type="secondary">原因：{capability.unavailable_reason}</Text>}
        </Card>
      ))}
    </Space>
  );
}

export function XiaoQAnswerCard({ response }: { response: XiaoQMessageResponse }) {
  const provenance = Array.isArray(response.provenance) ? response.provenance : response.provenance ? [response.provenance] : [];
  const isExperiment = response.target_type === 'experiment';
  const isSourcing = response.target_type === 'sourcing_1688';
  const isBusinessClosure = response.target_type === 'business_closure';
  return (
    <Card title={<Space wrap><span>小Q回答</span><ModeTag mode={response.mode} /><TruthStatusTag status={response.truth_status} /></Space>}>
      <Paragraph style={{ whiteSpace: 'pre-wrap', fontSize: 15 }}>{response.answer}</Paragraph>
      {response.case_summary && <Alert type="info" message="案件摘要" description={response.case_summary} style={{ marginBottom: 16 }} />}

      <Text strong>{isExperiment
        ? '经营实验证据'
        : isSourcing
          ? '受控来源、快照与成本证据'
          : isBusinessClosure
            ? '订单、结算与最终利润证据'
            : '证据与反证'}</Text>
      {response.evidence.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="本次回答没有可核验证据" />
      ) : (
        <Space direction="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
          {response.evidence.map((item, index) => (
            <Card key={item.id ?? `${item.title}-${index}`} size="small">
              <Space wrap><Text strong>{item.title}</Text><TruthStatusTag status={item.truth_status} /></Space>
              {item.summary && <Paragraph style={{ margin: '8px 0' }}>{item.summary}</Paragraph>}
              <Space wrap>
                {item.observed_at && <Text type="secondary">观察时间：{item.observed_at}</Text>}
                {item.run_id !== undefined && <Text type="secondary">Run：<Text code>{item.run_id}</Text></Text>}
                {item.snapshot_id !== undefined && <Text type="secondary">快照：<Text code>{item.snapshot_id}</Text></Text>}
                {item.snapshot_sha256 && <Text type="secondary">SHA-256：<Text code>{item.snapshot_sha256}</Text></Text>}
                {item.source_url && <a href={item.source_url} target="_blank" rel="noreferrer">查看来源</a>}
              </Space>
            </Card>
          ))}
        </Space>
      )}

      {response.unknowns.length > 0 && (
        <Alert
          type="warning"
          showIcon
          message={isExperiment
            ? '闸门阻断与仍然未知'
            : isSourcing
              ? '限制与仍然未知'
              : isBusinessClosure
                ? '经营闭环仍然未知'
                : '仍然未知'}
          description={<ul style={{ margin: 0, paddingLeft: 20 }}>{response.unknowns.map((item) => <li key={item}>{item}</li>)}</ul>}
          style={{ marginTop: 16 }}
        />
      )}

      {response.links.length > 0 && (
        <Space wrap style={{ marginTop: 16 }}>
          {response.links.map((link) => <Button key={`${link.href}-${link.label}`} href={link.href}>{link.label}</Button>)}
        </Space>
      )}

      <Descriptions size="small" column={1} style={{ marginTop: 16 }}>
        <Descriptions.Item label="追踪编号"><Text code>{response.trace_id}</Text></Descriptions.Item>
        <Descriptions.Item label="Agent"><Text code>{response.agent_id}</Text></Descriptions.Item>
        {provenance.length > 0 && <Descriptions.Item label="来源记录"><Text type="secondary">{JSON.stringify(provenance)}</Text></Descriptions.Item>}
      </Descriptions>
    </Card>
  );
}
