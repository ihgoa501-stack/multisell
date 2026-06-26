'use client';

import { useCallback, useState } from 'react';
import { Button, Form, Input, InputNumber, Modal, Space, message } from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DollarOutlined,
  RollbackOutlined,
} from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';
import ConfirmDialog from '@/components/ui/ConfirmDialog';

// ---------- Types ----------

interface ModalActionState {
  record: Record<string, unknown>;
  action: 'approve' | 'reject';
}

interface DirectActionState {
  record: Record<string, unknown>;
  action: 'receive' | 'refund';
  label: string;
}

// ---------- Action config per status ----------

interface ActionConfig {
  key: 'approve' | 'reject' | 'receive' | 'refund';
  label: string;
  icon: React.ReactNode;
  type?: 'primary' | 'default';
  danger?: boolean;
  /** true = opens a modal form; false = direct call with confirm dialog */
  modal: boolean;
}

const STATUS_ACTIONS: Record<string, ActionConfig[]> = {
  pending: [
    {
      key: 'approve',
      label: '审核通过',
      icon: <CheckCircleOutlined />,
      type: 'primary',
      modal: true,
    },
    {
      key: 'reject',
      label: '审核拒绝',
      icon: <CloseCircleOutlined />,
      danger: true,
      modal: true,
    },
  ],
  approved: [
    {
      key: 'receive',
      label: '确认收货',
      icon: <RollbackOutlined />,
      type: 'primary',
      modal: false,
    },
  ],
  received: [
    {
      key: 'refund',
      label: '执行退款',
      icon: <DollarOutlined />,
      type: 'primary',
      modal: false,
    },
  ],
  rejected: [],
  refunded: [],
};

// ---------- Page ----------

export default function AftersalesPage() {
  const qc = useQueryClient();

  // Modal action state (approve / reject)
  const [modalAction, setModalAction] = useState<ModalActionState | null>(null);
  const [form] = Form.useForm();

  // Direct-action confirm dialog (receive / refund)
  const [directAction, setDirectAction] = useState<DirectActionState | null>(null);

  // ---------- Shared mutation for all workflow actions ----------

  const actionMutation = useMutation({
    mutationFn: async ({
      id,
      action,
      values,
    }: {
      id: number;
      action: string;
      values: Record<string, unknown>;
    }) => {
      return apiClient.post(`/v1/aftersales/${id}/${action}`, values);
    },
    onSuccess: () => {
      message.success('操作成功');
      setModalAction(null);
      setDirectAction(null);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['crud', '/aftersales'] });
    },
    onError: (e: Error) => message.error(`操作失败: ${e.message}`),
  });

  // ---------- Handlers ----------

  const handleAction = useCallback(
    (record: Record<string, unknown>, config: ActionConfig) => {
      if (config.modal) {
        form.resetFields();
        setModalAction({ record, action: config.key as 'approve' | 'reject' });
      } else {
        setDirectAction({
          record,
          action: config.key as 'receive' | 'refund',
          label: config.label,
        });
      }
    },
    [form],
  );

  const handleModalOk = async () => {
    if (!modalAction) return;
    const values = await form.validateFields();
    actionMutation.mutate({
      id: modalAction.record.id as number,
      action: modalAction.action,
      values,
    });
  };

  const handleConfirmDirect = () => {
    if (!directAction) return;
    actionMutation.mutate({
      id: directAction.record.id as number,
      action: directAction.action,
      values: {},
    });
  };

  const getModalTitle = () => {
    if (!modalAction) return '';
    const id = modalAction.record.id;
    switch (modalAction.action) {
      case 'approve':
        return `审核通过 #${id}`;
      case 'reject':
        return `审核拒绝 #${id}`;
    }
  };

  // ---------- Render row actions ----------

  const renderRowActions = useCallback(
    (record: Record<string, unknown>) => {
      const status = record.status as string;
      const actions = STATUS_ACTIONS[status] ?? [];
      if (actions.length === 0) return null;

      return (
        <Space size="small">
          {actions.map((act) => (
            <Button
              key={act.key}
              size="small"
              type={act.type ?? 'default'}
              danger={act.danger}
              icon={act.icon}
              onClick={() => handleAction(record, act)}
            >
              {act.label}
            </Button>
          ))}
        </Space>
      );
    },
    [handleAction],
  );

  // ---------- Render ----------

  return (
    <>
      <CrudListPage
        resource="/aftersales"
        title="售后"
        singular="售后单"
        searchPlaceholder="搜索订单ID / SKU ID / 原因..."
        columns={[
          { title: 'ID', dataIndex: 'id', width: 70 },
          { title: '订单ID', dataIndex: 'order_id', width: 100 },
          { title: 'SKU ID', dataIndex: 'sku_id', width: 90 },
          { title: '退货数量', dataIndex: 'return_quantity', width: 100 },
          { title: '原因', dataIndex: 'reason', width: 200 },
          {
            title: '退款金额',
            dataIndex: 'refund_amount',
            width: 120,
            render: fmtMoney,
          },
          { title: '状态', dataIndex: 'status', width: 110 },
          {
            title: '创建时间',
            dataIndex: 'created_at',
            width: 160,
            render: fmtDate,
          },
        ]}
        fields={[
          { name: 'order_id', label: '订单ID', type: 'number', required: true },
          { name: 'sku_id', label: 'SKU ID', type: 'number', required: true },
          { name: 'return_quantity', label: '退货数量', type: 'number', initialValue: 1 },
          { name: 'reason', label: '原因', type: 'textarea', required: true },
          { name: 'refund_amount', label: '退款金额', type: 'number' },
          { name: 'status', label: '状态', initialValue: 'pending' },
        ]}
        renderRowActions={renderRowActions}
      />

      {/* Approve / Reject modal */}
      <Modal
        title={getModalTitle()}
        open={!!modalAction}
        onCancel={() => {
          setModalAction(null);
          form.resetFields();
        }}
        onOk={handleModalOk}
        confirmLoading={actionMutation.isPending}
        okText="确认"
        cancelText="取消"
        width={500}
        destroyOnClose
      >
        <Form form={form} layout="vertical" preserve={false}>
          {modalAction?.action === 'approve' && (
            <>
              <Form.Item
                name="inspection_result"
                label="验货结果"
                rules={[{ required: true, message: '请输入验货结果' }]}
              >
                <Input.TextArea
                  rows={4}
                  placeholder="描述退货商品的检查结果..."
                />
              </Form.Item>
              <Form.Item
                name="refund_amount"
                label="退款金额"
                rules={[{ required: true, message: '请输入退款金额' }]}
              >
                <InputNumber
                  style={{ width: '100%' }}
                  min={0}
                  precision={2}
                  prefix="¥"
                  placeholder="输入退款金额"
                />
              </Form.Item>
            </>
          )}
          {modalAction?.action === 'reject' && (
            <Form.Item
              name="rejection_reason"
              label="拒绝原因"
              rules={[{ required: true, message: '请输入拒绝原因' }]}
            >
              <Input.TextArea rows={4} placeholder="描述拒绝的原因..." />
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* Direct action confirm dialog (receive / refund) */}
      <ConfirmDialog
        open={!!directAction}
        title={directAction?.label ?? ''}
        content={`确定要${directAction?.label ?? ''}吗？`}
        okText="确认"
        cancelText="取消"
        confirmLoading={actionMutation.isPending}
        onOk={handleConfirmDirect}
        onCancel={() => setDirectAction(null)}
      />
    </>
  );
}
