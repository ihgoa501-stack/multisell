'use client';

import { useState } from 'react';
import { Alert, Button, Card, Col, Input, InputNumber, Row, Skeleton, Space, Typography } from 'antd';
import { useMutation, useQuery } from '@tanstack/react-query';
import PageHeader from '@/components/ui/PageHeader';
import { ApiError } from '@/lib/api-client';
import { getXiaoQCapabilities, getXiaoQIdentity, sendXiaoQMessage } from '@/features/xiaoq/api';
import { XiaoQAnswerCard, XiaoQBoundaryBanner, XiaoQCapabilities } from '@/features/xiaoq/components';

const { Paragraph, Text } = Typography;
const prompts = ['这个案件还缺什么关键证据？', '这个案件的最强反证是什么？', '为什么这个案件当前不能进入下一步？'];

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.category === 'auth') return '当前账号没有访问权限，请重新登录或检查权限。';
    if (error.category === 'timeout') return '请求超时，小Q暂时没有完成回答。';
    if (error.category === 'network') return '无法连接服务，请检查网络后重试。';
  }
  return error instanceof Error ? error.message : '小Q暂时无法回答。';
}

export default function XiaoQPage() {
  const [message, setMessage] = useState('');
  const [demandCaseId, setDemandCaseId] = useState<number | null>(null);
  const [lastRequest, setLastRequest] = useState<{ message: string; demand_case_id: number } | null>(null);

  const identity = useQuery({ queryKey: ['xiao-q-identity'], queryFn: getXiaoQIdentity, retry: 1 });
  const capabilities = useQuery({ queryKey: ['xiao-q-capabilities'], queryFn: getXiaoQCapabilities, retry: 1 });
  const ask = useMutation({ mutationFn: sendXiaoQMessage });

  const submit = (value = message) => {
    const trimmed = value.trim();
    if (!trimmed || !demandCaseId || ask.isPending) return;
    const request = { message: trimmed, demand_case_id: demandCaseId };
    setMessage('');
    setLastRequest(request);
    ask.mutate(request);
  };

  const setupError = identity.error ?? capabilities.error;

  return (
    <div style={{ padding: '16px 20px', minHeight: '100%' }}>
      <PageHeader title="小Q" subtitle="凌镜的 Owner 经营助手：读取系统、解释证据、提出建议" />
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <XiaoQBoundaryBanner identity={identity.data} />

        {setupError && (
          <Alert
            type="error"
            showIcon
            message="小Q基础信息加载失败"
            description={errorMessage(setupError)}
            action={<Button onClick={() => { void identity.refetch(); void capabilities.refetch(); }}>重试</Button>}
          />
        )}

        <Row gutter={[16, 16]}>
          <Col xs={24} lg={16}>
            <Card title="问小Q">
              <Paragraph type="secondary">先从当前经营主线提问。小Q不知道的内容会明确标为未知。</Paragraph>
              <Space direction="vertical" size={4} style={{ width: '100%', marginBottom: 12 }}>
                <Text strong>候选市场案件 ID</Text>
                <InputNumber
                  min={1}
                  precision={0}
                  value={demandCaseId}
                  onChange={(value) => setDemandCaseId(value)}
                  placeholder="请输入要查询的案件 ID"
                  style={{ width: '100%' }}
                  disabled={ask.isPending}
                />
                <Text type="secondary">小Q V1 只读取一个已存在的候选市场案件，不会跨案件猜测。</Text>
              </Space>
              <Space wrap style={{ marginBottom: 12 }}>
                {prompts.map((prompt) => <Button key={prompt} onClick={() => submit(prompt)} disabled={ask.isPending || !demandCaseId}>{prompt}</Button>)}
              </Space>
              <Input.TextArea
                value={message}
                onChange={(event) => setMessage(event.target.value)}
                onPressEnter={(event) => { if (!event.shiftKey) { event.preventDefault(); submit(); } }}
                placeholder="输入关于候选市场、经营证据或实验状态的问题"
                autoSize={{ minRows: 3, maxRows: 8 }}
                disabled={ask.isPending}
              />
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 12 }}>
                <Text type="secondary">Enter 发送，Shift+Enter 换行</Text>
                <Button type="primary" onClick={() => submit()} loading={ask.isPending} disabled={!message.trim() || !demandCaseId}>发送</Button>
              </div>
            </Card>

            {ask.isPending && <Card style={{ marginTop: 16 }}><Skeleton active paragraph={{ rows: 4 }} /></Card>}
            {ask.error && (
              <Alert
                type="error"
                showIcon
                message="回答失败"
                description={errorMessage(ask.error)}
                action={<Button onClick={() => lastRequest && ask.mutate(lastRequest)} disabled={!lastRequest}>重试上次问题</Button>}
                style={{ marginTop: 16 }}
              />
            )}
            {ask.data && <div style={{ marginTop: 16 }}><XiaoQAnswerCard response={ask.data} /></div>}
          </Col>

          <Col xs={24} lg={8}>
            <Card title="已登记能力">
              {capabilities.isLoading ? <Skeleton active paragraph={{ rows: 5 }} /> : <XiaoQCapabilities capabilities={capabilities.data ?? []} />}
            </Card>
          </Col>
        </Row>
      </Space>
    </div>
  );
}
