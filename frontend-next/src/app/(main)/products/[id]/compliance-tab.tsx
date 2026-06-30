'use client';

import { useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { Button, Input, message, Modal, Table, Tag } from 'antd';
import { CheckCircleOutlined, WarningOutlined, CloseCircleOutlined } from '@ant-design/icons';
import apiClient from '@/lib/api-client';

interface EvidenceItem {
  rule: string;
  field?: string;
  value?: string;
  source: string;
}

interface CheckResult {
  id: number;
  product_id: number;
  status: string;
  risk_level: string | null;
  evidence: EvidenceItem[];
  scanned_at: string;
  is_suppressed: boolean;
}

export default function ComplianceTab({ productId }: { productId: string }) {
  const [results, setResults] = useState<CheckResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [scanning, setScanning] = useState(false);

  const fetchResults = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get<{ data: CheckResult[] }>(`/v1/compliance/results?product_id=${productId}&page=1&size=10`);
      setResults(res.data?.data ?? []);
    } catch {
      // Silent — listing may not have compliance data yet.
    } finally {
      setLoading(false);
    }
  };

  const triggerScan = async () => {
    setScanning(true);
    try {
      await apiClient.post('/v1/compliance/check', {
        product_id: productId,
        // ponytail: hardcoded for now — fetched from product context in Phase 2.
        product_name: '',
        category: '',
        country: '',
        platform: '',
      });
      message.success('合规检查已触发');
      await fetchResults();
    } catch {
      message.error('合规检查失败');
    } finally {
      setScanning(false);
    }
  };

  const handleSuppress = async (id: number, reason: string) => {
    try {
      await apiClient.put(`/v1/compliance/results/${id}/suppress`, { reason });
      message.success('已标记为已确认');
      await fetchResults();
    } catch {
      message.error('确认失败');
    }
  };

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await apiClient.get<{ data: CheckResult[] }>(`/v1/compliance/results?product_id=${productId}&page=1&size=10`);
        if (!cancelled) setResults(res.data?.data ?? []);
      } catch {
        // Silent — listing may not have compliance data yet.
      }
    })();
    return () => { cancelled = true; };
  }, [productId]);

  const statusTag = (status: string) => {
    const map: Record<string, { color: string; icon: ReactNode }> = {
      pass: { color: 'green', icon: <CheckCircleOutlined /> },
      warning: { color: 'orange', icon: <WarningOutlined /> },
      fail: { color: 'red', icon: <CloseCircleOutlined /> },
    };
    const s = map[status] ?? { color: 'default', icon: null };
    const labelMap: Record<string, string> = {
      pass: '通过',
      warning: '警告',
      fail: '不合规',
    };
    return (
      <Tag color={s.color} icon={s.icon ?? undefined}>
        {labelMap[status] ?? status.toUpperCase()}
      </Tag>
    );
  };

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', gap: 8, alignItems: 'center' }}>
        <Button type="primary" loading={scanning} onClick={triggerScan}>
          触发合规检查
        </Button>
        <Button onClick={fetchResults} loading={loading}>
          刷新
        </Button>
      </div>

      <Table
        dataSource={results}
        rowKey="id"
        loading={loading}
        columns={[
          { title: '状态', dataIndex: 'status', render: (v: string) => statusTag(v), width: 100 },
          { title: '风险等级', dataIndex: 'risk_level', render: (v: string | null) => v ?? '-', width: 100 },
          {
            title: '证据',
            dataIndex: 'evidence',
            ellipsis: true,
            render: (ev: EvidenceItem[]) =>
              ev?.map((e, i) => (
                <Tag key={i}>
                  {e.rule}: {e.value}
                </Tag>
              )),
          },
          {
            title: '扫描时间',
            dataIndex: 'scanned_at',
            render: (v: string) => new Date(v).toLocaleString(),
            width: 160,
          },
          {
            title: '操作',
            width: 110,
            render: (_: unknown, row: CheckResult) =>
              !row.is_suppressed ? (
                <Button
                  size="small"
                  onClick={() => {
                    Modal.confirm({
                      title: '确认该发现？',
                      content: (
                        <Input.TextArea id="suppress-reason" placeholder="原因..." rows={2} />
                      ),
                      onOk: () => {
                        const el = document.getElementById(
                          'suppress-reason',
                        ) as HTMLTextAreaElement;
                        return handleSuppress(row.id, el?.value ?? '已确认');
                      },
                    });
                  }}
                >
                  确认
                </Button>
              ) : (
                <Tag>已确认</Tag>
              ),
          },
        ]}
      />
    </div>
  );
}
