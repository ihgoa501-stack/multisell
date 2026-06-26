'use client';

import { Form, Input, Select, Button, Card, message, Typography } from 'antd';
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import { feedbackTypeList, feedbackSeverityList } from './FeedbackStatusBadge';

const { TextArea } = Input;
const { Title } = Typography;

interface FeedbackFormProps {
  projectId: number;
  onSuccess?: () => void;
}

export default function FeedbackForm({ projectId, onSuccess }: FeedbackFormProps) {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      const res = await apiClient.post<any>('/v1/feedback/submissions', {
        project_id: projectId,
        title: values.title,
        description: values.description,
        feedback_type: values.feedback_type,
        severity: values.severity,
        url: values.url || window.location.href,
        user_agent: navigator.userAgent,
      });
      if (res.code === 0) {
        message.success('反馈提交成功！感谢你的建议 🙏');
        form.resetFields();
        if (onSuccess) {
          onSuccess();
        }
        router.push('/feedback/' + res.data?.id);
      } else {
        message.error(res.message || '提交失败');
      }
    } catch (err: any) {
      message.error('提交失败: ' + (err.message || '未知错误'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <Title level={4}>提交反馈</Title>
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        initialValues={{ feedback_type: 'feature' }}
      >
        <Form.Item
          name="title"
          label="标题"
          rules={[{ required: true, message: '请输入标题' }, { max: 500 }]}
        >
          <Input placeholder="用一句话描述你的想法..." maxLength={500} showCount />
        </Form.Item>

        <Form.Item
          name="description"
          label="详细描述"
          rules={[{ required: true, message: '请描述你的想法或遇到的问题' }]}
        >
          <TextArea rows={4} placeholder="请详细描述你的建议、需求或遇到的问题..." />
        </Form.Item>

        <Form.Item
          name="feedback_type"
          label="反馈类型"
          rules={[{ required: true }]}
        >
          <Select options={feedbackTypeList} />
        </Form.Item>

        <Form.Item name="severity" label="严重程度">
          <Select options={feedbackSeverityList} allowClear placeholder="请选择（可选）" />
        </Form.Item>

        <Form.Item name="url" label="当前页面 URL" hidden>
          <Input />
        </Form.Item>

        <Form.Item>
          <Button type="primary" htmlType="submit" loading={loading} size="large">
            提交反馈
          </Button>
        </Form.Item>
      </Form>
    </Card>
  );
}
