'use client';

import { useState } from 'react';
import { Descriptions, Table, Tabs, Tag, Button, Modal, Form, Select, message, Space, Typography } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import { getCurrentOperator } from '@/lib/user';
import PageContainer from '@/components/ui/PageContainer';
import type { Result, PageResult } from '@/types/api';

const { Text } = Typography;

interface ApprovalRequest {
  id: number;
  product_id: number;
  request_type: string;
  requester: string;
  reviewer: string;
  status: string;
  old_value: string;
  new_value: string;
  reason: string;
  review_note: string;
  expires_at: string | null;
  created_at: string;
  updated_at: string;
  target_type?: string;
  target_id?: number;
  risk_level?: string;
}

interface ApprovalStats {
  pending_count: number;
  approved_count: number;
  rejected_count: number;
  total_count: number;
  avg_review_hours: number;
  escalated_count: number;
}

const REQUEST_TYPE_MAP: Record<string, string> = {
  publish: '上架',
  price_change: '改价',
  delist: '下架',
  content_update: '内容更新',
};

const STATUS_MAP: Record<string, { label: string; color: string }> = {
  pending: { label: '待审批', color: 'orange' },
  approved: { label: '已通过', color: 'green' },
  rejected: { label: '已驳回', color: 'red' },
};

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function fmtType(t: string) {
  return REQUEST_TYPE_MAP[t] || t;
}

function fmtStatus(s: string) {
  const entry = STATUS_MAP[s];
  if (!entry) return <Tag>{s}</Tag>;
  return <Tag color={entry.color}>{entry.label}</Tag>;
}

export default function ApprovalPage() {
  const queryClient = useQueryClient();
  const [tabKey, setTabKey] = useState('pending');
  const [page, setPage] = useState(1);
  const [reviewModalOpen, setReviewModalOpen] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState<ApprovalRequest | null>(null);
  const [form] = Form.useForm();

  const statusFilter = tabKey === 'all' ? '' : tabKey === 'pending' ? 'pending' : 'approved,rejected';
  const queryKey = ['approval', tabKey, page];

  const { data, isLoading } = useQuery<PageResult<ApprovalRequest>>({
    queryKey,
    queryFn: () =>
      apiClient.getPage('/v1/approval', {
        page: String(page),
        size: '20',
        ...(statusFilter ? { status: statusFilter } : {}),
      }),
  });

  const { data: statsData } = useQuery<Result<ApprovalStats>>({
    queryKey: ['approval-stats'],
    queryFn: () => apiClient.get('/v1/approval/stats'),
    refetchInterval: 30000,
  });

  const reviewMutation = useMutation({
    mutationFn: async ({ id, action, reviewNote }: { id: number; action: string; reviewNote: string }) => {
      return apiClient.put(`/v1/approval/${id}/review`, {
        action,
        reviewer: getCurrentOperator(),
        review_note: reviewNote,
      });
    },
    onSuccess: () => {
      message.success('审批完成');
      setReviewModalOpen(false);
      setSelectedRequest(null);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['approval'] });
      queryClient.invalidateQueries({ queryKey: ['approval-stats'] });
      queryClient.invalidateQueries({ queryKey: ['owner-risk-summary'] });
      queryClient.invalidateQueries({ queryKey: ['owner-suggestions'] });
      queryClient.invalidateQueries({ queryKey: ['owner-pending-approvals'] });
    },
    onError: (err: Error) => {
      message.error(`审批失败: ${err.message}`);
    },
  });

  const handleReview = (record: ApprovalRequest) => {
    setSelectedRequest(record);
    setReviewModalOpen(true);
  };

  const handleReviewSubmit = (action: string) => {
    if (!selectedRequest) return;
    const reviewNote = form.getFieldValue('review_note') || '';
    reviewMutation.mutate({ id: selectedRequest.id, action, reviewNote });
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '商品ID', dataIndex: 'product_id', width: 90 },
    { title: '类型', dataIndex: 'request_type', width: 100, render: fmtType },
    { title: '状态', dataIndex: 'status', width: 90, render: fmtStatus },
    { title: '发起人', dataIndex: 'requester', width: 100 },
    { title: '原因', dataIndex: 'reason', ellipsis: true },
    {
      title: '旧值',
      dataIndex: 'old_value',
      width: 120,
      ellipsis: true,
      render: (v: string) => v || '-',
    },
    {
      title: '新值',
      dataIndex: 'new_value',
      width: 120,
      ellipsis: true,
      render: (v: string) => v || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 160,
      render: formatDate,
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record: ApprovalRequest) => {
        if (record.status !== 'pending') {
          return (
            <Text type="secondary">
              {record.status === 'approved' ? '已通过' : '已驳回'}
            </Text>
          );
        }
        return (
          <Button
            type="primary"
            size="small"
            icon={<CheckCircleOutlined />}
            onClick={() => handleReview(record)}
          >
            审批
          </Button>
        );
      },
    },
  ];

  const tabItems = [
    {
      key: 'pending',
      label: (
        <span>
          <ClockCircleOutlined /> 待审批
          {statsData?.data?.pending_count != null && statsData.data.pending_count > 0 && (
            <Tag color="orange" style={{ marginLeft: 4 }}>
              {statsData.data.pending_count}
            </Tag>
          )}
        </span>
      ),
    },
    {
      key: 'approved',
      label: (
        <span>
          <CheckCircleOutlined /> 已审批
        </span>
      ),
    },
    {
      key: 'all',
      label: <span>全部</span>,
    },
  ];

  return (
    <PageContainer title="审批工作台">
      <Tabs activeKey={tabKey} onChange={(k) => { setTabKey(k); setPage(1); }} items={tabItems} />

      <Table
        dataSource={data?.data || []}
        columns={columns}
        rowKey="id"
        loading={isLoading}
        pagination={{
          current: page,
          pageSize: 20,
          total: data?.total || 0,
          onChange: setPage,
          showSizeChanger: false,
          showTotal: (t) => `共 ${t} 条`,
        }}
        size="middle"
      />

      <Modal
        title={`审批请求 #${selectedRequest?.id || ''}`}
        open={reviewModalOpen}
        onCancel={() => {
          setReviewModalOpen(false);
          setSelectedRequest(null);
          form.resetFields();
        }}
        footer={
          <Space>
            <Button
              danger
              icon={<CloseCircleOutlined />}
              loading={reviewMutation.isPending}
              onClick={() => handleReviewSubmit('reject')}
            >
              拒绝，保持阻塞
            </Button>
            <Button
              type="primary"
              icon={<CheckCircleOutlined />}
              loading={reviewMutation.isPending}
              onClick={() => handleReviewSubmit('approve')}
              style={{ backgroundColor: '#52c41a', borderColor: '#52c41a' }}
            >
              批准，允许执行任务
            </Button>
          </Space>
        }
      >
        {selectedRequest && (
          <div style={{ marginBottom: 16 }}>
            <Descriptions bordered size="small" column={1}>
              <Descriptions.Item label="业务动作">{fmtType(selectedRequest.request_type)}</Descriptions.Item>
              <Descriptions.Item label="风险等级">
                <Tag color={selectedRequest.risk_level === 'high' ? 'red' : selectedRequest.risk_level === 'medium' ? 'orange' : 'green'}>
                  {selectedRequest.risk_level || '未标记'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="目标对象">
                {selectedRequest.target_type || '-'} #{selectedRequest.target_id || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="发起人">{selectedRequest.requester}</Descriptions.Item>
              <Descriptions.Item label="Agent 理由">{selectedRequest.reason || '无'}</Descriptions.Item>
              <Descriptions.Item label="批准后会发生什么">
                系统允许对应刊登任务进入执行流程；审批本身不会直接改价、改库存、改订单或发布到真实平台。
              </Descriptions.Item>
              <Descriptions.Item label="不批准会怎样">
                刊登任务保持阻塞，Owner 可回到候选商品或任务详情补充信息后重新评估。
              </Descriptions.Item>
            </Descriptions>
          </div>
        )}
        <Form form={form} layout="vertical">
          <Form.Item name="review_note" label="审批备注">
            <Select
              mode="tags"
              placeholder="选择或输入审批备注"
              maxCount={1}
              options={[
                { label: '已确认，同意', value: '已确认，同意' },
                { label: '需要更多信息', value: '需要更多信息' },
                { label: '不符合要求', value: '不符合要求' },
              ]}
              onChange={(v: string[]) => form.setFieldsValue({ review_note: v[0] || '' })}
            />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
