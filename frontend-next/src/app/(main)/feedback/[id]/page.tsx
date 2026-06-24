'use client';

import { useEffect, useState } from 'react';
import {
  Typography, Card, Spin, Tag, Descriptions, Space, Button, Divider,
  Input, List, Form, message, Row, Col, Timeline, Progress, Result,
} from 'antd';
import {
  ArrowLeftOutlined, LikeOutlined, DislikeOutlined,
  SendOutlined, MinusCircleOutlined,
} from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import { StatusBadge, TypeBadge, SeverityBadge } from '@/components/feedback/FeedbackStatusBadge';
import dayjs from 'dayjs';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

export default function FeedbackDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [commentForm] = Form.useForm();
  const [commentLoading, setCommentLoading] = useState(false);
  const [voting, setVoting] = useState(false);

  const fetchDetail = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get<any>(`/v1/feedback/submissions/${id}`);
      if (res.code === 0 && res.data) {
        setData(res.data);
      } else {
        setNotFound(true);
      }
    } catch {
      setNotFound(true);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (id) fetchDetail();
  }, [id]);

  const handleVote = async (voteType: string) => {
    setVoting(true);
    try {
      const res = await apiClient.post<any>(`/v1/feedback/submissions/${id}/vote`, { vote_type: voteType });
      if (res.code === 0) {
        fetchDetail();
      }
    } catch (err: any) {
      message.error(err.message || '投票失败');
    } finally {
      setVoting(false);
    }
  };

  const handleComment = async (values: { body: string }) => {
    setCommentLoading(true);
    try {
      const res = await apiClient.post<any>(`/v1/feedback/submissions/${id}/comments`, {
        body: values.body,
        is_internal: false,
      });
      if (res.code === 0) {
        message.success('评论成功');
        commentForm.resetFields();
        fetchDetail();
      }
    } catch (err: any) {
      message.error(err.message || '评论失败');
    } finally {
      setCommentLoading(false);
    }
  };

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>;
  }

  if (notFound) {
    return <Result status="404" title="反馈不存在" extra={<Button onClick={() => router.push('/feedback')}>返回</Button>} />;
  }

  const d = data;

  return (
    <div style={{ maxWidth: 900, margin: '0 auto', padding: 24 }}>
      <Button type="link" icon={<ArrowLeftOutlined />} onClick={() => router.push('/feedback')} style={{ marginBottom: 16 }}>
        返回反馈列表
      </Button>

      <Row gutter={24}>
        <Col xs={24} lg={16}>
          <Card>
            <Space style={{ marginBottom: 12 }}>
              <StatusBadge status={d.status} />
              <TypeBadge type={d.feedback_type} />
              {d.severity && <SeverityBadge severity={d.severity} />}
            </Space>
            <Title level={4}>{d.title}</Title>
            <Paragraph style={{ fontSize: 15, whiteSpace: 'pre-wrap' }}>{d.description}</Paragraph>

            {d.attachments && d.attachments !== '[]' && (
              <div style={{ marginTop: 16 }}>
                <Text strong>附件：</Text>
                <Space>
                  {(JSON.parse(d.attachments) || []).map((att: any, i: number) => (
                    <Tag key={i}>{att.name || `附件 ${i + 1}`}</Tag>
                  ))}
                </Space>
              </div>
            )}

            <Divider />
            <Title level={5}>评论 ({d.comments?.length || 0})</Title>
            <List
              dataSource={d.comments || []}
              locale={{ emptyText: '暂无评论' }}
              renderItem={(c: any) => (
                <List.Item>
                  <List.Item.Meta
                    title={<Text strong>用户 #{c.user_id}</Text>}
                    description={<Paragraph style={{ whiteSpace: 'pre-wrap' }}>{c.body}</Paragraph>}
                  />
                  <Text type="secondary" style={{ fontSize: 12 }}>{dayjs(c.created_at).format('MM-DD HH:mm')}</Text>
                </List.Item>
              )}
            />

            <Form form={commentForm} layout="inline" onFinish={handleComment} style={{ marginTop: 16, width: '100%' }}>
              <Form.Item name="body" rules={[{ required: true, message: '请输入评论内容' }]} style={{ flex: 1 }}>
                <TextArea rows={2} placeholder="写下你的想法..." />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" icon={<SendOutlined />} loading={commentLoading}>
                  发表
                </Button>
              </Form.Item>
            </Form>
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card size="small" style={{ marginBottom: 16 }}>
            <div style={{ textAlign: 'center', marginBottom: 16 }}>
              <Space size="large">
                <Button
                  icon={<LikeOutlined />}
                  onClick={() => handleVote('upvote')}
                  loading={voting}
                  type={d.user_vote === 'upvote' ? 'primary' : 'default'}
                >
                  {d.vote_count || 0}
                </Button>
                <Button
                  icon={<DislikeOutlined />}
                  onClick={() => handleVote('downvote')}
                  loading={voting}
                  type={d.user_vote === 'downvote' ? 'primary' : 'default'}
                />
              </Space>
            </div>
          </Card>

          <Card size="small" title="详细信息" style={{ marginBottom: 16 }}>
            <Descriptions column={1} size="small">
              <Descriptions.Item label="反馈类型"><TypeBadge type={d.feedback_type} /></Descriptions.Item>
              {d.severity && <Descriptions.Item label="严重程度"><SeverityBadge severity={d.severity} /></Descriptions.Item>}
              <Descriptions.Item label="优先级">
                <Progress percent={d.priority} size="small" format={(p) => `${p}`} />
              </Descriptions.Item>
              <Descriptions.Item label="提交时间">{dayjs(d.created_at).format('YYYY-MM-DD HH:mm')}</Descriptions.Item>
              {d.reviewed_at && (
                <Descriptions.Item label="审核时间">{dayjs(d.reviewed_at).format('YYYY-MM-DD HH:mm')}</Descriptions.Item>
              )}
              {d.shipped_at && (
                <Descriptions.Item label="上线时间">{dayjs(d.shipped_at).format('YYYY-MM-DD HH:mm')}</Descriptions.Item>
              )}
            </Descriptions>
          </Card>

          {d.status_logs && d.status_logs.length > 0 && (
            <Card size="small" title="状态历程">
              <Timeline
                items={d.status_logs.map((sl: any) => ({
                  children: (
                    <>
                      <Text strong>{sl.to_status === 'pending' ? '已提交' : sl.to_status}</Text>
                      <br />
                      <Text type="secondary" style={{ fontSize: 12 }}>{dayjs(sl.created_at).format('MM-DD HH:mm')}</Text>
                      {sl.note && <Paragraph style={{ fontSize: 13, margin: '4px 0' }}>{sl.note}</Paragraph>}
                    </>
                  ),
                }))}
              />
            </Card>
          )}
        </Col>
      </Row>
    </div>
  );
}
