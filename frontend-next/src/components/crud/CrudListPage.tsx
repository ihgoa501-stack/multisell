'use client';

import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Form, Input, InputNumber, Modal, Select, Space, Table, message } from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  EditOutlined,
  DeleteOutlined,
  DownloadOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import FilterBar from '@/components/ui/FilterBar';
import BatchActionBar from '@/components/ui/BatchActionBar';
import ConfirmDialog from '@/components/ui/ConfirmDialog';
import type { FilterOption } from '@/components/ui/FilterBar';
import type { BatchActionItem } from '@/components/ui/BatchActionBar';

// ---------- Re-exported types ----------

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
  /** Optional search placeholder; disable search with empty string or null. */
  searchPlaceholder?: string;
  /** Multi-dimension filter configs for FilterBar integration. */
  filters?: FilterOption[];
  /** Extra filter query params appended to list requests. */
  extraFilters?: Record<string, string>;
  /** Whether to expose create/edit/delete. Default true. */
  editable?: boolean;
  /** Render extra row actions after the default edit/delete. */
  renderRowActions?: (record: Record<string, unknown>) => React.ReactNode;
  /** Custom page size, default 10. */
  pageSize?: number;
  /** Whether to show export button. */
  showExport?: boolean;
  /** Export callback. */
  onExport?: () => void;
  /** Row selection + batch operations. */
  batchActions?: BatchActionItem[];
  /** Custom row key field. Default 'id'. */
  rowKey?: string;
}

interface ListResponse {
  data?: Record<string, unknown>[];
  total: number;
  page: number;
  size: number;
}

// ---------- Component ----------

export default function CrudListPage({
  resource,
  title,
  singular,
  columns,
  fields,
  searchPlaceholder = '搜索...',
  filters,
  extraFilters,
  editable = true,
  renderRowActions,
  pageSize = 10,
  showExport,
  onExport,
  batchActions,
  rowKey = 'id',
}: CrudListPageProps) {
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(pageSize);
  const [search, setSearch] = useState('');
  const [filterValues, setFilterValues] = useState<Record<string, string | number | null | undefined>>({});
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Record<string, unknown> | null>(null);
  const [slowLoad, setSlowLoad] = useState(false);
  const [loadAttempt, setLoadAttempt] = useState(0);
  const [form] = Form.useForm();

  // ---------- Deletion confirmation ----------
  const [deleteTarget, setDeleteTarget] = useState<Record<string, unknown> | null>(null);

  // ---------- Row selection ----------
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // ---------- Query ----------
  const listKey = useMemo(
    () => ['crud', resource, page, size, search, JSON.stringify({ ...filterValues, ...extraFilters })],
    [filterValues, extraFilters, resource, page, size, search],
  );
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: listKey,
    queryFn: async () => {
      const params: Record<string, string> = {
        page: String(page),
        size: String(size),
      };
      if (search) params.search = search;
      // Apply multi-dimension filters
      Object.entries(filterValues).forEach(([k, v]) => {
        if (v !== undefined && v !== null && v !== '') {
          params[k] = String(v);
        }
      });
      if (extraFilters) Object.assign(params, extraFilters);
      const res = await apiClient.getPage<Record<string, unknown>>(`/v1${resource}`, params);
      return res as unknown as ListResponse;
    },
  });

  useEffect(() => {
    if (!isLoading) {
      setSlowLoad(false);
      return;
    }

    const timer = window.setTimeout(() => setSlowLoad(true), 8000);
    return () => window.clearTimeout(timer);
  }, [isLoading, loadAttempt]);

  // ---------- Mutations ----------
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
      const id = editing?.[rowKey];
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
      setDeleteTarget(null);
      qc.invalidateQueries({ queryKey: ['crud', resource] });
    },
    onError: (e: Error) => message.error(`删除失败: ${e.message}`),
  });

  // ---------- Table columns ----------
  const tableColumns = useMemo(() => {
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
        width: batchActions ? 160 : 140,
        fixed: 'right' as const,
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
              <Button
                size="small"
                type="link"
                danger
                icon={<DeleteOutlined />}
                onClick={() => setDeleteTarget(record)}
              >
                删除
              </Button>
            )}
          </Space>
        ),
      });
    }
    return cols as never;
  }, [columns, editable, renderRowActions, batchActions, form]);

  // ---------- Handlers ----------
  const handleSubmit = async () => {
    const values = await form.validateFields();
    if (editing) {
      updateMutation.mutate(values);
    } else {
      createMutation.mutate(values);
    }
  };

  const handleFilterChange = (key: string, value: string | number | null) => {
    setFilterValues((prev) => ({ ...prev, [key]: value }));
    setPage(1);
    setSelectedRowKeys([]);
  };

  const handleResetFilters = () => {
    setSearch('');
    setFilterValues({});
    setPage(1);
    setSelectedRowKeys([]);
  };

  const handleClearSelection = () => {
    setSelectedRowKeys([]);
  };

  const hasActiveFilters =
    search.length > 0 || Object.values(filterValues).some((v) => v !== undefined && v !== null && v !== '');

  // ---------- Render ----------
  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 12,
        }}
      >
        <h1 style={{
          fontFamily: 'var(--ds)',
          fontWeight: 700,
          fontSize: 'var(--text-h1)',
          color: 'var(--t1)',
          margin: 0,
        }}>{title}</h1>
        <Space>
          {showExport && (
            <Button icon={<DownloadOutlined />} onClick={onExport}
              style={{ fontFamily: 'var(--body)', fontSize: '0.8rem' }}>
              导出
            </Button>
          )}
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}
            style={{ fontFamily: 'var(--body)', fontSize: '0.8rem' }}>
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
              style={{ fontFamily: 'var(--body)', fontSize: '0.8rem' }}
            >
              新建{singular}
            </Button>
          )}
        </Space>
      </div>

      {/* Search + Filters */}
      {(searchPlaceholder || filters) && (
        <FilterBar
          search={search}
          onSearch={(v) => {
            setSearch(v);
            setPage(1);
            setSelectedRowKeys([]);
          }}
          searchPlaceholder={searchPlaceholder ?? undefined}
          filters={filters}
          filterValues={filterValues}
          onFilterChange={handleFilterChange}
          onReset={hasActiveFilters ? handleResetFilters : undefined}
        />
      )}

      {/* Batch actions */}
      {batchActions && batchActions.length > 0 && (
        <BatchActionBar
          selectedCount={selectedRowKeys.length}
          actions={batchActions}
          onClearSelection={handleClearSelection}
        />
      )}

      {/* Table / bounded error state */}
      {isError || slowLoad ? (
        <Alert
          type={isError ? 'error' : 'warning'}
          showIcon
          title={isError ? `无法加载${title}` : `${title}加载时间过长`}
          description={
            isError
              ? error instanceof Error ? error.message : '数据请求失败，请稍后重试。'
              : '服务暂时没有响应。请重新加载；如果问题持续，请检查网络或访问权限。'
          }
          action={(
            <Button
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => {
                setSlowLoad(false);
                setLoadAttempt((attempt) => attempt + 1);
                void refetch();
              }}
            >
              重新加载
            </Button>
          )}
        />
      ) : (
      <Table
        rowKey={rowKey}
        loading={isLoading}
        dataSource={data?.data}
        columns={tableColumns}
        scroll={{ x: 'max-content' }}
        rowSelection={
          batchActions && batchActions.length > 0
            ? {
                selectedRowKeys,
                onChange: (keys) => setSelectedRowKeys(keys),
              }
            : undefined
        }
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
      )}

      {/* Create / Edit modal */}
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
                <InputNumber style={{ width: '100%' }} />
              ) : f.type === 'select' ? (
                <Select options={f.options} allowClear={!f.required} />
              ) : (
                <Input />
              )}
            </Form.Item>
          ))}
        </Form>
      </Modal>

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        open={!!deleteTarget}
        title={`删除${singular}`}
        content={`确定要删除此${singular}吗？此操作不可撤销。`}
        okType="danger"
        okText="确认删除"
        confirmLoading={deleteMutation.isPending}
        risk="high"
        onOk={() => deleteTarget && deleteMutation.mutate(deleteTarget[rowKey])}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  );
}

// ---------- Helpers ----------

/** Format an ISO timestamp for table display. */
export function fmtDate(v: unknown): string {
  if (!v) return '-';
  const s = String(v);
  const d = dayjs(s);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm') : s;
}

/** Format currency in CNY. */
export function fmtMoney(v: unknown): string {
  if (v === null || v === undefined || v === '') return '-';
  const n = Number(v);
  if (Number.isNaN(n)) return String(v);
  return `¥${n.toFixed(2)}`;
}
