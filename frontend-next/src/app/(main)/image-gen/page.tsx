'use client';

import { Button } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';
import { Image } from 'antd';

function renderImageUrls(v: unknown): React.ReactNode {
  if (!v) return '-';
  try {
    const urls = typeof v === 'string' ? JSON.parse(v) : v;
    if (Array.isArray(urls) && urls.length > 0) {
      return <Image src={urls[0]} alt="" width={60} height={60} style={{ objectFit: 'cover', borderRadius: 4 }} preview={{ mask: `共${urls.length}张` }} />;
    }
  } catch { /* ignore parse errors */ }
  return '-';
}

export default function ImageGenPage() {
  const router = useRouter();

  return (
    <CrudListPage
      resource="/image-gen"
      title="AI 图片生成"
      singular="生成任务"
      searchPlaceholder="搜索 prompt..."
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '商品 ID', dataIndex: 'product_id', width: 100 },
        { title: '图片', dataIndex: 'image_urls', width: 80, render: renderImageUrls },
        { title: 'Prompt', dataIndex: 'prompt' },
        { title: '状态', dataIndex: 'status', width: 100 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      fields={[
        { name: 'product_id', label: '商品 ID' },
        { name: 'prompt', label: 'Prompt', type: 'textarea', required: true },
        { name: 'style', label: '风格' },
        { name: 'size', label: '尺寸' },
      ]}
      renderRowActions={(record) => (
        <Button size="small" type="link" icon={<EyeOutlined />} onClick={() => router.push(`/image-gen/${record.id}`)}>详情</Button>
      )}
    />
  );
}
