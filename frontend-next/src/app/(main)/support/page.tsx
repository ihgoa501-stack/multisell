'use client';

import { useMemo, useState } from 'react';
import {
  Tag,
  Drawer,
  Input,
  Button,
  Select,
  Space,
  Table,
  message,
  Spin,
  Empty,
  Descriptions,
  Divider,
  List,
} from 'antd';
import { ReloadOutlined, SendOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import FilterBar from '@/components/ui/FilterBar';
import type { FilterOption } from '@/components/ui/FilterBar';
import { fmtDate } from '@/components/crud/CrudListPage';

const { TextArea } = Input;

const statusMap: Record<string, { color: string; label: string }> = {
  open: { color: 'blue', label: '开启' },
  in_progress: { color: 'orange', label: '处理中' },
  resolved: { color: 'green', label: '已解决' },
  closed: { color: 'default', label: '关闭' },
};

const priorityMap: Record<string, { color: string; label: string }> = {
  low: { color: 'default', label: '低' },
  medium: { color: 'blue', label: '中' },
  high: { color: 'orange', label: '高' },
  urgent: { color: 'red', label: '紧急' },
};

const filters: FilterOption[] = [
  {
    key: 'status',
    label: '状态',
    options: [
      { label: '开启', value: 'open' },
      { label: '处理中', value: 'in_progress' },
      { label: '已解决', value: 'resolved' },
      { label: '关闭', value: 'closed' },
    ],
  },
  {
    key: 'priority',
    label: '优先级',
    options: [
      { label: '低', value: 'low' },
      { label: '中', value: 'medium' },
      { label: '高', value: 'high' },
      { label: '紧急', value: 'urgent' },
    ],
  },
];

interface Message {
  id: number;
  content: string;
  sender_type: string;
  created_at: string;
}

const senderLabels: Record<string, string> = {
  customer: '客户',
  support: '客服',
  system: '系统',
  ai: 'AI 助手',
};

export default function SupportPage() {
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(10);
  const [search, setSearch] = useState('');
  const [filterValues, setFilterValues] = useState<
    Record<string, string | number | null | undefined>
  >({});

  // Drawer state
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedConv, setSelectedConv] = useState<Record<string, unknown> | null>(null);
  const [replyText, setReplyText] = useState('');
  const [sending, setSending] = useState(false);
  const [closing, setClosing] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<string | undefined>(undefined);

  const queryParams = useMemo(() => {
    const params: Record<string, string> = { page: String(page), size: String(size) };
    if (search) params.search = search;
    Object.entries(filterValues).forEach(([k, v]) => {
      if (v !== undefined && v !== null && v !== '') params[k] = String(v);
    });
    return params;
  }, [page, size, search, filterValues]);

  // Conversations list
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['support-conversations', queryParams],
    queryFn: () =>
      apiClient.getPage<Record<string, unknown>>('/v1/support/conversations', queryParams),
  });

  // Messages for selected conversation
  const {
    data: messagesData,
    isLoading: messagesLoading,
    refetch: refetchMessages,
  } = useQuery({
    queryKey: ['support-messages', selectedConv?.id],
    queryFn: () =>
      apiClient.get<Message[]>(`/v1/support/conversations/${selectedConv!.id}/messages`),
    enabled: !!selectedConv,
  });

  // Reply templates
  const { data: templatesData } = useQuery({
    queryKey: ['support-templates'],
    queryFn: () =>
      apiClient.get<{ id: number; name: string; content: string }[]>('/v1/support/templates'),
    enabled: drawerOpen,
  });

  const templates = templatesData?.data || [];

  const handleRowClick = (record: Record<string, unknown>) => {
    setSelectedConv(record);
    setReplyText('');
    setSelectedTemplate(undefined);
    setDrawerOpen(true);
  };

  const handleTemplateChange = (value: string) => {
    setSelectedTemplate(value);
    const tmpl = templates.find((t) => String(t.id) === value);
    if (tmpl) setReplyText(tmpl.content);
  };

  const handleSendReply = async () => {
    if (!replyText.trim() || !selectedConv) return;
    setSending(true);
    try {
      await apiClient.post(`/v1/support/conversations/${selectedConv.id}/reply`, {
        content: replyText,
      });
      message.success('回复发送成功');
      setReplyText('');
      setSelectedTemplate(undefined);
      refetchMessages();
      qc.invalidateQueries({ queryKey: ['support-conversations'] });
    } catch (e: unknown) {
      const err = e instanceof Error ? e.message : '未知错误';
      message.error(`发送失败: ${err}`);
    } finally {
      setSending(false);
    }
  };

  const handleCloseConversation = async () => {
    if (!selectedConv) return;
    setClosing(true);
    try {
      await apiClient.post(`/v1/support/conversations/${selectedConv.id}/close`);
      message.success('会话已关闭');
      setDrawerOpen(false);
      setSelectedConv(null);
      qc.invalidateQueries({ queryKey: ['support-conversations'] });
    } catch (e: unknown) {
      const err = e instanceof Error ? e.message : '未知错误';
      message.error(`关闭失败: ${err}`);
    } finally {
      setClosing(false);
    }
  };

  const columns = [
    { title: '会话ID', dataIndex: 'id', width: 80 },
    { title: '客户名称', dataIndex: 'customer_name', width: 140 },
    { title: '主题', dataIndex: 'subject', width: 240 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: unknown) => {
        const s = statusMap[String(v)] ?? { color: 'default', label: String(v) };
        return <Tag color={s.color}>{s.label}</Tag>;
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      width: 90,
      render: (v: unknown) => {
        const p = priorityMap[String(v)] ?? { color: 'default', label: String(v) };
        return <Tag color={p.color}>{p.label}</Tag>;
      },
    },
    { title: '最后消息时间', dataIndex: 'last_message_at', width: 160, render: fmtDate },
  ];

  const hasActiveFilters =
    search.length > 0 ||
    Object.values(filterValues).some(
      (v) => v !== undefined && v !== null && v !== '',
    );

  const messages = (messagesData?.data || []) as Message[];

  return (
    <div style={{ padding: '16px 20px', minHeight: '100%' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 12,
        }}
      >
        <h1
          style={{
            fontFamily: 'var(--ds)',
            fontWeight: 700,
            fontSize: 'var(--text-h1)',
            color: 'var(--t1)',
            margin: 0,
          }}
        >
          客服中心
        </h1>
        <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
          刷新
        </Button>
      </div>

      <FilterBar
        search={search}
        onSearch={(v) => {
          setSearch(v);
          setPage(1);
        }}
        searchPlaceholder="搜索会话ID / 客户名称 / 主题..."
        filters={filters}
        filterValues={filterValues}
        onFilterChange={(key, value) => {
          setFilterValues((p) => ({ ...p, [key]: value }));
          setPage(1);
        }}
        onReset={
          hasActiveFilters
            ? () => {
                setSearch('');
                setFilterValues({});
                setPage(1);
              }
            : undefined
        }
      />

      <Table
        rowKey="id"
        loading={isLoading}
        dataSource={data?.data}
        columns={columns}
        scroll={{ x: 'max-content' }}
        onRow={(record) => ({
          onClick: () => handleRowClick(record),
          style: { cursor: 'pointer' },
        })}
        locale={{ emptyText: <Empty description="暂无会话" /> }}
        pagination={{
          current: data?.page ?? page,
          pageSize: data?.size ?? size,
          total: data?.total ?? 0,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p, s) => {
            setPage(p);
            setSize(s);
          },
        }}
      />

      {/* Conversation detail drawer */}
      <Drawer
        title={selectedConv ? `会话 #${selectedConv.id}` : ''}
        placement="right"
        width={520}
        open={drawerOpen}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedConv(null);
        }}
        destroyOnClose
      >
        {selectedConv && (
          <>
            <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
              <Descriptions.Item label="客户名称">
                {(selectedConv.customer_name as string) || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="邮箱">
                {(selectedConv.customer_email as string) || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="平台">
                {(selectedConv.platform as string) || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="主题">
                {(selectedConv.subject as string) || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag
                  color={
                    statusMap[String(selectedConv.status)]?.color || 'default'
                  }
                >
                  {statusMap[String(selectedConv.status)]?.label ||
                    String(selectedConv.status)}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="优先级">
                <Tag
                  color={
                    priorityMap[String(selectedConv.priority)]?.color || 'default'
                  }
                >
                  {priorityMap[String(selectedConv.priority)]?.label ||
                    String(selectedConv.priority)}
                </Tag>
              </Descriptions.Item>
            </Descriptions>

            <Divider titlePlacement="left" plain>
              消息记录
            </Divider>
            <div style={{ maxHeight: 300, overflowY: 'auto', marginBottom: 16 }}>
              <Spin spinning={messagesLoading}>
                {messages.length === 0 ? (
                  <Empty description="暂无消息" />
                ) : (
                  <List
                    dataSource={messages}
                    renderItem={(msg: Message) => (
                      <List.Item>
                        <List.Item.Meta
                          title={
                            <span>
                              {senderLabels[msg.sender_type] || msg.sender_type}{' '}
                              <span style={{ fontSize: 12, color: '#999' }}>
                                {dayjs(msg.created_at).format('MM-DD HH:mm')}
                              </span>
                            </span>
                          }
                          description={msg.content}
                        />
                      </List.Item>
                    )}
                  />
                )}
              </Spin>
            </div>

            <Divider titlePlacement="left" plain>
              回复
            </Divider>
            <Space direction="vertical" style={{ width: '100%' }} size={8}>
              {templates.length > 0 && (
                <Select
                  allowClear
                  placeholder="选择回复模板..."
                  style={{ width: '100%' }}
                  value={selectedTemplate}
                  onChange={handleTemplateChange}
                  options={templates.map((t) => ({
                    label: t.name,
                    value: String(t.id),
                  }))}
                />
              )}
              <TextArea
                rows={4}
                value={replyText}
                onChange={(e) => setReplyText(e.target.value)}
                placeholder="输入回复内容..."
              />
              <Space>
                <Button
                  type="primary"
                  icon={<SendOutlined />}
                  loading={sending}
                  disabled={!replyText.trim()}
                  onClick={handleSendReply}
                >
                  发送回复
                </Button>
                <Button danger loading={closing} onClick={handleCloseConversation}>
                  关闭会话
                </Button>
              </Space>
            </Space>
          </>
        )}
      </Drawer>
    </div>
  );
}
