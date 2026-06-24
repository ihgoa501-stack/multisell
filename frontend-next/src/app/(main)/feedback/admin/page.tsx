'use client';

import { useEffect, useState } from 'react';
import {
  Typography, Card, Row, Col, Statistic, Table, Tag, Space, Button,
  Spin, Select, Modal, Form, Input, message, Tabs,
} from 'antd';
import {
  CheckCircleOutlined, ClockCircleOutlined, SendOutlined, RiseOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import { StatusBadge, TypeBadge, feedbackStatusList } from '@/components/feedback/FeedbackStatusBadge';

const { Title, Text } = Typography;
const { TextArea } = Input;

export default function FeedbackAdminPage() {
  const router = useRouter();
  const [projects, setProjects] = useState<any[]>([]);
  const [projectId, setProjectId] = useState<number | null>(null);
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  // Status update modal
  const [statusModal, setStatusModal] = useState<{ open: boolean; id: number; currentStatus: string } | null>(null);
  const [updateForm] = Form.useForm();
  const [updating, setUpdating] = useState(false);

  const init = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get<any[]>('/v1/feedback/projects');
      if (res.code === 0 && res.data) {
        setProjects(res.data);
        if (res.data.length > 0) {
          setProjectId(res.data[0].id);
        }
      }
    } catch {
      message.error('无法加载项目');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { init(); }, []);

  const fetchStats = async () => {
    if (!projectId) return;
    try {
      const res = await apiClient.get<any>(`/v1/feedback/projects/${projectId}/stats`);
      if (res.code === 0) setStats(res.data);
    } catch { /* ignore */ }
  };

  useEffect(() => { fetchStats(); }, [projectId]);

  const handleUpdateStatus = async () => {
    if (!statusModal) return;
    setUpdating(true);
    try {
      const values = await updateForm.validateFields();
      const res = await apiClient.put<any>(`/v1/feedback/submissions/${statusModal.id}/status`, values);
      if (res.code === 0) {
        message.success('状态已更新');
        setStatusModal(null);
        updateForm.resetFields();
        fetchList();
        fetchStats();
      }
    } catch (err: any) {
      if (err.errorFields) return; // validation error
      message.error(err.message || '更新失败');
    } finally {
      setUpdating(false);
    }
  };

  // We'll use a simple approach with state-managed list
  const [listData, setListData] = useState<any[]>([]);
  const [listLoading, setListLoading] = useState(false);
  const [listTab, setListTab] = useState('pending');

  const fetchList = async () => {
    if (!projectId) return;
    setListLoading(true);
    try {
      const status = listTab === 'all' ? '' : listTab;
      const res = await apiClient.getPage<any>(`/v1/feedback/projects/${projectId}/submissions`, {
        page: '1', size: '50', status,
      });
      if (res.code === 0) setListData(res.data || []);
    } catch { setListData([]); } finally { setListLoading(false); }
  };

  useEffect(() => { fetchList(); }, [projectId, listTab]);

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    {
      title: '标题', dataIndex: 'title', ellipsis: true,
      render: (v: string, r: any) => <a onClick={() => router.push(`/feedback/${r.id}`)}>{v}</a>,
    },
    {
      title: '类型', dataIndex: 'feedback_type', width: 90,
      render: (v: string) => <TypeBadge type={v} />,
    },
    {
      title: '状态', dataIndex: 'status', width: 90,
      render: (v: string) => <StatusBadge status={v} />,
    },
    {
      title: '投票', dataIndex: 'vote_count', width: 60,
    },
    {
      title: '优先级', dataIndex: 'priority', width: 80,
      render: (v: number) => <Tag color={v > 60 ? 'red' : v > 30 ? 'orange' : 'blue'}>{v}</Tag>,
    },
    { title: '提交时间', dataIndex: 'created_at', width: 160, render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm') },
    {
      title: '操作', width: 150,
      render: (_: any, r: any) => (
        <Button
          type="link"
          size="small"
          onClick={() => {
            setStatusModal({ open: true, id: r.id, currentStatus: r.status });
            updateForm.setFieldsValue({ status: r.status });
          }}
        >
          更新状态
        </Button>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
        <Title level={3} style={{ margin: 0 }}>反馈管理</Title>
        {projects.length > 1 && (
          <Space>
            <span>项目：</span>
            <Select
              value={projectId}
              onChange={setProjectId}
              options={projects.map((p) => ({ value: p.id, label: p.name }))}
              style={{ width: 200 }}
            />
          </Space>
        )}
      </Row>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col xs={12} sm={6}>
          <Card><Statistic title="待审核" value={stats?.pending_review || 0} prefix={<ClockCircleOutlined />} valueStyle={{ color: '#faad14' }} /></Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card><Statistic title="已采纳" value={stats?.accepted || 0} prefix={<CheckCircleOutlined />} valueStyle={{ color: '#52c41a' }} /></Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card><Statistic title="已上线" value={stats?.shipped || 0} prefix={<SendOutlined />} valueStyle={{ color: '#1677ff' }} /></Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card><Statistic title="平均优先级" value={stats?.avg_priority?.toFixed(1) || 0} prefix={<RiseOutlined />} /></Card>
        </Col>
      </Row>

      <Card>
        <Tabs
          activeKey={listTab}
          onChange={setListTab}
          items={[
            { key: 'pending', label: '待审核' },
            { key: 'under_review', label: '审核中' },
            { key: 'accepted', label: '已采纳' },
            { key: 'all', label: '全部' },
          ]}
          style={{ marginBottom: 8 }}
        />
        <Table
          dataSource={listData}
          columns={columns}
          rowKey="id"
          loading={listLoading}
          pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
          size="small"
          scroll={{ x: 800 }}
        />
      </Card>

      <Modal
        title="更新反馈状态"
        open={statusModal?.open}
        onOk={handleUpdateStatus}
        onCancel={() => { setStatusModal(null); updateForm.resetFields(); }}
        confirmLoading={updating}
      >
        <Form form={updateForm} layout="vertical">
          <Form.Item name="status" label="新状态" rules={[{ required: true }]}>
            <Select options={feedbackStatusList} />
          </Form.Item>
          <Form.Item name="reviewer_notes" label="审核备注">
            <TextArea rows={3} placeholder="审核意见（可选）" />
          </Form.Item>
          <Form.Item name="reject_reason" label="拒绝原因">
            <TextArea rows={2} placeholder="如果拒绝，请说明原因" />
          </Form.Item>
          <Form.Item name="assigned_to" label="分配给">
            <Select
              allowClear
              mode="tags"
              maxCount={1}
              placeholder="输入用户ID分配（可选）"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
