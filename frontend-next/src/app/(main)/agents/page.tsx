'use client';

import { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Col,
  Form,
  Input,
  message,
  Modal,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
} from 'antd';
import {
  PlayCircleOutlined,
  ReloadOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';

const { Text } = Typography;

// ---------- Types ----------
interface AgentListItem {
  id: string;
  name: string;
  squad: string;
  autonomy: string;
  decision_points: string[];
  description: string;
  status: string;
}

interface ActionRunResponse {
  trace_id: string;
  agent_id?: string;
  output?: string;
  status?: string;
}

// ---------- Color helpers ----------
const squadColor = (squad: string): string => {
  if (squad === 'autonomous') return 'blue';
  if (squad === 'governance') return 'purple';
  if (squad === 'ops') return 'gold';
  return 'default';
};

const autonomyColor = (level: string): string => {
  if (level === 'advisory') return 'green';
  if (level === 'guided') return 'blue';
  if (level === 'autonomous') return 'cyan';
  if (level === 'supervised') return 'orange';
  return 'default';
};

const statusColor = (status: string): string => {
  if (status === 'active' || status === 'running') return 'green';
  if (status === 'paused' || status === 'idle') return 'orange';
  if (status === 'error' || status === 'failed') return 'red';
  return 'blue';
};

// ---------- Page ----------
export default function AgentsPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [runModalOpen, setRunModalOpen] = useState(false);
  const [activeAgent, setActiveAgent] = useState<AgentListItem | null>(null);
  const [form] = Form.useForm<{ decision_point: string; context: string }>();

  const { data: agents, isLoading } = useQuery({
    queryKey: ['agents-list'],
    queryFn: async () => {
      const res = await apiClient.get<AgentListItem[]>('/v1/agents');
      return res.data ?? [];
    },
  });

  // 按 squad 分组统计
  const squadStats = useMemo(() => {
    const map = new Map<string, number>();
    (agents ?? []).forEach((a) => {
      map.set(a.squad, (map.get(a.squad) ?? 0) + 1);
    });
    return Array.from(map.entries());
  }, [agents]);

  const runMutation = useMutation({
    mutationFn: async (params: {
      agentId: string;
      decision_point: string;
      context: Record<string, unknown>;
    }) => {
      const res = await apiClient.post<ActionRunResponse>(
        `/v1/agents/${params.agentId}/actions`,
        {
          decision_point: params.decision_point,
          context: params.context,
        },
      );
      return res.data;
    },
    onSuccess: (data) => {
      message.success(`已触发，trace_id: ${data?.trace_id ?? '-'}`);
      setRunModalOpen(false);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['agents-list'] });
      if (data?.trace_id && activeAgent) {
        router.push(`/agents/${activeAgent.id}/trace/${data.trace_id}`);
      }
    },
    onError: (e: Error) => message.error(`执行失败: ${e.message}`),
  });

  const handleOpenRun = (agent: AgentListItem) => {
    setActiveAgent(agent);
    form.resetFields();
    form.setFieldsValue({
      decision_point: agent.decision_points?.[0] ?? '',
      context: '{}',
    });
    setRunModalOpen(true);
  };

  const handleSubmitRun = async () => {
    if (!activeAgent) return;
    const values = await form.validateFields();
    let contextObj: Record<string, unknown> = {};
    try {
      contextObj = values.context ? JSON.parse(values.context) : {};
    } catch {
      message.error('context 不是合法的 JSON');
      return;
    }
    runMutation.mutate({
      agentId: activeAgent.id,
      decision_point: values.decision_point,
      context: contextObj,
    });
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 140 },
    {
      title: '名称',
      dataIndex: 'name',
      width: 160,
      render: (v: string, record: AgentListItem) => (
        <Button
          type="link"
          style={{ padding: 0 }}
          onClick={() => router.push(`/agents/${record.id}`)}
        >
          {v}
        </Button>
      ),
    },
    {
      title: 'Squad',
      dataIndex: 'squad',
      width: 110,
      render: (v: string) => <Tag color={squadColor(v)}>{v}</Tag>,
    },
    {
      title: 'Autonomy',
      dataIndex: 'autonomy',
      width: 110,
      render: (v: string) => <Tag color={autonomyColor(v)}>{v}</Tag>,
    },
    {
      title: '决策点',
      dataIndex: 'decision_points',
      width: 240,
      render: (v: string[]) => (
        <Space size={4} wrap>
          {(v ?? []).map((dp) => (
            <Tag key={dp} color="blue">
              {dp}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      ellipsis: true,
      render: (v: string) => (
        <Text type="secondary" ellipsis={{ tooltip: v }}>
          {v ?? '-'}
        </Text>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag>,
    },
    {
      title: '操作',
      key: 'actions',
      width: 110,
      fixed: 'right' as const,
      render: (_: unknown, record: AgentListItem) => (
        <Button
          type="primary"
          size="small"
          icon={<PlayCircleOutlined />}
          onClick={() => handleOpenRun(record)}
        >
          运行
        </Button>
      ),
    },
  ];

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
        <h1 style={{ fontSize: 24, fontWeight: 600, margin: 0 }}>Agent 列表</h1>
        <Button
          icon={<ReloadOutlined />}
          onClick={() => qc.invalidateQueries({ queryKey: ['agents-list'] })}
        >
          刷新
        </Button>
      </div>

      {/* 顶部统计 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic
              title="Agent 总数"
              value={agents?.length ?? 0}
              prefix={<TeamOutlined style={{ color: '#1677ff' }} />}
            />
          </Card>
        </Col>
        {squadStats.map(([squad, count]) => (
          <Col xs={12} sm={6} key={squad}>
            <Card>
              <Statistic
                title={
                  <Space size={4}>
                    <Tag color={squadColor(squad)} style={{ marginRight: 0 }}>
                      {squad}
                    </Tag>
                    <span>Agent 数</span>
                  </Space>
                }
                value={count}
                valueStyle={{
                  color:
                    squad === 'autonomous'
                      ? '#1677ff'
                      : squad === 'governance'
                        ? '#722ed1'
                        : squad === 'ops'
                          ? '#fa8c16'
                          : undefined,
                }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      <Card>
        <Table
          rowKey="id"
          loading={isLoading}
          dataSource={agents ?? []}
          columns={columns}
          scroll={{ x: 'max-content' }}
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 个 Agent`,
          }}
        />
      </Card>

      {/* 运行 Modal */}
      <Modal
        title={`运行 Agent: ${activeAgent?.name ?? ''}`}
        open={runModalOpen}
        onCancel={() => {
          setRunModalOpen(false);
          form.resetFields();
        }}
        onOk={handleSubmitRun}
        confirmLoading={runMutation.isPending}
        okText="运行"
        cancelText="取消"
        width={560}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="decision_point"
            label="决策点"
            rules={[{ required: true, message: '请选择决策点' }]}
          >
            <Select
              placeholder="选择决策点"
              options={(activeAgent?.decision_points ?? []).map((dp) => ({
                label: dp,
                value: dp,
              }))}
            />
          </Form.Item>
          <Form.Item
            name="context"
            label="Context (JSON)"
            rules={[{ required: true, message: '请输入 context' }]}
            extra="请输入合法的 JSON 对象"
          >
            <Input.TextArea rows={6} placeholder='{"key": "value"}' />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
