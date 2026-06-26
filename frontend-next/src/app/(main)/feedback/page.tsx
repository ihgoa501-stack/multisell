'use client';

import { useState, useEffect } from 'react';
import { Typography, Button, Tabs, Empty, Spin, Row, Col } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import FeedbackCard from '@/components/feedback/FeedbackCard';
import { StatusBadge } from '@/components/feedback/FeedbackStatusBadge';

const { Title } = Typography;

export default function FeedbackPortalPage() {
  const router = useRouter();
  const [submissions, setSubmissions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState('mine');

  const fetchData = async (status?: string) => {
    setLoading(true);
    try {
      // Try to get my submissions first; fall back to public list
      const params: any = { page: 1, size: 20 };
      if (status) params.status = status;

      // First get default project
      const projRes = await apiClient.get<any[]>('/v1/feedback/projects');
      if (projRes.code === 0 && projRes.data && projRes.data.length > 0) {
        params.project_id = projRes.data[0].id;
      }

      const res = await apiClient.getPage('/v1/feedback/mine', params);
      if (res.code === 0) {
        setSubmissions(res.data || []);
      }
    } catch {
      // Not logged in or no data - show empty
      setSubmissions([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

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
        onChange={(k) => setTab(k)}
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
        submissions.map((item: any) => (
          <FeedbackCard key={item.id} item={item} />
        ))
      )}
    </div>
  );
}
