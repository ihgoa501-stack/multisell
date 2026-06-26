'use client';

import { useMemo, useState } from 'react';
import { Form, Input, Modal, Select, Space, message } from 'antd';
import dayjs from 'dayjs';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import ConfirmDialog from '@/components/ui/ConfirmDialog';

// ---------- Helpers ----------

function fmtDate(v: unknown): string {
  if (!v) return '-';
  const s = String(v);
  const d = dayjs(s);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm') : s;
}

// ---------- Types ----------

interface ProductRecord {
  id: number;
  name: string;
  subtitle: string;
  brand_id: number;
  category_id: number;
  unit: string;
  status: string;
  cargo_type: string;
  main_image: string;
  created_at: string;
  [key: string]: unknown;
}

interface ListResponse {
  data?: ProductRecord[];
  total: number;
  page: number;
  size: number;
}

// ---------- Badge renderer ----------

function StatusBadge({ status }: { status: string }) {
  const style: Record<string, string | number> = {
    padding: '1px 6px',
    borderRadius: 100,
    fontSize: '0.62rem',
    fontWeight: 500,
    display: 'inline-block',
  };

  switch (status) {
    case 'active':
    case 'published':
      return (
        <span style={{ ...style, background: 'rgba(52,211,153,0.08)', color: 'var(--g4)' }}>
          {status === 'active' ? '上架' : '已发布'}
        </span>
      );
    case 'inactive':
    case 'draft':
      return (
        <span style={{ ...style, background: 'rgba(34,211,238,0.08)', color: 'var(--c4)' }}>
          {status === 'inactive' ? '下架' : '草稿'}
        </span>
      );
    case 'pending':
    case 'review':
      return (
        <span style={{ ...style, background: 'rgba(99,102,241,0.08)', color: 'var(--i4)' }}>
          待审核
        </span>
      );
    case 'failed':
      return (
        <span style={{ ...style, background: 'rgba(248,113,113,0.08)', color: 'var(--r4)' }}>
          失败
        </span>
      );
    case 'processing':
    case 'ai_processing':
      return (
        <span style={{ ...style, background: 'rgba(34,211,238,0.08)', color: 'var(--c4)' }}>
          AI处理中
        </span>
      );
    default:
      return <span style={{ ...style, background: 'rgba(156,163,175,0.08)', color: 'var(--t3)' }}>{status}</span>;
  }
}

// ---------- Columns config ----------

const columns = [
  { title: 'ID', dataIndex: 'id' as const, width: 70 },
  { title: '商品名称', dataIndex: 'name' as const, width: 200 },
  { title: '副标题', dataIndex: 'subtitle' as const, width: 200 },
  { title: '品牌ID', dataIndex: 'brand_id' as const, width: 90 },
  { title: '分类ID', dataIndex: 'category_id' as const, width: 90 },
  { title: '单位', dataIndex: 'unit' as const, width: 80 },
  { title: '状态', dataIndex: 'status' as const, width: 100, render: (v: unknown) => <StatusBadge status={String(v ?? '')} /> },
  { title: '货品类型', dataIndex: 'cargo_type' as const, width: 110 },
  { title: '创建时间', dataIndex: 'created_at' as const, width: 160, render: fmtDate },
];

interface FieldConfig {
  name: string;
  label: string;
  type?: 'text' | 'number' | 'textarea' | 'select';
  required?: boolean;
  options?: { label: string; value: string | number }[];
  initialValue?: unknown;
}

const fields: FieldConfig[] = [
  { name: 'name', label: '商品名称', required: true },
  { name: 'subtitle', label: '副标题' },
  { name: 'brand_id', label: '品牌ID', type: 'number' },
  { name: 'category_id', label: '分类ID', type: 'number' },
  { name: 'unit', label: '单位' },
  { name: 'status', label: '状态', initialValue: 'active' },
  { name: 'main_image', label: '主图URL' },
  { name: 'cargo_type', label: '货品类型' },
];

// ---------- Page Component ----------

export default function ProductsPage() {
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(10);
  const [search, setSearch] = useState('');
  const [filterValues, setFilterValues] = useState<Record<string, string | number | null | undefined>>({});
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<ProductRecord | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<ProductRecord | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [form] = Form.useForm();

  // ---------- Query ----------
  const listKey = useMemo(
    () => ['crud', '/products', page, size, search, JSON.stringify(filterValues)],
    [page, size, search, filterValues],
  );

  const { data, isLoading, refetch } = useQuery({
    queryKey: listKey,
    queryFn: async () => {
      const params: Record<string, string> = {
        page: String(page),
        size: String(size),
      };
      if (search) params.search = search;
      Object.entries(filterValues).forEach(([k, v]) => {
        if (v !== undefined && v !== null && v !== '') {
          params[k] = String(v);
        }
      });
      const res = await apiClient.getPage<ProductRecord>('/v1/products', params);
      return res as unknown as ListResponse;
    },
  });

  // ---------- Mutations ----------
  const createMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) =>
      apiClient.post('/v1/products', values),
    onSuccess: () => {
      message.success('已创建');
      setModalOpen(false);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['crud', '/products'] });
    },
    onError: (e: Error) => message.error(`创建失败: ${e.message}`),
  });

  const updateMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) => {
      const id = editing?.id;
      return apiClient.put(`/v1/products/${id}`, values);
    },
    onSuccess: () => {
      message.success('已更新');
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['crud', '/products'] });
    },
    onError: (e: Error) => message.error(`更新失败: ${e.message}`),
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: unknown) => apiClient.delete(`/v1/products/${id}`),
    onSuccess: () => {
      message.success('已删除');
      setDeleteTarget(null);
      qc.invalidateQueries({ queryKey: ['crud', '/products'] });
    },
    onError: (e: Error) => message.error(`删除失败: ${e.message}`),
  });

  // ---------- Handlers ----------
  const handleSearch = (value: string) => {
    setSearch(value);
    setPage(1);
    setSelectedRowKeys([]);
  };

  const handleCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const handleEdit = (record: ProductRecord) => {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    const values = await form.validateFields();
    if (editing) {
      updateMutation.mutate(values);
    } else {
      createMutation.mutate(values);
    }
  };

  const handleClearSelection = () => setSelectedRowKeys([]);

  // ---------- Pagination ----------
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  // ---------- Render ----------
  const records = data?.data ?? [];

  return (
    <div
      style={{
        padding: '16px 20px',
        background: 'var(--bg)',
        minHeight: '100%',
        display: 'flex',
        flexDirection: 'column',
        fontFamily: 'var(--body)',
        color: 'var(--t1)',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <h1 style={{ fontSize: 22, fontWeight: 600, margin: 0, fontFamily: 'var(--ds)' }}>
          商品
        </h1>
        <Space size={8}>
          <button
            onClick={() => refetch()}
            style={{
              padding: '4px 12px',
              borderRadius: 6,
              border: '1px solid var(--bd)',
              background: 'var(--s2)',
              color: 'var(--t1)',
              fontSize: 13,
              cursor: 'pointer',
              fontFamily: 'var(--body)',
              lineHeight: '22px',
            }}
          >
            刷新
          </button>
          <button
            onClick={handleCreate}
            style={{
              padding: '4px 12px',
              borderRadius: 6,
              border: 'none',
              background: 'var(--i5)',
              color: '#fff',
              fontSize: 13,
              cursor: 'pointer',
              fontFamily: 'var(--body)',
              fontWeight: 500,
              lineHeight: '22px',
            }}
          >
            + 新建商品
          </button>
        </Space>
      </div>

      {/* Search bar */}
      <div style={{ marginBottom: 12 }}>
        <input
          value={search}
          onChange={(e) => handleSearch(e.target.value)}
          placeholder="搜索商品名称 / 编码 / 副标题..."
          style={{
            width: '100%',
            maxWidth: 360,
            padding: '5px 12px',
            borderRadius: 6,
            border: '1px solid var(--bd)',
            background: 'var(--s2)',
            color: 'var(--t1)',
            fontSize: 13,
            fontFamily: 'var(--body)',
            outline: 'none',
            lineHeight: '22px',
          }}
        />
      </div>

      {/* Bulk selection info */}
      {selectedRowKeys.length > 0 && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '8px 12px',
            background: 'var(--s2)',
            borderRadius: 6,
            marginBottom: 8,
            fontSize: 13,
            color: 'var(--t2)',
          }}
        >
          <span>已选 {selectedRowKeys.length} 项</span>
          <button
            onClick={handleClearSelection}
            style={{
              padding: '2px 8px',
              borderRadius: 4,
              border: '1px solid var(--bd)',
              background: 'var(--s1)',
              color: 'var(--t2)',
              fontSize: 12,
              cursor: 'pointer',
              fontFamily: 'var(--body)',
            }}
          >
            取消选择
          </button>
        </div>
      )}

      {/* Loading state */}
      {isLoading && (
        <div
          style={{
            padding: '40px 0',
            textAlign: 'center',
            color: 'var(--t3)',
            fontSize: 14,
          }}
        >
          加载中...
        </div>
      )}

      {/* Table */}
      {!isLoading && (
        <div
          style={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            border: '1px solid var(--bd)',
            borderRadius: 8,
            overflow: 'hidden',
            background: 'var(--s1)',
          }}
        >
          {/* Table header */}
          <div
            style={{
              display: 'flex',
              background: 'var(--s2)',
              borderBottom: '1px solid var(--bd)',
              fontFamily: 'var(--ds)',
              fontSize: 12,
              fontWeight: 600,
              color: 'var(--t4)',
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
            }}
          >
            {columns.map((col) => (
              <div
                key={col.dataIndex}
                style={{
                  padding: '10px 12px',
                  width: col.width,
                  minWidth: col.width,
                  flexShrink: 0,
                }}
              >
                {col.title}
              </div>
            ))}
            <div
              style={{
                padding: '10px 12px',
                flex: 1,
                minWidth: 140,
              }}
            >
              操作
            </div>
          </div>

          {/* Table rows */}
          {records.length === 0 && (
            <div
              style={{
                padding: 40,
                textAlign: 'center',
                color: 'var(--t3)',
                fontSize: 14,
              }}
            >
              暂无数据
            </div>
          )}

          {records.map((record, idx) => (
            <div
              key={record.id}
              style={{
                display: 'flex',
                borderBottom: idx < records.length - 1 ? '1px solid var(--bd)' : 'none',
                fontSize: 13,
                transition: 'background 0.15s',
                cursor: 'default',
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLElement).style.background = 'var(--s1)';
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLElement).style.background = '';
              }}
            >
              {columns.map((col) => (
                <div
                  key={col.dataIndex}
                  style={{
                    padding: '10px 12px',
                    width: col.width,
                    minWidth: col.width,
                    flexShrink: 0,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {col.render
                    ? col.render(record[col.dataIndex])
                    : String(record[col.dataIndex] ?? '-')}
                </div>
              ))}
              <div
                style={{
                  padding: '10px 12px',
                  flex: 1,
                  minWidth: 140,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                }}
              >
                <button
                  onClick={() => handleEdit(record)}
                  style={{
                    padding: '2px 10px',
                    borderRadius: 6,
                    border: '1px solid var(--bd)',
                    background: 'var(--s2)',
                    color: 'var(--t2)',
                    fontSize: 12,
                    cursor: 'pointer',
                    fontFamily: 'var(--body)',
                    lineHeight: '22px',
                  }}
                >
                  编辑
                </button>
                <button
                  onClick={() => setDeleteTarget(record)}
                  style={{
                    padding: '2px 10px',
                    borderRadius: 6,
                    border: '1px solid var(--r4)',
                    background: 'transparent',
                    color: 'var(--r4)',
                    fontSize: 12,
                    cursor: 'pointer',
                    fontFamily: 'var(--body)',
                    lineHeight: '22px',
                  }}
                >
                  删除
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {!isLoading && total > 0 && (
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '12px 0',
            fontSize: 13,
            color: 'var(--t2)',
          }}
        >
          <span>共 {total} 条</span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <button
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
              style={{
                padding: '4px 10px',
                borderRadius: 6,
                border: '1px solid var(--bd)',
                background: 'var(--s2)',
                color: page <= 1 ? 'var(--t3)' : 'var(--t1)',
                fontSize: 12,
                cursor: page <= 1 ? 'not-allowed' : 'pointer',
                fontFamily: 'var(--body)',
                lineHeight: '22px',
              }}
            >
              上一页
            </button>
            <span style={{ padding: '0 8px', color: 'var(--t3)', fontSize: 12 }}>
              {page} / {totalPages}
            </span>
            <button
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
              style={{
                padding: '4px 10px',
                borderRadius: 6,
                border: '1px solid var(--bd)',
                background: 'var(--s2)',
                color: page >= totalPages ? 'var(--t3)' : 'var(--t1)',
                fontSize: 12,
                cursor: page >= totalPages ? 'not-allowed' : 'pointer',
                fontFamily: 'var(--body)',
                lineHeight: '22px',
              }}
            >
              下一页
            </button>
            <select
              value={size}
              onChange={(e) => {
                setSize(Number(e.target.value));
                setPage(1);
              }}
              style={{
                marginLeft: 8,
                padding: '4px 8px',
                borderRadius: 6,
                border: '1px solid var(--bd)',
                background: 'var(--s2)',
                color: 'var(--t1)',
                fontSize: 12,
                fontFamily: 'var(--body)',
                outline: 'none',
                lineHeight: '22px',
              }}
            >
              {[10, 20, 50, 100].map((n) => (
                <option key={n} value={n}>
                  {n}条/页
                </option>
              ))}
            </select>
          </div>
        </div>
      )}

      {/* Create / Edit modal */}
      <Modal
        title={editing ? '编辑商品' : '新建商品'}
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
              initialValue={f.initialValue}
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

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        open={!!deleteTarget}
        title="删除商品"
        content="确定要删除此商品吗？此操作不可撤销。"
        okType="danger"
        okText="确认删除"
        confirmLoading={deleteMutation.isPending}
        risk="high"
        onOk={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  );
}
