'use client';

import { useState } from 'react';
import { Card, Descriptions, Table, Spin, Result, Button, Space, Tag, Modal, Form, Input, Select, message } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeftOutlined, AuditOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';

interface SettlementItem {
  id: number;
  settlement_id: number;
  transaction_type: string;
  transaction_id: string;
  order_no: string;
  order_id?: number;
  sku_id?: number;
  amount: number;
  fee: number;
  net: number;
  quantity: number;
  occurred_at?: string;
  reconciliation_status: string;
  reconciliation_note: string;
  reconciled_at?: string;
  reconciled_by: string;
}

interface Settlement {
  id: number;
  platform_id?: number;
  settlement_no: string;
  period_start?: string;
  period_end?: string;
  currency: string;
  total_revenue: number;
  total_fee: number;
  total_refund: number;
  total_net: number;
  status: string;
  imported_at?: string;
  created_at: string;
}

interface SettlementDetailResponse {
  settlement: Settlement;
  items: SettlementItem[];
}

export default function SettlementDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;
  const queryClient = useQueryClient();

  const [reconcileOpen, setReconcileOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState<SettlementItem | null>(null);
  const [reconcileForm] = Form.useForm();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['settlement', id],
    queryFn: async () => {
      const res = await apiClient.get<SettlementDetailResponse>(`/v1/settlement/${id}`);
      return res.data;
    },
    retry: false,
  });

  const settlement = data?.settlement;
  const items = data?.items || [];

  const reconcileMutation = useMutation({
    mutationFn: async (values: {
      item_id?: number;
      reconciliation_status: string;
      reconciliation_note: string;
      reconciled_by: string;
    }) => {
      return apiClient.post(`/v1/settlement/${id}/reconcile`, values);
    },
    onSuccess: () => {
      message.success('对账成功');
      setReconcileOpen(false);
      reconcileForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ['settlement', id] });
    },
    onError: (err: Error) => {
      message.error(`操作失败: ${err.message}`);
    },
  });

  const handleOpenReconcile = (item: SettlementItem) => {
    setSelectedItem(item);
    reconcileForm.setFieldsValue({
      reconciliation_status: item.reconciliation_status === 'pending' ? 'matched' : item.reconciliation_status,
      reconciliation_note: item.reconciliation_note || '',
      reconciled_by: item.reconciled_by || 'admin',
    });
    setReconcileOpen(true);
  };

  const handleReconcileSubmit = async () => {
    const values = await reconcileForm.validateFields();
    reconcileMutation.mutate({
      item_id: selectedItem?.id,
      reconciliation_status: values.reconciliation_status,
      reconciliation_note: values.reconciliation_note,
      reconciled_by: values.reconciled_by,
    });
  };

  const columns = [
    { title: '交易 ID', dataIndex: 'transaction_id', key: 'transaction_id' },
    { title: '交易类型', dataIndex: 'transaction_type', key: 'transaction_type' },
    { title: '订单号', dataIndex: 'order_no', key: 'order_no' },
    {
      title: '金额',
      dataIndex: 'amount',
      key: 'amount',
      render: (v: number) => `¥${v.toFixed(2)}`,
    },
    {
      title: '费用',
      dataIndex: 'fee',
      key: 'fee',
      render: (v: number) => `¥${v.toFixed(2)}`,
    },
    {
      title: '净额',
      dataIndex: 'net',
      key: 'net',
      render: (v: number) => `¥${v.toFixed(2)}`,
    },
    {
      title: '对账状态',
      dataIndex: 'reconciliation_status',
      key: 'reconciliation_status',
      render: (status: string) => {
        const colors: Record<string, string> = {
          pending: 'gold',
          matched: 'green',
          mismatched: 'red',
        };
        return <Tag color={colors[status] || 'blue'}>{status.toUpperCase()}</Tag>;
      },
    },
    { title: '备注', dataIndex: 'reconciliation_note', key: 'reconciliation_note' },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: SettlementItem) => (
        <Button
          type="link"
          icon={<AuditOutlined />}
          onClick={() => handleOpenReconcile(record)}
        >
          对账
        </Button>
      ),
    },
  ];

  return (
    <PageContainer title="结算单详情">
      <Button
        icon={<ArrowLeftOutlined />}
        onClick={() => router.push('/settlement')}
        style={{ marginBottom: 'var(--space-lg)' }}
      >
        返回列表
      </Button>

      {isLoading ? (
        <Card>
          <div style={{ textAlign: 'center', padding: 'var(--space-3xl)' }}>
            <Spin tip="加载中..." />
          </div>
        </Card>
      ) : isError || !data ? (
        <Card>
          <Result status="info" title="结算单详情" subTitle="暂无详情数据或结算单不存在" />
        </Card>
      ) : (
        <Space direction="vertical" size="middle" style={{ display: 'flex' }}>
          <Card title="基本信息">
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="结算单号">{settlement?.settlement_no}</Descriptions.Item>
              <Descriptions.Item label="平台ID">{settlement?.platform_id || '-'}</Descriptions.Item>
              <Descriptions.Item label="账期开始">
                {settlement?.period_start ? dayjs(settlement.period_start).format('YYYY-MM-DD') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="账期结束">
                {settlement?.period_end ? dayjs(settlement.period_end).format('YYYY-MM-DD') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="币种">{settlement?.currency}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={settlement?.status === 'completed' ? 'green' : 'blue'}>{settlement?.status}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="导入时间">
                {settlement?.imported_at ? dayjs(settlement.imported_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {settlement?.created_at ? dayjs(settlement.created_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="金额汇总">
            <Descriptions bordered column={4} size="small">
              <Descriptions.Item label="总收入">¥{settlement?.total_revenue?.toFixed(2)}</Descriptions.Item>
              <Descriptions.Item label="总费用">¥{settlement?.total_fee?.toFixed(2)}</Descriptions.Item>
              <Descriptions.Item label="总退款">¥{settlement?.total_refund?.toFixed(2)}</Descriptions.Item>
              <Descriptions.Item label="净额合计">
                <span style={{ color: (settlement?.total_net || 0) >= 0 ? '#52c41a' : '#ff4d4f', fontWeight: 'bold' }}>
                  ¥{settlement?.total_net?.toFixed(2)}
                </span>
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="结算明细">
            <Table
              dataSource={items}
              columns={columns}
              rowKey="id"
              pagination={{ pageSize: 10 }}
              size="small"
            />
          </Card>
        </Space>
      )}

      <Modal
        title={`明细对账 — 交易 ID: ${selectedItem?.transaction_id || ''}`}
        open={reconcileOpen}
        onCancel={() => {
          setReconcileOpen(false);
          reconcileForm.resetFields();
        }}
        onOk={handleReconcileSubmit}
        confirmLoading={reconcileMutation.isPending}
        okText="保存对账"
        cancelText="取消"
      >
        <Form form={reconcileForm} layout="vertical">
          <Form.Item
            name="reconciliation_status"
            label="对账结果"
            rules={[{ required: true, message: '请选择对账结果' }]}
          >
            <Select
              options={[
                { label: '匹配 (MATCHED)', value: 'matched' },
                { label: '差异 (MISMATCHED)', value: 'mismatched' },
                { label: '待处理 (PENDING)', value: 'pending' },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="reconciled_by"
            label="对账人"
            rules={[{ required: true, message: '请输入对账人' }]}
          >
            <Input placeholder="对账人姓名" />
          </Form.Item>
          <Form.Item name="reconciliation_note" label="对账备注">
            <Input.TextArea rows={3} placeholder="添加对账差异说明或备注信息..." />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
