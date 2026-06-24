'use client';

import { Button, Card, Form, Input, Select, Space, message } from 'antd';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';

const STATUS_OPTIONS = [
  { label: '启用', value: 'active' },
  { label: '停用', value: 'inactive' },
  { label: '草稿', value: 'draft' },
];

const CARGO_TYPE_OPTIONS = [
  { label: '实物', value: 'physical' },
  { label: '虚拟', value: 'virtual' },
  { label: '服务', value: 'service' },
];

export default function ProductCreatePage() {
  const router = useRouter();
  const [form] = Form.useForm();

  const createMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) =>
      apiClient.post('/v1/products', values),
    onSuccess: () => {
      message.success('已创建商品');
      router.push('/products');
    },
    onError: (e: Error) => message.error(`创建失败: ${e.message}`),
  });

  const handleSubmit = async () => {
    const values = await form.validateFields();
    createMutation.mutate(values);
  };

  return (
    <PageContainer title="创建商品">
      <Card style={{ maxWidth: 720 }}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="商品名称" />
          </Form.Item>
          <Form.Item name="subtitle" label="副标题">
            <Input placeholder="商品副标题" />
          </Form.Item>
          <Form.Item name="brand_id" label="品牌 ID">
            <Input placeholder="品牌 ID" />
          </Form.Item>
          <Form.Item name="category_id" label="分类 ID">
            <Input placeholder="分类 ID" />
          </Form.Item>
          <Form.Item name="unit" label="单位" initialValue="件">
            <Input placeholder="件 / 个 / 箱" />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="draft">
            <Select options={STATUS_OPTIONS} />
          </Form.Item>
          <Form.Item name="cargo_type" label="货物类型" initialValue="physical">
            <Select options={CARGO_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={4} placeholder="商品描述" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" loading={createMutation.isPending} onClick={handleSubmit}>
                提交
              </Button>
              <Button onClick={() => router.push('/products')}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </PageContainer>
  );
}
