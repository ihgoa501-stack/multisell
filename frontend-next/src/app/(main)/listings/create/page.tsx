'use client';

import { Button, Card, Form, Input, Select, Space, message } from 'antd';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';

const STATUS_OPTIONS = [
  { label: '草稿', value: 'draft' },
  { label: '已发布', value: 'published' },
  { label: '已下架', value: 'unpublished' },
];

export default function ListingCreatePage() {
  const router = useRouter();
  const [form] = Form.useForm();

  const createMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) =>
      apiClient.post('/v1/listing', values),
    onSuccess: () => {
      message.success('已创建 listing');
      router.push('/listings');
    },
    onError: (e: Error) => message.error(`创建失败: ${e.message}`),
  });

  const handleSubmit = async () => {
    const values = await form.validateFields();
    createMutation.mutate(values);
  };

  return (
    <PageContainer title="创建 Listing">
      <Card style={{ maxWidth: 720 }}>
        <Form form={form} layout="vertical">
          <Form.Item name="product_id" label="商品" rules={[{ required: true, message: '请选择商品' }]}>
            <Select
              placeholder="选择商品"
              showSearch
              optionFilterProp="label"
              options={[]}
              notFoundContent="暂无商品（请先在商品页创建）"
            />
          </Form.Item>
          <Form.Item name="platform_id" label="平台" rules={[{ required: true, message: '请选择平台' }]}>
            <Select
              placeholder="选择平台"
              options={[
                { label: '淘宝', value: 'taobao' },
                { label: '京东', value: 'jd' },
                { label: '拼多多', value: 'pdd' },
                { label: '抖音', value: 'douyin' },
              ]}
            />
          </Form.Item>
          <Form.Item name="external_id" label="外部 ID">
            <Input placeholder="平台侧商品/Listing ID" />
          </Form.Item>
          <Form.Item name="listing_url" label="Listing URL">
            <Input placeholder="https://" />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="draft">
            <Select options={STATUS_OPTIONS} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" loading={createMutation.isPending} onClick={handleSubmit}>
                提交
              </Button>
              <Button onClick={() => router.push('/listings')}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </PageContainer>
  );
}
