'use client';

import { Card, Table, Tag, Result, Spin, Button } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeftOutlined } from '@ant-design/icons';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';

interface SupplierRow {
  supplier_id: number;
  supplier_name: string;
  supply_price: number | null;
  min_order_qty: number;
  spec_summary: string;
  package_length_cm: number | null;
  package_width_cm: number | null;
  package_height_cm: number | null;
  package_weight_kg: number | null;
}

interface ComparisonData {
  product_id: number;
  product_name: string;
  suppliers: SupplierRow[];
}

const columns = [
  { title: '供应商', dataIndex: 'supplier_name', key: 'supplier', width: 160 },
  {
    title: '供货价',
    dataIndex: 'supply_price',
    key: 'price',
    width: 120,
    render: (v: number | null) => (v ? `¥${v}` : '-'),
  },
  {
    title: '起订量',
    dataIndex: 'min_order_qty',
    key: 'moq',
    width: 100,
  },
  { title: '规格摘要', dataIndex: 'spec_summary', key: 'spec', ellipsis: true },
  {
    title: '包装尺寸',
    key: 'dimensions',
    width: 180,
    render: (_: unknown, r: SupplierRow) => {
      const d = [r.package_length_cm, r.package_width_cm, r.package_height_cm];
      return d.every(Boolean) ? `${d.join('×')}cm` : '-';
    },
  },
  {
    title: '包装重量',
    dataIndex: 'package_weight_kg',
    key: 'weight',
    width: 120,
    render: (v: number | null) => (v ? `${v}kg` : '-'),
  },
];

export default function SupplierComparisonPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;

  const { data, isLoading, isError } = useQuery({
    queryKey: ['product-supplier-comparison', id],
    queryFn: async () => {
      const res = await apiClient.get<ComparisonData>(`/v1/products/${id}/supplier-comparison`);
      return res.data;
    },
    retry: false,
  });

  if (isLoading) {
    return (
      <PageContainer title="供应商对比">
        <Card style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}><div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div></Card>
      </PageContainer>
    );
  }

  if (isError || !data) {
    return (
      <PageContainer title="供应商对比">
        <Result status="warning" title="加载失败" subTitle="无法获取供应商对比数据，请检查商品ID或稍后重试" />
      </PageContainer>
    );
  }

  return (
    <PageContainer title={`供应商对比 — ${data.product_name}`}>
      <Button icon={<ArrowLeftOutlined />} onClick={() => router.push(`/products/${id}`)} style={{ marginBottom: 16 }}>
        返回商品
      </Button>

      {data.suppliers.length === 0 ? (
        <Card style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
          <Result
            status="info"
            title="暂无供应商数据"
            subTitle="该商品尚未关联任何供应商。请在「商品-供应商管理」中添加关联。"
          />
        </Card>
      ) : (
        <Table
          dataSource={data.suppliers}
          columns={columns}
          rowKey="supplier_id"
          pagination={false}
          bordered
          summary={() => (
            <Table.Summary.Row>
              <Table.Summary.Cell index={0}>对比</Table.Summary.Cell>
              <Table.Summary.Cell index={1}>
                {data.suppliers.every(s => s.supply_price) ? (
                  <Tag color="blue">价差 ¥{range(data.suppliers.map(s => s.supply_price!))}</Tag>
                ) : '-'}
              </Table.Summary.Cell>
              <Table.Summary.Cell index={2} colSpan={4}>
                {data.suppliers.length} 家供应商
              </Table.Summary.Cell>
            </Table.Summary.Row>
          )}
        />
      )}
    </PageContainer>
  );
}

function range(nums: number[]): string {
  const min = Math.min(...nums);
  const max = Math.max(...nums);
  return min === max ? `${min}` : `${min} ~ ${max}`;
}
