'use client';

import { useMemo, useState } from 'react';
import { Button, Form, Input, Modal, Popconfirm, Select, Space, Table, message } from 'antd';
import { PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import type { PageResult } from '@/types/api';

export interface CrudColumn {
  title: string;
  dataIndex: string;
  width?: number | string;
  render?: (value: unknown, record: Record<string, unknown>) => React.ReactNode;
}

export interface CrudField {
  name: string;
  label: string;
  type?: 'text' | 'number' | 'textarea' | 'select';
  required?: boolean;
  options?: { label: string; value: string | number }[];
  initialValue?: unknown;
}

export interface CrudListPageProps {
  /** Resource path under /api/v1, e.g. "/order". Must NOT include leading /api/v1. */
  resource: string;
  title: string;
  /** Singular label, e.g. "订单". */
  singular: string;
  columns: CrudColumn[];
  fields: CrudField[];
  /** Optional search placeholder; disable search with empty string. */
  searchPlaceholder?: string;
  /** Extra filter query params appended to list requests. */
  extraFilters?: Record<string, string>;
  /** Whether to expose create/edit/delete. Default true. */
  editable?: boolean;
  /** Render extra row actions after the default edit/delete. */
  renderRowActions?: (record: Record<string, unknown>) => React.ReactNode;
  /** Custom page size, default 10. */
  pageSize?: number;
}

interface ListResponse {
  data?: Record<string, unknown>[];
  total: number;
  page: number;
  size: number;
}

export default function CrudListPage({
  resource,
  title,
  singular,
  columns,
  fields,
  searchPlaceholder = '搜索...',
  extraFilters,
  editable = true,
  renderRowActions,
  pageSize = 10,
}: CrudListPageProps) {
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(pageSize);
  const [search, setSearch] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Record<string, unknown> | null>(null);
  const [form] = Form.useForm();

  const listKey = ['crud', resource, page, size, search, JSON.stringify(extraFilters ?? {})];
  const { data, isLoading, refetch } = useQuery({
    queryKey: listKey,
    queryFn: async () => {
      const params: Record<string, string> = {
        page: String(page),
        size: String(size),
      };
      if (search) params.search = search;
      if (extraFilters) Object.assign(params, extraFilters);
      const res = await apiClient.getPage<Record<string, unknown>>(`/v1${resource}`, params);
      return res as unknown as ListResponse;
    },
  });

  const createMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) =>
      apiClient.post(`/v1${resource}`, values),
    onSuccess: () => {
      message.success('已创建');
      setModalOpen(false);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['crud', resource] });
    },
    onError: (e: Error) => message.error(`创建失败: ${e.message}`),
  });

  const updateMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) => {
      const id = editing?.id;
      return apiClient.put(`/v1${resource}/${id}`, values);
    },
    onSuccess: () => {
      message.success('已更新');
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['crud', resource] });
    },
    onError: (e: Error) => message.error(`更新失败: ${e.message}`),
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: unknown) => apiClient.delete(`/v1${resource}/${id}`),
    onSuccess: () => {
      message.success('已删除');
      qc.invalidateQueries({ queryKey: ['crud', resource] });
    },
    onError: (e: Error) => message.error(`删除失败: ${e.message}`),
  });

  const tableColumns = useMemo(() => {
    // Use any[] to avoid Antd 6 strict ColumnType inference friction across
    // mixed render shapes; this is a generic CRUD wrapper, not a typed model.
    const cols: Record<string, unknown>[] = columns.map((c) => ({
      title: c.title,
      dataIndex: c.dataIndex,
      width: c.width,
      render: c.render,
    }));
    if (editable || renderRowActions) {
      cols.push({
        title: '操作',
        key: '__actions__',
        width: 180,
        fixed: 'right',
        render: (_: unknown, record: Record<string, unknown>) => (
          <Space size="small">
            {editable && (
              <Button
                size="small"
                type="link"
                icon={<EditOutlined />}
                onClick={() => {
                  setEditing(record);
                  form.setFieldsValue(record);
                  setModalOpen(true);
                }}
              >
                编辑
              </Button>
            )}
            {renderRowActions?.(record)}
            {editable && (
              <Popconfirm
                title={`确认删除此${singular}？`}
                onConfirm={() => deleteMutation.mutate(record.id)}
              >
                <Button size="small" type="link" danger icon={<DeleteOutlined />}>
                  删除
                </Button>
              </Popconfirm>
            )}
          </Space>
        ),
      });
    }
    return cols as never;
  }, [columns, editable, renderRowActions, singular, deleteMutation, form, editing]);

  const handleSubmit = async () => {
    const values = await form.validateFields();
    if (editing) {
      updateMutation.mutate(values);
    } else {
      createMutation.mutate(values);
    }
  };

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h1 style={{ fontSize: 22, fontWeight: 600, margin: 0 }}>{title}</h1>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
            刷新
          </Button>
          {editable && (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditing(null);
                form.resetFields();
                setModalOpen(true);
              }}
            >
              新建{singular}
            </Button>
          )}
        </Space>
      </div>

      {searchPlaceholder && (
        <div style={{ marginBottom: 16 }}>
          <Input.Search
            placeholder={searchPlaceholder}
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(1);
            }}
            style={{ maxWidth: 400 }}
            allowClear
          />
        </div>
      )}

      <Table
        rowKey="id"
        loading={isLoading}
        dataSource={data?.data}
        columns={tableColumns}
        scroll={{ x: 'max-content' }}
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

      <Modal
        title={editing ? `编辑${singular}` : `新建${singular}`}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false);
          setEditing(null);
          form.resetFields();
        }}
        onOk={handleSubmit}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        width={560}
        destroyOnClose
      >
        <Form form={form} layout="vertical" preserve={false}>
          {fields.map((f) => (
            <Form.Item
              key={f.name}
              name={f.name}
              label={f.label}
              rules={f.required ? [{ required: true, message: `请输入${f.label}` }] : []}
            >
              {f.type === 'textarea' ? (
                <Input.TextArea rows={3} />
              ) : f.type === 'number' ? (
                <Input type="number" />
              ) : f.type === 'select' ? (
                <Select options={f.options} allowClear={!f.required} />
              ) : (
                <Input />
              )}
            </Form.Item>
          ))}
        </Form>
      </Modal>
    </div>
  );
}

/** Helper: format an ISO timestamp for table display. */
export function fmtDate(v: unknown): string {
  if (!v) return '-';
  const s = String(v);
  const d = dayjs(s);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm') : s;
}

/** Helper: format currency in CNY. */
export function fmtMoney(v: unknown): string {
  if (v === null || v === undefined || v === '') return '-';
  const n = Number(v);
  if (Number.isNaN(n)) return String(v);
  return `¥${n.toFixed(2)}`;
}
