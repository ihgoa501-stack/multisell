'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Tag,
  Button,
  Upload,
  Radio,
  Space,
  message,
  Modal,
  Descriptions,
  Typography,
} from 'antd';
import {
  InboxOutlined,
  UploadOutlined,
  EyeOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import { getToken } from '@/lib/auth';
import type { Result, PageResult } from '@/types/api';

// ─────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────

interface ImportBatch {
  id: number;
  source_type: string;
  file_name: string;
  status: string;
  total_rows: number;
  success_count: number;
  error_count: number;
  error_summary?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

// ─────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────

const BATCH_TYPES = [
  { label: '商品', value: 'product' },
  { label: '订单', value: 'order' },
  { label: '库存', value: 'inventory' },
];

const STATUS_CONFIG: Record<string, { color: string; label: string }> = {
  pending: { color: 'default', label: '待处理' },
  processing: { color: 'processing', label: '处理中' },
  completed: { color: 'success', label: '已完成' },
  failed: { color: 'error', label: '失败' },
};

const SOURCE_TYPE_LABELS: Record<string, string> = {
  product: '商品',
  order: '订单',
  inventory: '库存',
};

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

// ─────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────

function statusTag(status: string) {
  const cfg = STATUS_CONFIG[status] || { color: 'default', label: status };
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

function sourceTypeLabel(t: string) {
  return SOURCE_TYPE_LABELS[t] || t;
}

// ─────────────────────────────────────────────────────────
// Page
// ─────────────────────────────────────────────────────────

export default function BatchOpsPage() {
  const [batchType, setBatchType] = useState<string>('product');
  const [uploading, setUploading] = useState(false);
  const [uploadResult, setUploadResult] = useState<ImportBatch | null>(null);

  // ── Batch list state ──
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [batches, setBatches] = useState<ImportBatch[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  // ── Detail modal ──
  const [detailBatch, setDetailBatch] = useState<ImportBatch | null>(null);

  // ── Load batch list ──
  const loadBatches = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiClient.getPage<ImportBatch>('/v1/importbatch', {
        page: String(page),
        size: String(pageSize),
      });
      if (res.data) {
        setBatches(res.data);
        setTotal(res.total);
      }
    } catch {
      message.error('加载批次列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  useEffect(() => {
    loadBatches();
  }, [loadBatches]);

  // ── Poll processing batches every 5s ──
  useEffect(() => {
    const processingBatches = batches.filter((b) => b.status === 'processing');
    if (processingBatches.length === 0) return;

    const poll = async () => {
      const updated = new Map<number, ImportBatch>();

      for (const b of processingBatches) {
        try {
          const res = await apiClient.get<ImportBatch>(`/v1/importbatch/${b.id}`);
          if (res.data) {
            updated.set(b.id, res.data);
          }
        } catch {
          // Individual poll failure — skip silently
        }
      }

      if (updated.size > 0) {
        setBatches((prev) =>
          prev.map((b) => (updated.has(b.id) ? updated.get(b.id)! : b)),
        );
      }
    };

    const interval = setInterval(poll, 5000);
    return () => clearInterval(interval);
  }, [batches]);

  // ── Upload handler ──
  const handleUpload = async (file: File): Promise<boolean> => {
    setUploading(true);
    setUploadResult(null);

    const formData = new FormData();
    formData.append('file', file);
    formData.append('source_type', batchType);

    try {
      const token = getToken();
      const res = await fetch(`${API_BASE}/v1/importbatch/upload`, {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        body: formData,
      });

      const result: Result<ImportBatch> = await res.json();

      if (result.code === 0 && result.data) {
        message.success(`上传成功！批次 #${result.data.id}`);
        setUploadResult(result.data);
        setPage(1);
        loadBatches();
      } else {
        message.error(result.message || '上传失败');
      }
    } catch {
      message.error('上传请求失败，请检查网络连接');
    } finally {
      setUploading(false);
    }

    return false; // prevent default upload behaviour
  };

  // ── Re-upload: set type and scroll to top ──
  const handleReUpload = (batch: ImportBatch) => {
    setBatchType(batch.source_type);
    window.scrollTo({ top: 0, behavior: 'smooth' });
    message.info('请在上方重新选择文件上传');
  };

  // ── Table columns ──
  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: '类型',
      dataIndex: 'source_type',
      width: 90,
      render: (v: string) => sourceTypeLabel(v),
    },
    {
      title: '文件名',
      dataIndex: 'file_name',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: string) => statusTag(v),
    },
    {
      title: '总行数',
      dataIndex: 'total_rows',
      width: 80,
      align: 'right' as const,
    },
    {
      title: '成功',
      dataIndex: 'success_count',
      width: 80,
      align: 'right' as const,
      render: (v: number) => (
        <span style={{ color: v > 0 ? '#52c41a' : undefined }}>{v}</span>
      ),
    },
    {
      title: '失败',
      dataIndex: 'error_count',
      width: 80,
      align: 'right' as const,
      render: (v: number) => (
        <span style={{ color: v > 0 ? '#ff4d4f' : undefined }}>{v}</span>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 160,
      render: (t: string) =>
        t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '-',
    },
    {
      title: '操作',
      width: 140,
      render: (_: unknown, record: ImportBatch) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => setDetailBatch(record)}
          >
            查看
          </Button>
          <Button
            type="link"
            size="small"
            icon={<UploadOutlined />}
            onClick={() => handleReUpload(record)}
          >
            重新上传
          </Button>
        </Space>
      ),
    },
  ];

  // ── Render ──
  const uploadArea = (
    <div style={{ marginBottom: 0 }}>
      <div style={{ marginBottom: 16 }}>
        <Typography.Text strong style={{ marginRight: 12 }}>
          批次类型：
        </Typography.Text>
        <Radio.Group
          value={batchType}
          onChange={(e) => setBatchType(e.target.value)}
          options={BATCH_TYPES}
          optionType="button"
          buttonStyle="solid"
          disabled={uploading}
        />
      </div>

      <Upload.Dragger
        accept=".csv,.xlsx,.xls"
        beforeUpload={(file) => {
          handleUpload(file as File);
          return false;
        }}
        showUploadList={false}
        disabled={uploading}
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined />
        </p>
        <p className="ant-upload-text">
          {uploading ? '上传中...' : '点击或拖拽文件到此区域上传'}
        </p>
        <p className="ant-upload-hint">支持 CSV、XLSX 格式，单文件不超过 20MB</p>
      </Upload.Dragger>

      {uploadResult && (
        <Card
          size="small"
          style={{ marginTop: 16, background: '#f6ffed', borderColor: '#b7eb8f' }}
        >
          <Space>
            <Tag color="success">已上传</Tag>
            <span>
              批次 #{uploadResult.id} —{' '}
              {sourceTypeLabel(uploadResult.source_type)} /{' '}
              {uploadResult.file_name}
            </span>
            {statusTag(uploadResult.status)}
          </Space>
        </Card>
      )}
    </div>
  );

  return (
    <div style={{ padding: 24 }}>
      {/* ── Header ── */}
      <Typography.Title
        level={3}
        style={{
          fontFamily: 'var(--ds)',
          fontWeight: 700,
          margin: '0 0 20px 0',
        }}
      >
        <UploadOutlined style={{ marginRight: 8 }} />
        批量运营
      </Typography.Title>

      {/* ── Upload card ── */}
      <Card
        size="small"
        title="上传批量文件"
        style={{ marginBottom: 20 }}
        styles={{ body: { padding: 20 } }}
      >
        {uploadArea}
      </Card>

      {/* ── Batch list card ── */}
      <Card
        size="small"
        title={
          <Space>
            <span>批次列表</span>
            <Button
              type="text"
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => loadBatches()}
            />
          </Space>
        }
        styles={{ body: { padding: 0 } }}
      >
        <Table<ImportBatch>
          rowKey="id"
          columns={columns}
          dataSource={batches}
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
          scroll={{ x: 900 }}
          size="middle"
          locale={{
            emptyText: '暂无批次记录',
          }}
        />
      </Card>

      {/* ── Detail modal ── */}
      <Modal
        title={`批次详情 #${detailBatch?.id}`}
        open={!!detailBatch}
        onCancel={() => setDetailBatch(null)}
        footer={
          <Button onClick={() => setDetailBatch(null)}>关闭</Button>
        }
        width={640}
      >
        {detailBatch && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="ID" span={2}>
              {detailBatch.id}
            </Descriptions.Item>
            <Descriptions.Item label="类型">
              {sourceTypeLabel(detailBatch.source_type)}
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              {statusTag(detailBatch.status)}
            </Descriptions.Item>
            <Descriptions.Item label="文件名" span={2}>
              {detailBatch.file_name || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="总行数">
              {detailBatch.total_rows}
            </Descriptions.Item>
            <Descriptions.Item label="成功数">
              {detailBatch.success_count}
            </Descriptions.Item>
            <Descriptions.Item label="失败数">
              {detailBatch.error_count}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间" span={2}>
              {detailBatch.created_at
                ? dayjs(detailBatch.created_at).format('YYYY-MM-DD HH:mm:ss')
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="更新时间" span={2}>
              {detailBatch.updated_at
                ? dayjs(detailBatch.updated_at).format('YYYY-MM-DD HH:mm:ss')
                : '-'}
            </Descriptions.Item>
            {detailBatch.created_by && (
              <Descriptions.Item label="创建人" span={2}>
                {detailBatch.created_by}
              </Descriptions.Item>
            )}
            {detailBatch.error_summary && (
              <Descriptions.Item label="错误摘要" span={2}>
                <pre
                  style={{
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    maxHeight: 200,
                    overflow: 'auto',
                    margin: 0,
                    fontSize: 12,
                  }}
                >
                  {detailBatch.error_summary}
                </pre>
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Modal>
    </div>
  );
}
