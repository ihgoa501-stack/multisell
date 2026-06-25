'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function SuppliersPage() {
  return (
    <CrudListPage
      resource="/suppliers"
      title="供应商"
      singular="供应商"
      searchPlaceholder="搜索供应商名称 / 联系人 / 电话..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '供应商名称', dataIndex: 'name', width: 200 },
        { title: '联系人', dataIndex: 'contact', width: 120 },
        { title: '电话', dataIndex: 'phone', width: 140 },
        { title: '邮箱', dataIndex: 'email', width: 180 },
        { title: '地址', dataIndex: 'address', width: 220 },
        { title: '备注', dataIndex: 'remark', width: 180 },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'name', label: '供应商名称', required: true },
        { name: 'contact', label: '联系人' },
        { name: 'phone', label: '电话' },
        { name: 'email', label: '邮箱' },
        { name: 'address', label: '地址', type: 'textarea' },
        { name: 'remark', label: '备注', type: 'textarea' },
        { name: 'status', label: '状态', initialValue: 'active' },
      ]}
    />
  );
}
