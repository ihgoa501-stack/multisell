'use client';

import { Card, Descriptions, Result, Spin, Image } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { Button } from 'antd';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';

interface ProductDetail {
  id?: string | number;
  name?: string;
  subtitle?: string;
  brand_id?: string | number;
  category_id?: string | number;
  status?: string;
  main_image?: string;
  cargo_type?: string;
  unit?: string;
  description?: string;
}

export default function ProductDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;

  const { data, isLoading, isError } = useQuery({
    queryKey: ['product', id],
    queryFn: async () => {
      const res = await apiClient.get<ProductDetail>(`/v1/products/${id}`);
      return res.data;
    },
    retry: false,
  });

  return (
    <PageContainer title="商品详情">
      <Button
        icon={<ArrowLeftOutlined />}
        onClick={() => router.push('/products')}
        style={{ marginBottom: 16 }}
      >
        返回列表
      </Button>

      {isLoading ? (
        <Card>
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin tip="加载中..." />
          </div>
        </Card>
      ) : isError || !data ? (
        <Card>
          <Result status="info" title="商品详情" subTitle="暂无详情数据或商品不存在" />
        </Card>
      ) : (
        <Card>
          {data.main_image && (
            <div style={{ marginBottom: 16 }}>
              <Image src={data.main_image} alt={data.name} width={200} />
            </div>
          )}
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="ID">{data.id ?? id}</Descriptions.Item>
            <Descriptions.Item label="名称">{data.name ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="副标题">{data.subtitle ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="品牌 ID">{data.brand_id ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="分类 ID">{data.category_id ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">{data.status ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="货物类型">{data.cargo_type ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="单位">{data.unit ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>
              {data.description ?? '-'}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      )}
    </PageContainer>
  );
}
