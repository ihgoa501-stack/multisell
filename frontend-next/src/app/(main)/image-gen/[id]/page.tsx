'use client';

import { useParams, useRouter } from 'next/navigation';
import { Button, Card, Descriptions, Image, message, Result, Skeleton, Tag, Typography } from 'antd';
import { ArrowLeftOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';

interface ImageGenRecord {
  id: number;
  product_id: number;
  prompt: string;
  style: string;
  negative_prompt: string;
  size: string;
  requested_count: number;
  status: string;
  image_urls: string[] | null;
  error_message: string;
  created_by: number;
  batch_id: string;
  created_at: string;
  updated_at: string;
}

const STATUS_COLORS: Record<string, string> = {
  pending: 'default',
  processing: 'processing',
  completed: 'success',
  failed: 'error',
};
const STATUS_LABELS: Record<string, string> = {
  pending: '待处理',
  processing: '生成中',
  completed: '已完成',
  failed: '失败',
};

function fmtDate(v: string): string {
  return v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '-';
}

export default function ImageGenDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['imagegen', id],
    queryFn: async () => {
      const res = await apiClient.get<ImageGenRecord>(`/v1/imagegen/${id}`);
      return res.data;
    },
    retry: false,
  });

  const deleteMutation = useMutation({
    mutationFn: () => apiClient.delete(`/v1/imagegen/${id}`),
    onSuccess: () => { message.success('已删除'); router.push('/image-gen'); },
    onError: (e: Error) => message.error('删除失败: ' + e.message),
  });

  const regenerateMutation = useMutation({
    mutationFn: () => apiClient.post('/v1/imagegen', {
      product_id: data?.product_id,
      prompt: data?.prompt,
      style: data?.style,
      negative_prompt: data?.negative_prompt,
      size: data?.size,
    }),
    onSuccess: () => { message.success('已创建新生成任务'); router.push('/image-gen'); },
    onError: (e: Error) => message.error('重新生成失败: ' + e.message),
  });

  if (isLoading) {
    return (
      <PageContainer title="生成详情">
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/image-gen')} style={{ marginBottom: 'var(--space-lg)' }}>返回列表</Button>
        <Card><Skeleton active paragraph={{ rows: 8 }} /></Card>
      </PageContainer>
    );
  }

  if (isError || !data) {
    return (
      <PageContainer title="生成详情">
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/image-gen')} style={{ marginBottom: 'var(--space-lg)' }}>返回列表</Button>
        <Result status="error" title="加载失败" subTitle="无法获取生成详情" extra={<Button onClick={() => refetch()}>重试</Button>} />
      </PageContainer>
    );
  }

  const imageUrls: string[] = [];
  if (data.image_urls) {
    try {
      const raw = data.image_urls;
      if (Array.isArray(raw)) imageUrls.push(...raw);
      else if (typeof raw === 'string') {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) imageUrls.push(...parsed);
      }
    } catch { /* ignore parse errors */ }
  }

  return (
    <PageContainer title={`生成任务 #${data.id}`}>
      <div style={{ marginBottom: 'var(--space-lg)', display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/image-gen')}>返回列表</Button>
        {data.status === 'completed' && (
          <Button icon={<ReloadOutlined />} loading={regenerateMutation.isPending} onClick={() => regenerateMutation.mutate()}>重新生成</Button>
        )}
        <Button danger icon={<DeleteOutlined />} loading={deleteMutation.isPending} onClick={() => deleteMutation.mutate()}>删除</Button>
      </div>

      <Card style={{ marginBottom: 'var(--space-lg)' }}>
        <Descriptions bordered column={1} size="small">
          <Descriptions.Item label="ID">{data.id}</Descriptions.Item>
          <Descriptions.Item label="商品 ID">{data.product_id}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={STATUS_COLORS[data.status] ?? 'default'}>{STATUS_LABELS[data.status] ?? data.status}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Prompt">{data.prompt}</Descriptions.Item>
          <Descriptions.Item label="负面 Prompt">{data.negative_prompt || '-'}</Descriptions.Item>
          <Descriptions.Item label="风格">{data.style || '-'}</Descriptions.Item>
          <Descriptions.Item label="尺寸">{data.size || '-'}</Descriptions.Item>
          <Descriptions.Item label="请求数量">{data.requested_count}</Descriptions.Item>
          <Descriptions.Item label="批次 ID">{data.batch_id || '-'}</Descriptions.Item>
          {data.error_message && (
            <Descriptions.Item label="错误信息"><Typography.Text type="danger">{data.error_message}</Typography.Text></Descriptions.Item>
          )}
          <Descriptions.Item label="创建时间">{fmtDate(data.created_at)}</Descriptions.Item>
          <Descriptions.Item label="更新时间">{fmtDate(data.updated_at)}</Descriptions.Item>
        </Descriptions>
      </Card>

      {imageUrls.length > 0 && (
        <Card title={`生成的图片 (${imageUrls.length} 张)`}>
          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            {imageUrls.map((url, i) => (
              <Image key={i} src={url} alt={`生成图片 ${i + 1}`} width={200} style={{ borderRadius: 8, objectFit: 'cover' }} />
            ))}
          </div>
        </Card>
      )}
    </PageContainer>
  );
}
