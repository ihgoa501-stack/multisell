'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function ShippingPage() {
  return (
    <CrudListPage
      resource="/shipping/providers"
      title="物流供应商"
      singular="物流供应商"
      searchPlaceholder="搜索物流商名称 / 编码..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '物流商名称', dataIndex: 'name', width: 200 },
        { title: '编码', dataIndex: 'code', width: 120 },
        { title: '联系人', dataIndex: 'contact', width: 120 },
        { title: '电话', dataIndex: 'phone', width: 140 },
        { title: '备注', dataIndex: 'remark', width: 200 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'name', label: '物流商名称', required: true },
        { name: 'code', label: '编码' },
        { name: 'contact', label: '联系人' },
        { name: 'phone', label: '电话' },
        { name: 'remark', label: '备注', type: 'textarea' },
        { name: 'status', label: '状态', initialValue: 'active' },
      ]}
    />
  );
}
