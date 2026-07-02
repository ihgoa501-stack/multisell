'use client';

import { Tabs } from 'antd';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

const RULE_TYPE_OPTIONS = [
  { label: '百分比', value: 'percentage' },
  { label: '固定数量', value: 'fixed' },
  { label: '优先级', value: 'priority' },
  { label: '备份', value: 'backup' },
];

const ALLOCATION_TYPE_OPTIONS = [
  { label: '运费', value: 'shipping' },
  { label: '关税', value: 'customs' },
  { label: '管理费', value: 'overhead' },
  { label: '物流费', value: 'freight' },
];

const ALLOCATION_METHOD_OPTIONS = [
  { label: '按重量', value: 'weight' },
  { label: '按体积', value: 'volume' },
  { label: '按价值', value: 'value' },
  { label: '平均分摊', value: 'equal' },
];

const BINARY_STATUS_OPTIONS = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
];

export default function AllocationPage() {
  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <h1
        style={{
          fontFamily: 'var(--ds)',
          fontWeight: 700,
          fontSize: 'var(--text-h1)',
          color: 'var(--t1)',
          margin: '0 0 16px 0',
        }}
      >
        分摊设置
      </h1>
      <Tabs
        defaultActiveKey="warehouses"
        items={[
          {
            key: 'warehouses',
            label: '仓库管理',
            children: (
              <CrudListPage
                resource="/allocation/warehouses"
                title="仓库管理"
                singular="仓库"
                searchPlaceholder="搜索仓库名称/代码..."
                columns={[
                  { title: 'ID', dataIndex: 'id', width: 70 },
                  { title: '名称', dataIndex: 'name' },
                  { title: '代码', dataIndex: 'code', width: 130 },
                  { title: '地址', dataIndex: 'address' },
                  { title: '联系人', dataIndex: 'contact', width: 100 },
                  { title: '状态', dataIndex: 'status', width: 80 },
                ]}
                fields={[
                  { name: 'name', label: '名称', required: true },
                  { name: 'code', label: '代码' },
                  { name: 'address', label: '地址' },
                  { name: 'contact', label: '联系人' },
                  { name: 'phone', label: '电话' },
                  { name: 'status', label: '状态', type: 'select', options: BINARY_STATUS_OPTIONS, initialValue: 1 },
                ]}
              />
            ),
          },
          {
            key: 'rules',
            label: '分摊规则',
            children: (
              <CrudListPage
                resource="/allocation/rules"
                title="分摊规则"
                singular="规则"
                searchPlaceholder="搜索规则名称..."
                columns={[
                  { title: 'ID', dataIndex: 'id', width: 70 },
                  { title: '名称', dataIndex: 'name' },
                  { title: '类型', dataIndex: 'rule_type', width: 100 },
                  { title: '仓库', dataIndex: 'warehouse_id', width: 80 },
                  { title: '优先级', dataIndex: 'priority', width: 80 },
                  { title: '分摊比例', dataIndex: 'allocation_pct', width: 100, render: (v: unknown) => (v != null ? `${v}%` : '-') },
                  { title: 'SKU', dataIndex: 'sku_id', width: 80, render: (v: unknown) => (v != null && Number(v) > 0 ? String(v) : '-') },
                  { title: '状态', dataIndex: 'status', width: 80 },
                ]}
                fields={[
                  { name: 'name', label: '名称', required: true },
                  { name: 'rule_type', label: '类型', type: 'select', required: true, options: RULE_TYPE_OPTIONS },
                  { name: 'warehouse_id', label: '仓库ID', type: 'number', required: true },
                  { name: 'priority', label: '优先级', type: 'number', initialValue: 0 },
                  { name: 'sku_id', label: 'SKU ID', type: 'number' },
                  { name: 'allocation_pct', label: '分摊比例(%)', type: 'number' },
                  { name: 'allocation_qty', label: '分摊数量', type: 'number' },
                  { name: 'status', label: '状态', type: 'select', options: BINARY_STATUS_OPTIONS, initialValue: 1 },
                ]}
              />
            ),
          },
          {
            key: 'batches',
            label: '成本分摊批次',
            children: (
              <CrudListPage
                resource="/allocation/cost/batches"
                title="成本分摊批次"
                singular="批次"
                searchPlaceholder="搜索..."
                columns={[
                  { title: 'ID', dataIndex: 'id', width: 70 },
                  { title: '分摊类型', dataIndex: 'allocation_type', width: 100 },
                  { title: '方法', dataIndex: 'allocation_method', width: 90 },
                  { title: '总金额', dataIndex: 'total_amount', width: 110 },
                  { title: '币种', dataIndex: 'currency', width: 70 },
                  { title: '行数', dataIndex: 'row_count', width: 70 },
                  { title: '状态', dataIndex: 'status', width: 90 },
                  { title: '创建人', dataIndex: 'created_by', width: 100 },
                  { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
                ]}
                fields={[
                  { name: 'allocation_type', label: '分摊类型', type: 'select', required: true, options: ALLOCATION_TYPE_OPTIONS },
                  { name: 'allocation_method', label: '分摊方法', type: 'select', required: true, options: ALLOCATION_METHOD_OPTIONS },
                  { name: 'total_amount', label: '总金额', type: 'number', required: true },
                  { name: 'currency', label: '币种', initialValue: 'CNY' },
                  { name: 'source_filename', label: '源文件名' },
                  { name: 'created_by', label: '创建人' },
                ]}
              />
            ),
          },
        ]}
      />
    </div>
  );
}
