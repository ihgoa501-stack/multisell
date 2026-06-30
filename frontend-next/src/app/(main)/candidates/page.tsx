'use client';

import { useState, useEffect } from 'react';
import { Card, Table, Button, Modal, message, Badge, Typography } from 'antd';
import { DatabaseOutlined } from '@ant-design/icons';
import apiClient from '@/lib/api-client';

interface CompletenessItem {
  dimension: string;
  label: string;
  complete: boolean;
  reason: string;
}

interface Candidate {
  id: number;
  title: string;
  category_name: string;
  cost_price: number;
  target_price: number;
  currency: string;
  target_platform: string;
  status: string;
  completeness: CompletenessItem[];
  seed_data: boolean;
}

const statusBadgeMap: Record<string, { text: string; color: 'success' | 'warning' | 'default' }> = {
  complete: { text: '完整', color: 'success' },
  incomplete: { text: '资料不完整', color: 'warning' },
  pending: { text: '待检查', color: 'default' },
};

const platformLabelMap: Record<string, string> = {
  ozon: 'Ozon',
  shopee: 'Shopee',
  lazada: 'Lazada',
};

export default function CandidatesPage() {
  const [data, setData] = useState<Candidate[]>([]);
  const [loading, setLoading] = useState(false);
  const [seeding, setSeeding] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [detailModal, setDetailModal] = useState<Candidate | null>(null);

  const loadCandidates = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get<{
        code: number;
        message: string;
        data: Candidate[];
        total: number;
        page: number;
        size: number;
      }>('/v1/candidates', {
        page: String(page),
        size: String(pageSize),
      });
      if (res.data) {
        setData(res.data.data || []);
        setTotal(res.data.total || 0);
      }
    } catch {
      message.error('加载候选商品列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadCandidates();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize]);

  const handleSeed = async () => {
    setSeeding(true);
    try {
      const res = await apiClient.post('/v1/candidates/seed');
      if (res.code === 0) {
        message.success('种子数据生成成功');
        await loadCandidates();
      } else {
        message.error(res.message || '种子数据生成失败');
      }
    } catch {
      message.error('种子数据请求失败');
    } finally {
      setSeeding(false);
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
    },
    {
      title: '类目',
      dataIndex: 'category_name',
      width: 130,
      ellipsis: true,
      render: (val: string) => val || '-',
    },
    {
      title: '成本价',
      dataIndex: 'cost_price',
      width: 120,
      render: (price: number, record: Candidate) =>
        price != null ? `${price.toFixed(2)} ${record.currency || ''}` : '-',
    },
    {
      title: '目标价',
      dataIndex: 'target_price',
      width: 110,
      render: (price: number) =>
        price != null ? `¥${price.toFixed(2)}` : '-',
    },
    {
      title: '目标平台',
      dataIndex: 'target_platform',
      width: 120,
      render: (platform: string) => platformLabelMap[platform] || platform || '-',
    },
    {
      title: '完整性',
      dataIndex: 'status',
      width: 120,
      render: (status: string) => {
        const mapping = statusBadgeMap[status] || {
          text: status || '未知',
          color: 'default' as const,
        };
        return <Badge status={mapping.color} text={mapping.text} />;
      },
    },
    {
      title: '操作',
      width: 80,
      render: (_: unknown, record: Candidate) => (
        <Button
          type="link"
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            setDetailModal(record);
          }}
        >
          详情
        </Button>
      ),
    },
  ];

  return (
    <div
      style={{
        padding: '16px 20px',
        background: 'var(--bg)',
        minHeight: '100%',
      }}
    >
      <h1
        style={{
          fontFamily: 'var(--ds)',
          fontWeight: 700,
          fontSize: 'var(--text-h1)',
          color: 'var(--t1)',
          margin: '0 0 16px 0',
        }}
      >
        候选商品
      </h1>

      {/* Toolbar */}
      <Card
        size="small"
        style={{ marginBottom: 16 }}
        styles={{
          body: {
            padding: '12px 20px',
            display: 'flex',
            alignItems: 'center',
            gap: 12,
          },
        }}
      >
        <Button
          type="primary"
          icon={<DatabaseOutlined />}
          onClick={handleSeed}
          loading={seeding}
        >
          生成种子数据
        </Button>
      </Card>

      {/* Table */}
      <Card size="small" styles={{ body: { padding: 0 } }}>
        <Table<Candidate>
          rowKey="id"
          columns={columns}
          dataSource={data}
          loading={loading}
          onRow={(record) => ({
            onClick: () => setDetailModal(record),
            style: { cursor: 'pointer' },
          })}
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
          scroll={{ x: 850 }}
        />
      </Card>

      {/* Detail Modal */}
      <Modal
        title={detailModal ? `候选商品 #${detailModal.id}` : ''}
        open={!!detailModal}
        onCancel={() => setDetailModal(null)}
        footer={null}
        width={600}
      >
        {detailModal && (
          <div>
            <Typography.Title level={5} style={{ marginTop: 0 }}>
              {detailModal.title}
            </Typography.Title>

            <div
              style={{
                marginBottom: 16,
                color: 'var(--t2)',
                lineHeight: 1.8,
              }}
            >
              <div>类目：{detailModal.category_name || '-'}</div>
              <div>
                成本价：{detailModal.cost_price?.toFixed(2)}{' '}
                {detailModal.currency} → 目标价：¥
                {detailModal.target_price?.toFixed(2)}
              </div>
              <div>
                目标平台：{platformLabelMap[detailModal.target_platform] ||
                  detailModal.target_platform || '-'}
              </div>
              <div style={{ marginTop: 8 }}>
                完整性状态：
                <Badge
                  status={
                    statusBadgeMap[detailModal.status]?.color || 'default'
                  }
                  text={
                    statusBadgeMap[detailModal.status]?.text ||
                    detailModal.status ||
                    '未知'
                  }
                />
              </div>
            </div>

            <Typography.Title level={5}>完整性检查明细</Typography.Title>
            <table
              style={{ width: '100%', borderCollapse: 'collapse' }}
            >
              <thead>
                <tr
                  style={{
                    borderBottom: '1px solid var(--border)',
                    textAlign: 'left',
                  }}
                >
                  <th style={{ padding: '8px 12px', fontWeight: 600 }}>
                    维度
                  </th>
                  <th style={{ padding: '8px 12px', fontWeight: 600 }}>
                    结果
                  </th>
                  <th style={{ padding: '8px 12px', fontWeight: 600 }}>
                    说明
                  </th>
                </tr>
              </thead>
              <tbody>
                {detailModal.completeness?.map((item) => (
                  <tr
                    key={item.dimension}
                    style={{ borderBottom: '1px solid var(--border)' }}
                  >
                    <td style={{ padding: '8px 12px' }}>{item.label}</td>
                    <td style={{ padding: '8px 12px' }}>
                      {item.complete ? (
                        <span
                          style={{
                            color: '#52c41a',
                            fontSize: 16,
                            fontWeight: 700,
                          }}
                        >
                          &#10003;
                        </span>
                      ) : (
                        <span
                          style={{
                            color: '#ff4d4f',
                            fontSize: 16,
                            fontWeight: 700,
                          }}
                        >
                          &#10007;
                        </span>
                      )}
                    </td>
                    <td
                      style={{ padding: '8px 12px', color: 'var(--t2)' }}
                    >
                      {item.reason || '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Modal>
    </div>
  );
}
