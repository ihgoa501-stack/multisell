'use client';

import { useState } from 'react';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Form,
  Input,
  message,
  Modal,
  Row,
  Space,
  Spin,
  Tag,
  Timeline,
  Typography,
} from 'antd';
import {
  ReloadOutlined,
  CheckOutlined,
  CloseOutlined,
  ThunderboltOutlined,
  AuditOutlined,
  ArrowLeftOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';

const { Text, Title } = Typography;

// ---------- Types ----------
interface UnifiedAction {
  id: string;
  title: string;
  agent_id: string;
  risk_level: string;
  confidence: number;
  status: string;
  trace_id?: string;
  description?: string;
  risk_reason?: string;
  payload?: Record<string, unknown>;
  before_snapshot?: Record<string, unknown>;
  after_snapshot?: Record<string, unknown>;
  proposed_at?: string;
  approved_at?: string;
  executing_at?: string;
  executed_at?: string;
  reviewed_at?: string;
  rejected_at?: string;
  reject_reason?: string;
  operator?: string;
}

// ---------- Color helpers ----------
const riskColor = (level: string): string => {
  if (level === 'high' || level === 'critical') return 'red';
  if (level === 'medium') return 'orange';
  if (level === 'low') return 'green';
  return 'blue';
};

const statusColor = (status: string): string => {
  if (status === 'suggested') return 'blue';
  if (status === 'approved' || status === 'executed') return 'green';
  if (status === 'executing') return 'cyan';
  if (status === 'rejected' || status === 'failed') return 'red';
  if (status === 'reviewed') return 'default';
  return 'blue';
};

// ---------- Page ----------
export default function ActionDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const qc = useQueryClient();
  const actionId = params?.id ?? '';
  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectForm] = Form.useForm<{ reason: string }>();

  const { data: action, isLoading } = useQuery({
    queryKey: ['ai-action', actionId],
    queryFn: async () => {
      const res = await apiClient.get<UnifiedAction>(`/v1/ai/actions/${actionId}`);
      return res.data;
    },
    enabled: !!actionId,
  });

  const approveMutation = useMutation({
    mutationFn: async () =>
      apiClient.post<unknown>(`/v1/ai/actions/${actionId}/approve`, {
        operator: 'operator',
      }),
    onSuccess: () => {
      message.success('已批准');
      qc.invalidateQueries({ queryKey: ['ai-action', actionId] });
    },
    onError: (e: Error) => message.error(`批准失败: ${e.message}`),
  });

  const rejectMutation = useMutation({
    mutationFn: async (reason: string) =>
      apiClient.post<unknown>(`/v1/ai/actions/${actionId}/reject`, {
        operator: 'operator',
        reason,
      }),
    onSuccess: () => {
      message.success('已拒绝');
      setRejectOpen(false);
      rejectForm.resetFields();
      qc.invalidateQueries({ queryKey: ['ai-action', actionId] });
    },
    onError: (e: Error) => message.error(`拒绝失败: ${e.message}`),
  });

  const executeMutation = useMutation({
    mutationFn: async () =>
      apiClient.post<unknown>(`/v1/ai/actions/${actionId}/execute`, {
        operator: 'operator',
      }),
    onSuccess: () => {
      message.success('已执行');
      qc.invalidateQueries({ queryKey: ['ai-action', actionId] });
    },
    onError: (e: Error) => message.error(`执行失败: ${e.message}`),
  });

  const reviewMutation = useMutation({
    mutationFn: async () =>
      apiClient.post<unknown>(`/v1/ai/actions/${actionId}/review`),
    onSuccess: () => {
      message.success('已标记审查');
      qc.invalidateQueries({ queryKey: ['ai-action', actionId] });
    },
    onError: (e: Error) => message.error(`审查失败: ${e.message}`),
  });

  const handleReject = async () => {
    const values = await rejectForm.validateFields();
    rejectMutation.mutate(values.reason);
  };

  const hasSnapshot =
    action?.before_snapshot || action?.after_snapshot
      ? Object.keys(action?.before_snapshot ?? {}).length > 0 ||
        Object.keys(action?.after_snapshot ?? {}).length > 0
      : false;

  // 审计时间线
  const auditItems = [
    {
      label: 'proposed_at',
      time: action?.proposed_at,
      color: 'blue',
      label_cn: '提议',
    },
    {
      label: 'approved_at',
      time: action?.approved_at,
      color: 'green',
      label_cn: '批准',
    },
    {
      label: 'executing_at',
      time: action?.executing_at,
      color: 'cyan',
      label_cn: '执行中',
    },
    {
      label: 'executed_at',
      time: action?.executed_at,
      color: 'green',
      label_cn: '已执行',
    },
    {
      label: 'reviewed_at',
      time: action?.reviewed_at,
      color: 'gray',
      label_cn: '已审查',
    },
  ]
    .filter((it) => it.time)
    .map((it) => ({
      color: it.color,
      children: (
        <div>
          <Text strong>{it.label_cn}</Text>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {dayjs(it.time).format('YYYY-MM-DD HH:mm:ss')}
            </Text>
          </div>
        </div>
      ),
    }));

  return (
    <div style={{ padding: 24 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <Space>
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => router.back()}
          >
            返回
          </Button>
          <Title level={3} style={{ margin: 0 }}>
            Action 审查室
          </Title>
        </Space>
        <Button
          icon={<ReloadOutlined />}
          onClick={() => qc.invalidateQueries({ queryKey: ['ai-action', actionId] })}
        >
          刷新
        </Button>
      </div>

      <Spin spinning={isLoading}>
        {!action && !isLoading ? (
          <Text>未找到 Action</Text>
        ) : action ? (
          <>
            {/* 顶部：标题 + 状态 + 风险 */}
            <Card style={{ marginBottom: 16 }}>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'flex-start',
                  flexWrap: 'wrap',
                  gap: 12,
                }}
              >
                <div style={{ flex: 1, minWidth: 0 }}>
                  <Space size={8} wrap>
                    <Title level={4} style={{ margin: 0 }}>
                      {action.title}
                    </Title>
                    <Tag color={statusColor(action.status)}>{action.status}</Tag>
                    <Tag color={riskColor(action.risk_level)}>
                      风险: {action.risk_level}
                    </Tag>
                    <Tag
                      color={
                        action.confidence >= 0.8
                          ? 'green'
                          : action.confidence >= 0.5
                            ? 'orange'
                            : 'red'
                      }
                    >
                      置信度 {(action.confidence * 100).toFixed(0)}%
                    </Tag>
                  </Space>
                  <div style={{ marginTop: 8 }}>
                    <Space size={16}>
                      <Text type="secondary">Action ID: {action.id}</Text>
                      <Text type="secondary">Agent: {action.agent_id}</Text>
                      {action.trace_id && (
                        <Button
                          type="link"
                          size="small"
                          style={{ padding: 0 }}
                          onClick={() =>
                            router.push(
                              `/agents/${action.agent_id}/trace/${action.trace_id}`,
                            )
                          }
                        >
                          查看 Trace
                        </Button>
                      )}
                    </Space>
                  </div>
                </div>
              </div>
            </Card>

            <Row gutter={16}>
              {/* 左侧：Before/After 对比 */}
              <Col xs={24} lg={12}>
                <Card title="Before / After 对比" style={{ marginBottom: 16 }}>
                  {hasSnapshot ? (
                    <Row gutter={12}>
                      <Col xs={24} sm={12}>
                        <Text strong style={{ display: 'block', marginBottom: 8 }}>
                          Before
                        </Text>
                        <pre
                          style={{
                            background: '#fff1f0',
                            border: '1px solid #ffa39e',
                            padding: 10,
                            borderRadius: 4,
                            fontSize: 12,
                            overflow: 'auto',
                            maxHeight: 320,
                            margin: 0,
                          }}
                        >
                          {JSON.stringify(action.before_snapshot ?? {}, null, 2)}
                        </pre>
                      </Col>
                      <Col xs={24} sm={12}>
                        <Text strong style={{ display: 'block', marginBottom: 8 }}>
                          After
                        </Text>
                        <pre
                          style={{
                            background: '#f6ffed',
                            border: '1px solid #b7eb8f',
                            padding: 10,
                            borderRadius: 4,
                            fontSize: 12,
                            overflow: 'auto',
                            maxHeight: 320,
                            margin: 0,
                          }}
                        >
                          {JSON.stringify(action.after_snapshot ?? {}, null, 2)}
                        </pre>
                      </Col>
                    </Row>
                  ) : (
                    <>
                      <Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
                        无 Snapshot，展示 Payload：
                      </Text>
                      <pre
                        style={{
                          background: '#f6f6f6',
                          padding: 10,
                          borderRadius: 4,
                          fontSize: 12,
                          overflow: 'auto',
                          maxHeight: 320,
                          margin: 0,
                        }}
                      >
                        {JSON.stringify(action.payload ?? {}, null, 2)}
                      </pre>
                    </>
                  )}
                </Card>
              </Col>

              {/* 右侧：风险说明 + 描述 */}
              <Col xs={24} lg={12}>
                <Card title="描述与风险说明" style={{ marginBottom: 16 }}>
                  <Descriptions column={1} size="small">
                    <Descriptions.Item label="描述">
                      {action.description ?? '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="风险说明">
                      {action.risk_reason ?? '-'}
                    </Descriptions.Item>
                    {action.reject_reason && (
                      <Descriptions.Item label="拒绝原因">
                        <Text type="danger">{action.reject_reason}</Text>
                      </Descriptions.Item>
                    )}
                    {action.operator && (
                      <Descriptions.Item label="操作人">
                        {action.operator}
                      </Descriptions.Item>
                    )}
                  </Descriptions>
                </Card>
              </Col>
            </Row>

            {/* 底部：操作区 */}
            <Card title="操作区" style={{ marginBottom: 16 }}>
              <Space size="middle" wrap>
                <Button
                  type="primary"
                  icon={<CheckOutlined />}
                  loading={approveMutation.isPending}
                  onClick={() => approveMutation.mutate()}
                  disabled={action.status !== 'suggested'}
                >
                  批准
                </Button>
                <Button
                  danger
                  icon={<CloseOutlined />}
                  onClick={() => setRejectOpen(true)}
                  disabled={action.status !== 'suggested'}
                >
                  拒绝
                </Button>
                <Button
                  icon={<ThunderboltOutlined />}
                  loading={executeMutation.isPending}
                  onClick={() => executeMutation.mutate()}
                  disabled={
                    action.status !== 'approved' && action.status !== 'suggested'
                  }
                >
                  执行
                </Button>
                <Button
                  icon={<AuditOutlined />}
                  loading={reviewMutation.isPending}
                  onClick={() => reviewMutation.mutate()}
                  disabled={action.status === 'reviewed'}
                >
                  标记已审查
                </Button>
              </Space>
            </Card>

            {/* 底部最下：审计时间线 */}
            <Card title="审计时间线">
              {auditItems.length === 0 ? (
                <Text type="secondary">暂无审计记录</Text>
              ) : (
                <Timeline items={auditItems} />
              )}
            </Card>
          </>
        ) : null}
      </Spin>

      {/* 拒绝原因输入框 */}
      <Modal
        title="拒绝 Action"
        open={rejectOpen}
        onCancel={() => {
          setRejectOpen(false);
          rejectForm.resetFields();
        }}
        onOk={handleReject}
        confirmLoading={rejectMutation.isPending}
        okText="确认拒绝"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <Form form={rejectForm} layout="vertical">
          <Form.Item
            name="reason"
            label="拒绝原因"
            rules={[{ required: true, message: '请输入拒绝原因' }]}
          >
            <Input.TextArea rows={3} placeholder="请输入拒绝原因..." />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
