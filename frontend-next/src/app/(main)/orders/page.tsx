'use client';

import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

export default function OrdersPage() {
  return (
    <CrudListPage
      resource="/order"
      title="订单"
      singular="订单"
      searchPlaceholder="搜索订单号 / 收件人 / 运单号..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '订单号', dataIndex: 'order_no', width: 180 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '收件人', dataIndex: 'recipient_name', width: 120 },
        { title: '运单号', dataIndex: 'tracking_number', width: 160 },
        { title: '实付金额', dataIndex: 'pay_amount', width: 120, render: fmtMoney },
        { title: '利润', dataIndex: 'profit_amount', width: 120, render: fmtMoney },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'order_no', label: '订单号', required: true },
        { name: 'status', label: '状态', initialValue: 'pending' },
        { name: 'tracking_number', label: '运单号' },
        { name: 'recipient_name', label: '收件人' },
        { name: 'recipient_phone', label: '联系电话' },
        { name: 'shipping_address', label: '收货地址', type: 'textarea' },
        { name: 'total_amount', label: '商品总额', type: 'number' },
        { name: 'shipping_fee', label: '运费', type: 'number' },
        { name: 'pay_amount', label: '实付金额', type: 'number' },
        { name: 'payment_method', label: '支付方式' },
        { name: 'remark', label: '备注', type: 'textarea' },
      ]}
      editable={false}
    />
  );
}
