'use client';

import { useState } from 'react';
import {
  Button, Card, Col, Form, Input, InputNumber, message,
  Modal, Row, Select, Space, Switch, Table, Tag, Typography,
} from 'antd';
import {
  DeleteOutlined, EditOutlined, PlusOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import type { Result } from '@/types/api';

const { Title } = Typography;

interface PolicyRule {
  id: number;
  name: string;
  description: string;
  risk_level: string;
  action_type: string;
  squad_id: string;
  agent_id: string;
  business_object_type: string;
  max_amount: number | null;
  max_quantity: number | null;
  min_confidence: number | null;
  auto_approve: boolean;
  outcome: string;
  priority: number;
  enabled: boolean;
}

type PolicyRuleFormData = {
  name: string;
  description: string;
  risk_level: string;
  action_type: string;
  agent_id: string;
  business_object_type: string;
  max_amount: number | null;
  max_quantity: number | null;
  min_confidence: number | null;
  outcome: string;
  priority: number;
  enabled: boolean;
};

const OUTCOME_COLORS: Record<string, string> = {
  auto_approve: 'green',
  escalate: 'orange',
  block: 'red',
};

export default function PolicySettingsPage() {
  const queryClient = useQueryClient();
  const [detailModal, setDetailModal] = useState<PolicyRule | null>(null);
  const [formModalOpen, setFormModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<PolicyRule | null>(null);
  const [form] = Form.useForm();

  const { data: rulesData, isLoading } = useQuery<Result<PolicyRule[]>>({
    queryKey: ['policy-rules'],
    queryFn: () => apiClient.get<PolicyRule[]>('/v1/policy/rules'),
  });

  const rules = rulesData?.data || [];

  const createMutation = useMutation({
    mutationFn: (values: PolicyRuleFormData) =>
      apiClient.post('/v1/policy/rules', values),
    onSuccess: () => {
      message.success('创建成功');
      setFormModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['policy-rules'] });
    },
    onError: () => message.error('创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: (values: PolicyRuleFormData & { id: number }) =>
      apiClient.put('/v1/policy/rules/' + values.id, values),
    onSuccess: () => {
      message.success('更新成功');
      setFormModalOpen(false);
      setEditingRule(null);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['policy-rules'] });
    },
    onError: () => message.error('更新失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) =>
      apiClient.delete('/v1/policy/rules/' + id),
    onSuccess: () => {
      message.success('删除成功');
      queryClient.invalidateQueries({ queryKey: ['policy-rules'] });
    },
    onError: () => message.error('删除失败'),
  });

  const toggleMutation = useMutation({
    mutationFn: (id: number) =>
      apiClient.post('/v1/policy/rules/' + id + '/toggle'),
    onSuccess: () => {
      message.success('状态已更新');
      queryClient.invalidateQueries({ queryKey: ['policy-rules'] });
    },
    onError: () => message.error('状态更新失败'),
  });

  const openCreateModal = () => {
    setEditingRule(null);
    form.resetFields();
    setFormModalOpen(true);
  };

  const openEditModal = (rule: PolicyRule) => {
    setEditingRule(rule);
    form.setFieldsValue({
      name: rule.name,
      description: rule.description,
      risk_level: rule.risk_level,
      action_type: rule.action_type,
      agent_id: rule.agent_id,
      business_object_type: rule.business_object_type,
      max_amount: rule.max_amount,
      max_quantity: rule.max_quantity,
      min_confidence: rule.min_confidence,
      outcome: rule.outcome,
      priority: rule.priority,
      enabled: rule.enabled,
    });
    setFormModalOpen(true);
  };

  const handleFormSubmit = () => {
    form.validateFields()
      .then((values) => {
        const payload: PolicyRuleFormData = {
          ...values,
          max_amount: values.max_amount ?? null,
          max_quantity: values.max_quantity ?? null,
          min_confidence: values.min_confidence ?? null,
        };
        if (editingRule) {
          updateMutation.mutate({ ...payload, id: editingRule.id });
        } else {
          createMutation.mutate(payload);
        }
      })
      .catch(() => {});
  };

  const handleDelete = (rule: PolicyRule) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除规则「' + rule.name + '」吗？此操作不可恢复。',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: () => deleteMutation.mutate(rule.id),
    });
  };

  const isFormPending = createMutation.isPending || updateMutation.isPending;

  const columns = [
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      sorter: (a: PolicyRule, b: PolicyRule) => b.priority - a.priority,
    },
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: PolicyRule) => (
        <a onClick={() => setDetailModal(record)}>{name}</a>
      ),
    },
    {
      title: '风险等级',
      dataIndex: 'risk_level',
      key: 'risk_level',
      width: 100,
      render: (level: string) => level ? <Tag>{level}</Tag> : <Tag>*</Tag>,
    },
    {
      title: '动作类型',
      dataIndex: 'action_type',
      key: 'action_type',
      width: 120,
      render: (t: string) => t === '*' ? <Tag>全部</Tag> : <Tag>{t}</Tag>,
    },
    {
      title: 'Agent',
      dataIndex: 'agent_id',
      key: 'agent_id',
      width: 80,
      render: (id: string | null) => id === '*' || !id ? <Tag>全部</Tag> : <Tag>{id}</Tag>,
    },
    {
      title: '阈值',
      key: 'thresholds',
      width: 200,
      render: (_: unknown, r: PolicyRule) => {
        const parts: string[] = [];
        if (r.max_amount) parts.push('¥' + r.max_amount);
        if (r.max_quantity) parts.push(r.max_quantity + '件');
        if (r.min_confidence) parts.push('置信>' + (r.min_confidence * 100).toFixed(0) + '%');
        return parts.length > 0 ? parts.join(', ') : '-';
      },
    },
    {
      title: '结果',
      dataIndex: 'outcome',
      key: 'outcome',
      width: 120,
      render: (o: string) => (
        <Tag color={OUTCOME_COLORS[o]}>
          {o === 'auto_approve' ? '自动批准' : o === 'escalate' ? '人工审批' : '阻止'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (e: boolean) => (e ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: PolicyRule) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openEditModal(record)}
          />
          <Switch
            size="small"
            checked={record.enabled}
            onChange={() => toggleMutation.mutate(record.id)}
          />
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record)}
          />
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Title level={3} style={{ margin: 0 }}>审批策略</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
          新建策略
        </Button>
      </div>
      <p style={{ color: '#666', marginBottom: 16 }}>
        审批策略决定 AI Agent 的动作是否需要人工审批、自动批准或阻止执行。
        策略按优先级顺序评估，最严格的结果生效。
      </p>

      <Card>
        <Table
          dataSource={rules}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={false}
          size="small"
        />
      </Card>

      {/* Detail modal */}
      <Modal
        title={detailModal?.name || '规则详情'}
        open={!!detailModal}
        onCancel={() => setDetailModal(null)}
        footer={<Button onClick={() => setDetailModal(null)}>关闭</Button>}
        width={560}
      >
        {detailModal && (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <tbody>
              <tr>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap', width: 100 }}>名称</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }} colSpan={3}>{detailModal.name}</td>
              </tr>
              <tr>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>描述</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }} colSpan={3}>{detailModal.description || '-'}</td>
              </tr>
              <tr>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>风险等级</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }}>{detailModal.risk_level || '全部'}</td>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>动作类型</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }}>{detailModal.action_type || '全部'}</td>
              </tr>
              <tr>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>Agent</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }}>{detailModal.agent_id || '全部'}</td>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>业务对象</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }}>{detailModal.business_object_type || '全部'}</td>
              </tr>
              <tr>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>最大金额</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }}>{detailModal.max_amount ? '¥' + detailModal.max_amount : '无限制'}</td>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>最大数量</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }}>{detailModal.max_quantity ? detailModal.max_quantity + '件' : '无限制'}</td>
              </tr>
              <tr>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>最低置信度</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }}>{detailModal.min_confidence ? (detailModal.min_confidence * 100).toFixed(0) + '%' : '无限制'}</td>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>结果</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }}>
                  <Tag color={OUTCOME_COLORS[detailModal.outcome]}>
                    {detailModal.outcome === 'auto_approve' ? '自动批准' : detailModal.outcome === 'escalate' ? '人工审批' : '阻止'}
                  </Tag>
                </td>
              </tr>
              <tr>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>优先级</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }}>{detailModal.priority}</td>
                <th style={{ padding: '8px 12px', background: '#fafafa', fontWeight: 500, border: '1px solid #f0f0f0', whiteSpace: 'nowrap' }}>是否启用</th>
                <td style={{ padding: '8px 12px', border: '1px solid #f0f0f0' }}>{detailModal.enabled ? '是' : '否'}</td>
              </tr>
            </tbody>
          </table>
        )}
      </Modal>

      {/* Create / Edit modal */}
      <Modal
        title={editingRule ? '编辑策略' : '新建策略'}
        open={formModalOpen}
        onCancel={() => {
          setFormModalOpen(false);
          setEditingRule(null);
          form.resetFields();
        }}
        footer={[
          <Button key="cancel" onClick={() => {
            setFormModalOpen(false);
            setEditingRule(null);
            form.resetFields();
          }}>
            取消
          </Button>,
          <Button key="submit" type="primary" loading={isFormPending} onClick={handleFormSubmit}>
            {editingRule ? '保存' : '创建'}
          </Button>,
        ]}
        width={640}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            priority: 0,
            enabled: true,
            risk_level: 'medium',
            outcome: 'escalate',
          }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="名称"
                rules={[{ required: true, message: '请输入规则名称' }]}
              >
                <Input placeholder="规则名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="priority" label="优先级">
                <InputNumber style={{ width: '100%' }} min={0} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="risk_level" label="风险等级">
                <Select
                  options={[
                    { value: 'low', label: '低' },
                    { value: 'medium', label: '中' },
                    { value: 'high', label: '高' },
                    { value: 'critical', label: '严重' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="outcome" label="结果">
                <Select
                  options={[
                    { value: 'auto_approve', label: '自动批准' },
                    { value: 'escalate', label: '人工审批' },
                    { value: 'block', label: '阻止' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="action_type" label="动作类型">
                <Input placeholder="如: list_product" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="agent_id" label="Agent">
                <Input placeholder="如: A1" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="business_object_type" label="业务对象类型">
            <Input placeholder="如: product" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="max_amount" label="最大金额">
                <InputNumber style={{ width: '100%' }} min={0} precision={2} prefix="¥" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="max_quantity" label="最大数量">
                <InputNumber style={{ width: '100%' }} min={0} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="min_confidence" label="最低置信度 (0-1)">
            <InputNumber
              style={{ width: '100%' }}
              min={0}
              max={1}
              step={0.1}
              placeholder="如: 0.8"
            />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="规则描述（可选）" />
          </Form.Item>

          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
