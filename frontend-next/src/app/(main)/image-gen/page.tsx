'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';

export default function ImageGenPage() {
  return (
    <CrudListPage
      resource="/image-gen"
      title="AI 图片生成"
      singular="生成任务"
      searchPlaceholder="搜索 prompt..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '商品 ID', dataIndex: 'product_id', width: 110 },
        { title: 'Prompt', dataIndex: 'prompt' },
        { title: '状态', dataIndex: 'status', width: 110 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'product_id', label: '商品 ID' },
        { name: 'prompt', label: 'Prompt', type: 'textarea', required: true },
        { name: 'style', label: '风格' },
        { name: 'size', label: '尺寸' },
      ]}
    />
  );
}
