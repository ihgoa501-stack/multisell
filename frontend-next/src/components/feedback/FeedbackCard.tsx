'use client';

import { Card, Button, Tag, Space, Typography, Progress } from 'antd';
import { LikeOutlined, MessageOutlined, ArrowRightOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { StatusBadge, TypeBadge, SeverityBadge } from './FeedbackStatusBadge';
import dayjs from 'dayjs';

const { Text, Paragraph } = Typography;

interface FeedbackCardProps {
  item: {
    id: number;
    title: string;
    description: string;
    feedback_type: string;
    status: string;
    severity?: string;
    priority: number;
    vote_count: number;
    comment_count: number;
    created_at: string;
    category?: { name: string };
    tags?: { name: string; color: string }[];
  };
}

export default function FeedbackCard({ item }: FeedbackCardProps) {
  const router = useRouter();

  return (
    <Card
      hoverable
      size="small"
      style={{ marginBottom: 12 }}
      onClick={() => router.push(`/feedback/${item.id}`)}
      actions={[
        <Space key="votes">
          <LikeOutlined /> {item.vote_count}
        </Space>,
        <Space key="comments">
          <MessageOutlined /> {item.comment_count}
        </Space>,
        <span key="view">
          详情 <ArrowRightOutlined />
        </span>,
      ]}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div style={{ flex: 1 }}>
          <Space style={{ marginBottom: 4 }}>
            <StatusBadge status={item.status} />
            <TypeBadge type={item.feedback_type} />
            {item.severity && <SeverityBadge severity={item.severity} />}
          </Space>
          <Text strong style={{ display: 'block', fontSize: 15, marginBottom: 4 }}>
            {item.title}
          </Text>
          <Paragraph ellipsis={{ rows: 2 }} type="secondary" style={{ marginBottom: 4 }}>
            {item.description}
          </Paragraph>
          <Space size={4}>
            {item.category && <Tag>{item.category.name}</Tag>}
            {item.tags?.map((t) => (
              <Tag key={t.name} color={t.color || undefined}>{t.name}</Tag>
            ))}
            <Text type="secondary" style={{ fontSize: 12 }}>
              {dayjs(item.created_at).format('YYYY-MM-DD')}
            </Text>
          </Space>
        </div>
        {item.priority > 0 && (
          <Progress
            type="circle"
            percent={item.priority}
            size={40}
            format={(p) => <span style={{ fontSize: 10 }}>{p}</span>}
          />
        )}
      </div>
    </Card>
  );
}
