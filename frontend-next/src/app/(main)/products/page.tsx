'use client';

import { Alert, Tag } from 'antd';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

const STATUS_MAP: Record<string, { label: string; color: string }> = {
  '0': { label: '草稿', color: 'default' },
  '1': { label: '上架', color: 'success' },
  '2': { label: '下架', color: 'warning' },
};

export default function ProductsPage() {
  return (
    <>
      <Alert
        type="info"
        showIcon
        title="商品档案不提供直接发布"
        description="外部发布必须从 1688 受控货源工作台进入，并经过商品机会授权、草稿内容冻结、Owner 独立审批和显式执行。"
        style={{ marginBottom: 16 }}
      />
      <CrudListPage
        resource="/product-master"
        title="商品"
        singular="商品"
        searchPlaceholder="搜索商品名称 / 编码 / 副标题..."
        columns={[
          { title: 'ID', dataIndex: 'id', width: 70 },
          { title: '商品名称', dataIndex: 'name', width: 200 },
          { title: '副标题', dataIndex: 'subtitle', width: 200 },
          { title: '品牌ID', dataIndex: 'brand_id', width: 90 },
          { title: '分类ID', dataIndex: 'category_id', width: 90 },
          { title: '单位', dataIndex: 'unit', width: 80 },
          { title: '状态', dataIndex: 'status', width: 100,
            render: (s: unknown) => {
              const v = STATUS_MAP[String(s)] || { label: String(s), color: 'default' };
              return <Tag color={v.color}>{v.label}</Tag>;
            },
          },
          { title: '货品类型', dataIndex: 'cargo_type', width: 110 },
          { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
        ]}
        fields={[
          { name: 'name', label: '商品名称', required: true },
          { name: 'subtitle', label: '副标题' },
          { name: 'brand_id', label: '品牌ID', type: 'number' },
          { name: 'category_id', label: '分类ID', type: 'number' },
          { name: 'unit', label: '单位' },
          { name: 'status', label: '状态', initialValue: '1' },
          { name: 'main_image', label: '主图URL' },
          { name: 'cargo_type', label: '货品类型' },
        ]}
      />
    </>
  );
}
