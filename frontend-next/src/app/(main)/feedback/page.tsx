'use client';

import { useState, useEffect } from 'react';
import { Typography, Button, Tabs, Empty, Spin, Row } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import FeedbackCard from '@/components/feedback/FeedbackCard';
import type { FeedbackProject, FeedbackSubmission } from '@/types/feedback';

const { Title } = Typography;

export default function FeedbackPortalPage() {
  const router = useRouter();
  const [submissions, setSubmissions] = useState<FeedbackSubmission[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState('mine');

  useEffect(() => {
    async function fetchData() {
      setLoading(true);
      try {
        const params: Record<string, string> = { page: '1', size: '20' };
        if (tab !== 'mine') params.status = tab;

        const projRes = await apiClient.get<FeedbackProject[]>('/v1/feedback/projects');
        if (projRes.code === 0 && projRes.data && projRes.data.length > 0) {
          params.project_id = String(projRes.data[0].id);
        }

        const res = await apiClient.getPage<FeedbackSubmission>('/v1/feedback/mine', params);
        if (res.code === 0) {
          setSubmissions(res.data || []);
        }
      } catch {
        setSubmissions([]);
      } finally {
        setLoading(false);
      }
    }

    void fetchData();
  }, [tab]);

  return (
    <div style={{ maxWidth: 800, margin: '0 auto', padding: 24 }}>
      <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
        <Title level={3} style={{ margin: 0 }}>用户反馈</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => router.push('/feedback/submit')}>
          提交反馈
        </Button>
      </Row>

      <Tabs
        activeKey={tab}
        onChange={setTab}
        items={[
          { key: 'mine', label: '我的反馈' },
          { key: 'pending', label: '待审核' },
          { key: 'accepted', label: '已采纳' },
          { key: 'shipped', label: '已上线' },
        ]}
        style={{ marginBottom: 16 }}
      />

      {loading ? (
        <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>
      ) : submissions.length === 0 ? (
        <Empty description="暂无反馈" />
      ) : (
        submissions.map((item) => (
          <FeedbackCard key={item.id} item={item} />
        ))
      )}
    </div>
  );
}
