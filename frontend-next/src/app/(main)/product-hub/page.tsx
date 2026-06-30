'use client';

import { Tag } from 'antd';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

const LIFECYCLE_COLORS: Record<string, string> = {
  idea: 'default',
  researching: 'blue',
  sampling: 'orange',
  approved: 'cyan',
  costed: 'geekblue',
  ready_to_list: 'purple',
  listed: 'processing',
  active: 'success',
  sunset: 'warning',
  archived: 'default',
};

const LIFECYCLE_LABELS: Record<string, string> = {
  idea: '创意',
  researching: '调研中',
  sampling: '打样中',
  approved: '已确认',
  costed: '已核算成本',
  ready_to_list: '待上架',
  listed: '已上架',
  active: '销售中',
  sunset: '衰退中',
  archived: '已归档',
};

export default function ProductHubPage() {
  return (
    <CrudListPage
      resource="/v1/product-hub"
      title="产品档案"
      singular="产品"
      columns={[
        { title: '编号', dataIndex: 'product_code', width: 140 },
        { title: '产品名称', dataIndex: 'name', width: 250 },
        {
          title: '业务模式',
          dataIndex: 'business_model',
          width: 120,
          render: (v: unknown) =>
            v === 'oem' ? 'OEM' : v === 'odm' ? 'ODM' : v === 'catalog' ? '选品采购' : (v as string) || '-',
        },
        {
          title: '生命周期',
          dataIndex: 'lifecycle_status',
          width: 120,
          render: (v: unknown) => (
            <Tag color={LIFECYCLE_COLORS[v as string] || 'default'}>
              {LIFECYCLE_LABELS[v as string] || (v as string) || '-'}
            </Tag>
          ),
        },
        { title: '目标市场', dataIndex: 'target_market', width: 120 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      filters={[
        {
          key: 'lifecycle_status',
          label: '生命周期',
          options: Object.entries(LIFECYCLE_LABELS).map(([k, v]) => ({ label: v, value: k })),
        },
      ]}
      fields={[
        { name: 'name', label: '产品名称', required: true },
        { name: 'product_code', label: '产品编号' },
        { name: 'target_market', label: '目标市场' },
        { name: 'business_model', label: '业务模式', type: 'select', options: [
          { label: 'OEM', value: 'oem' },
          { label: 'ODM', value: 'odm' },
          { label: '选品采购', value: 'catalog' },
          { label: '自有品牌', value: 'private_label' },
        ]},
        { name: 'description', label: '描述', type: 'textarea' },
      ]}
    />
  );
}
