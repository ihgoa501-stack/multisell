'use client';

import { useEffect, useState } from 'react';
import { Card, Spin, Result, Button, Select, Space, message } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import FeedbackForm from '@/components/feedback/FeedbackForm';
import type { FeedbackProject } from '@/types/feedback';

export default function FeedbackSubmitPage() {
  const router = useRouter();
  const [projectId, setProjectId] = useState<number | null>(null);
  const [projects, setProjects] = useState<FeedbackProject[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const init = async () => {
      try {
        const res = await apiClient.get<FeedbackProject[]>('/v1/feedback/projects');
        if (res.code === 0 && res.data && res.data.length > 0) {
          setProjects(res.data);
          setProjectId(res.data[0].id);
        }
      } catch {
        message.error('无法加载项目配置');
      } finally {
        setLoading(false);
      }
    };
    init();
  }, []);

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>;
  }

  if (!projectId) {
    return (
      <div style={{ maxWidth: 600, margin: '0 auto', padding: 24 }}>
        <Result
          status="info"
          title="反馈系统尚未配置"
          subTitle="请联系管理员初始化反馈项目配置"
          extra={<Button type="primary" onClick={() => router.push('/feedback')}>返回</Button>}
        />
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 700, margin: '0 auto', padding: 24 }}>
      <Button type="link" icon={<ArrowLeftOutlined />} onClick={() => router.push('/feedback')} style={{ marginBottom: 16 }}>
        返回反馈列表
      </Button>

      {projects.length > 1 && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Space>
            <span>反馈项目：</span>
            <Select
              value={projectId}
              onChange={setProjectId}
              options={projects.map((p) => ({ value: p.id, label: p.name }))}
              style={{ width: 200 }}
            />
          </Space>
        </Card>
      )}

      <FeedbackForm projectId={projectId} />
    </div>
  );
}
