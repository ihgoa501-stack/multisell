'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import {
  Button, Card, Col, Descriptions, Form, Input, InputNumber, Modal, Row, Space, Spin, Tag, Timeline, Typography, message,
} from 'antd';
import {
  CheckCircleOutlined, CloseCircleOutlined, DollarOutlined, RollbackOutlined, RobotOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';
import ConfirmDialog from '@/components/ui/ConfirmDialog';

const { Text } = Typography;

const STATUS_COLORS: Record<string, string> = {
  pending: 'orange', approved: 'blue', received: 'cyan', rejected: 'red', refunded: 'green',
};
const STATUS_LABELS: Record<string, string> = {
  pending: '待审核', approved: '已审核', received: '已收货', rejected: '已拒绝', refunded: '已退款',
};

interface ModalActionState { action: 'approve' | 'reject'; }
interface DirectActionState { action: 'receive' | 'refund'; label: string; }

export default function AftersalesDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const qc = useQueryClient();

  const { data, isLoading } = useQuery<Record<string, unknown>>({
    queryKey: ['aftersales', id],
    queryFn: async () => {
      const res = await apiClient.get<Record<string, unknown>>(`/v1/aftersales/${id}`);
      if (!res.data) throw new Error('Not found');
      return res.data;
    },
  });

  const [modalAction, setModalAction] = useState<ModalActionState | null>(null);
  const [directAction, setDirectAction] = useState<DirectActionState | null>(null);
  const [form] = Form.useForm();

  const actionMutation = useMutation({
    mutationFn: async ({ action, values }: { action: string; values: Record<string, unknown> }) =>
      apiClient.post(`/v1/aftersales/${id}/${action}`, values),
    onSuccess: () => {
      message.success('操作成功');
      setModalAction(null);
      setDirectAction(null);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['aftersales', id] });
    },
    onError: (e: Error) => message.error(`操作失败: ${e.message}`),
  });

  const handleModalOk = async () => {
    if (!modalAction) return;
    actionMutation.mutate({ action: modalAction.action, values: await form.validateFields() });
  };

  const status = (data?.status as string) || 'pending';
  const timelineItems = [
    ...(data?.created_at ? [{ color: 'blue' as const, children: <><Text strong>创建售后单</Text><Text type="secondary" style={{ marginLeft: 8 }}>{fmtDate(data.created_at)}</Text></> }] : []),
    ...(data?.approved_at ? [{ color: 'green' as const, children: <><Text strong>审核通过</Text><Text type="secondary" style={{ marginLeft: 8 }}>{fmtDate(data.approved_at)}</Text><br /><Text type="secondary">验货: {(data.inspection_result as string) || '-'}</Text></> }] : []),
    ...(data?.rejected_at ? [{ color: 'red' as const, children: <><Text strong>审核拒绝</Text><Text type="secondary" style={{ marginLeft: 8 }}>{fmtDate(data.rejected_at)}</Text><br /><Text type="secondary">原因: {(data.rejection_reason as string) || '-'}</Text></> }] : []),
    ...(data?.received_at ? [{ color: 'cyan' as const, children: <><Text strong>确认收货</Text><Text type="secondary" style={{ marginLeft: 8 }}>{fmtDate(data.received_at)}</Text></> }] : []),
    ...(data?.refunded_at ? [{ color: 'green' as const, children: <><Text strong>已退款</Text><Text type="secondary" style={{ marginLeft: 8 }}>{fmtDate(data.refunded_at)}</Text></> }] : []),
  ];

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  if (!data) return <PageContainer title="售后详情"><Text type="danger">售后单未找到</Text></PageContainer>;

  return (
    <PageContainer
      title={`售后单 #${id}`}
      extra={<Button onClick={() => router.push('/aftersales')}>返回列表</Button>}
    >
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col>
          <Tag color={STATUS_COLORS[status] || 'default'} style={{ fontSize: 14, padding: '4px 12px' }}>
            {STATUS_LABELS[status] || status}
          </Tag>
        </Col>
        <Col flex="auto">
          <Space>
            {status === 'pending' && (
              <>
                <Button type="primary" icon={<CheckCircleOutlined />} onClick={() => { form.resetFields(); setModalAction({ action: 'approve' }); }}>审核通过</Button>
                <Button danger icon={<CloseCircleOutlined />} onClick={() => { form.resetFields(); setModalAction({ action: 'reject' }); }}>审核拒绝</Button>
              </>
            )}
            {status === 'approved' && (
              <Button type="primary" icon={<RollbackOutlined />} onClick={() => setDirectAction({ action: 'receive', label: '确认收货' })}>确认收货</Button>
            )}
            {status === 'received' && (
              <Button type="primary" icon={<DollarOutlined />} onClick={() => setDirectAction({ action: 'refund', label: '执行退款' })}>执行退款</Button>
            )}
          </Space>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col span={16}>
          <Card title="基本信息" size="small">
            <Descriptions column={2} size="small">
              <Descriptions.Item label="售后单ID">{data.id as number}</Descriptions.Item>
              <Descriptions.Item label="订单ID">{(data.order_id as number) || '-'}</Descriptions.Item>
              <Descriptions.Item label="Item ID">{(data.item_id as number) ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="SKU ID">{(data.sku_id as number) ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="退货数量">{data.return_quantity as number}</Descriptions.Item>
              <Descriptions.Item label="退款金额">{fmtMoney(data.refund_amount)}</Descriptions.Item>
              <Descriptions.Item label="原因" span={2}>{(data.reason as string) || '-'}</Descriptions.Item>
              <Descriptions.Item label="验货结果" span={2}>{(data.inspection_result as string) || '-'}</Descriptions.Item>
              <Descriptions.Item label="拒绝原因" span={2}>{(data.rejection_reason as string) || '-'}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col span={8}>
          <Card title="审批信息" size="small">
            <Descriptions column={1} size="small">
              <Descriptions.Item label="创建人">{(data.created_by as string) || '-'}</Descriptions.Item>
              <Descriptions.Item label="审核人">{(data.approved_by as string) || '-'}</Descriptions.Item>
              <Descriptions.Item label="收货人">{(data.received_by as string) || '-'}</Descriptions.Item>
              <Descriptions.Item label="退款人">{(data.refunded_by as string) || '-'}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={12}>
          <Card title="处理时间线" size="small">
            {timelineItems.length > 0 ? (
              <Timeline items={timelineItems} />
            ) : (
              <Text type="secondary">暂无事件</Text>
            )}
          </Card>
        </Col>
        <Col span={12}>
          <Card title={<><RobotOutlined /> Agent 决策</>} size="small">
            <Text type="secondary">当前售后单可由 Agent 分析并提供处理建议</Text>
            <div style={{ marginTop: 12 }}>
              <Space>
                <Button
                  icon={<RobotOutlined />}
                  loading={actionMutation.isPending}
                  onClick={() => actionMutation.mutate({ action: 'auto-decide', values: {} })}
                >
                  Agent 分析建议
                </Button>
                <Button icon={<RobotOutlined />} onClick={() => window.open('/xiaoq', '_blank')}>
                  AI 咨询
                </Button>
              </Space>
            </div>
          </Card>
        </Col>
      </Row>

      <Modal
        title={modalAction?.action === 'approve' ? `审核通过 #${id}` : `审核拒绝 #${id}`}
        open={!!modalAction}
        onCancel={() => { setModalAction(null); form.resetFields(); }}
        onOk={handleModalOk}
        confirmLoading={actionMutation.isPending}
        okText="确认" cancelText="取消" width={500} destroyOnClose
      >
        <Form form={form} layout="vertical" preserve={false}>
          {modalAction?.action === 'approve' && (
            <>
              <Form.Item name="inspection_result" label="验货结果" rules={[{ required: true, message: '请输入验货结果' }]}>
                <Input.TextArea rows={4} placeholder="描述退货商品的检查结果..." />
              </Form.Item>
              <Form.Item name="refund_amount" label="退款金额" rules={[{ required: true, message: '请输入退款金额' }]}>
                <InputNumber style={{ width: '100%' }} min={0} precision={2} prefix="¥" placeholder="输入退款金额" />
              </Form.Item>
            </>
          )}
          {modalAction?.action === 'reject' && (
            <Form.Item name="rejection_reason" label="拒绝原因" rules={[{ required: true, message: '请输入拒绝原因' }]}>
              <Input.TextArea rows={4} placeholder="描述拒绝的原因..." />
            </Form.Item>
          )}
        </Form>
      </Modal>

      <ConfirmDialog
        open={!!directAction}
        title={directAction?.label ?? ''}
        content={`确定要${directAction?.label ?? ''}吗？`}
        okText="确认" cancelText="取消"
        confirmLoading={actionMutation.isPending}
        onOk={() => { if (directAction) actionMutation.mutate({ action: directAction.action, values: {} }); }}
        onCancel={() => setDirectAction(null)}
      />
    </PageContainer>
  );
}
