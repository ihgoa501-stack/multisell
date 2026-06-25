'use client';

import { useState } from 'react';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';
import { Table, Tag, Tabs, Typography, Space } from 'antd';
import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';

const { Text } = Typography;

// --- Types ---
interface BinLocation {
  id: number;
  location_code: string;
  warehouse: string;
  sku_id?: number;
  capacity: number;
  used: number;
  status: string;
}

interface InventoryTransfer {
  id: number;
  from_warehouse: string;
  to_warehouse: string;
  sku_id: number;
  quantity: number;
  status: string;
  carrier?: string;
  estimated_arrival?: string;
  created_at: string;
}

interface PageResponse<T> {
  data: T[];
  total: number;
  page: number;
  size: number;
}

export default function InventoryPage() {
  // Warehouse locations
  const { data: locData, isLoading: locLoading } = useQuery({
    queryKey: ['inventory-locations'],
    queryFn: async () => {
      const res = await apiClient.getPage<BinLocation>('/v1/inventory/locations', { page: '1', size: '50' });
      return res;
    },
  });

  // Transfers
  const { data: xferData, isLoading: xferLoading } = useQuery({
    queryKey: ['inventory-transfers'],
    queryFn: async () => {
      const res = await apiClient.getPage<InventoryTransfer>('/v1/inventory/transfers', { page: '1', size: '50' });
      return res;
    },
  });

  const locColumns = [
    { title: '库位码', dataIndex: 'location_code', width: 130 },
    { title: '仓库', dataIndex: 'warehouse', width: 120 },
    { title: 'SKU', dataIndex: 'sku_id', width: 90, render: (v?: number) => v ?? '-' },
    { title: '容量', dataIndex: 'capacity', width: 80 },
    { title: '已用', dataIndex: 'used', width: 80 },
    {
      title: '利用率',
      key: 'usage',
      width: 100,
      render: (_: unknown, r: BinLocation) => {
        const pct = r.capacity > 0 ? Math.round((r.used / r.capacity) * 100) : 0;
        return (
          <Tag color={pct >= 90 ? 'red' : pct >= 70 ? 'orange' : 'green'}>
            {pct}%
          </Tag>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: string) => {
        const color = v === 'available' ? 'green' : v === 'occupied' ? 'blue' : v === 'maintenance' ? 'orange' : 'default';
        return <Tag color={color}>{v}</Tag>;
      },
    },
  ];

  const transferStatusColor = (status: string): string => {
    if (status === 'completed') return 'green';
    if (status === 'in_transit') return 'blue';
    if (status === 'draft') return 'default';
    if (status === 'cancelled') return 'red';
    return 'default';
  };

  const xferColumns = [
    { title: '源仓库', dataIndex: 'from_warehouse', width: 120 },
    { title: '目标仓库', dataIndex: 'to_warehouse', width: 120 },
    { title: 'SKU', dataIndex: 'sku_id', width: 90 },
    { title: '数量', dataIndex: 'quantity', width: 80 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: string) => <Tag color={transferStatusColor(v)}>{v}</Tag>,
    },
    { title: '承运方', dataIndex: 'carrier', width: 100, render: (v?: string) => v ?? '-' },
    { title: '预计到达', dataIndex: 'estimated_arrival', width: 160, render: (v?: string) => (v ? fmtDate(v) : '-') },
  ];

  return (
    <>
      <CrudListPage
        resource="/inventory"
        title="库存"
        singular="库存"
        searchPlaceholder="搜索 SKU ID / 批次号 / 仓库..."
        columns={[
          { title: 'ID', dataIndex: 'id', width: 70 },
          { title: 'SKU ID', dataIndex: 'sku_id', width: 90 },
          { title: '仓库', dataIndex: 'warehouse', width: 120 },
          { title: '可用库存', dataIndex: 'quantity', width: 100 },
          { title: '锁定库存', dataIndex: 'locked_quantity', width: 100 },
          { title: '批次号', dataIndex: 'batch_no', width: 150 },
          { title: '更新时间', dataIndex: 'updated_at', width: 160, render: fmtDate },
        ]}
        fields={[
          { name: 'sku_id', label: 'SKU ID', type: 'number', required: true },
          { name: 'warehouse', label: '仓库', required: true },
          { name: 'quantity', label: '可用库存', type: 'number', initialValue: 0 },
          { name: 'locked_quantity', label: '锁定库存', type: 'number', initialValue: 0 },
          { name: 'batch_no', label: '批次号' },
        ]}
      />

      <div style={{ background: 'var(--bg)', padding: '0 20px 24px', marginTop: -8 }}>
        <Tabs
          defaultActiveKey="locations"
          items={[
            {
              key: 'locations',
              label: '库位管理',
              children: (
                <Table
                  rowKey="id"
                  loading={locLoading}
                  dataSource={locData?.data ?? []}
                  columns={locColumns}
                  size="small"
                  scroll={{ x: 'max-content' }}
                  pagination={{ pageSize: 10, total: locData?.total ?? 0, showSizeChanger: false }}
                />
              ),
            },
            {
              key: 'transfers',
              label: '调拨管理',
              children: (
                <Table
                  rowKey="id"
                  loading={xferLoading}
                  dataSource={xferData?.data ?? []}
                  columns={xferColumns}
                  size="small"
                  scroll={{ x: 'max-content' }}
                  pagination={{ pageSize: 10, total: xferData?.total ?? 0, showSizeChanger: false }}
                />
              ),
            },
          ]}
        />
      </div>
    </>
  );
}
