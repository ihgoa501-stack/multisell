'use client';

import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';
import { Button, Form, InputNumber, Input, Modal, Row, Col, Statistic, Space, message, Table, Tag, Tabs } from 'antd';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';

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
        // P3: Safety config, allocation, dead stock (#201)
  const qc = useQueryClient();
  const { data: safetyConfigs } = useQuery({
    queryKey: ['inventory-safety-configs'],
    queryFn: async () => { const res = await apiClient.get<any[]>('/v1/inventory/safety-configs'); return res.data; },
  });
  const safetyItems: any[] = (safetyConfigs as any) ?? [];

  const upsertSafetyMut = useMutation({
    mutationFn: (values: any) => apiClient.put<any>('/v1/inventory/safety-config/'+values.sku_id, values),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['inventory-safety-configs'] }); message.success('安全库存配置已保存'); },
  });

  const { data: deadStockRes } = useQuery({
    queryKey: ['inventory-dead-stock'],
    queryFn: async () => { const res = await apiClient.post<any[]>('/v1/inventory/dead-stock/analyze', {}); return res; },
  });
  const deadStockItems: any[] = (deadStockRes as any) ?? [];

  const { mutate: analyzeDeadStock, isPending: deadLoading } = useMutation({
    mutationFn: () => apiClient.post<any>('/v1/inventory/dead-stock/analyze', {}),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['inventory-dead-stock'] }); message.success('死库存分析完成'); },
  });

  const safetyColumns = [
    { title: 'SKU ID', dataIndex: 'sku_id', key: 'sku_id' },
    { title: '最低库存', dataIndex: 'min_stock_level', key: 'min_stock_level' },
    { title: '最高库存', dataIndex: 'max_stock_level', key: 'max_stock_level' },
    { title: '提前期(天)', dataIndex: 'lead_time_days', key: 'lead_time_days' },
    { title: '安全天数', dataIndex: 'safety_days', key: 'safety_days' },
    { title: '日均销量', dataIndex: 'daily_avg_sales', key: 'daily_avg_sales' },
    { title: '自动补货', dataIndex: 'auto_reorder', key: 'auto_reorder', render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '开启' : '关闭'}</Tag> },
  ];

  const deadStockColumns = [
    { title: 'SKU ID', dataIndex: 'sku_id', key: 'sku_id' },
    { title: 'SKU编码', dataIndex: 'sku_code', key: 'sku_code' },
    { title: '产品名', dataIndex: 'product_name', key: 'product_name' },
    { title: '仓库', dataIndex: 'warehouse', key: 'warehouse' },
    { title: '数量', dataIndex: 'current_qty', key: 'current_qty' },
    { title: '未动天数', dataIndex: 'days_since_move', key: 'days_since_move' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (v: string) => <Tag color={v === 'dead' ? 'red' : 'orange'}>{v === 'dead' ? '死库存' : '滞销'}</Tag> },
    { title: '建议', dataIndex: 'suggestion', key: 'suggestion' },
  ];

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

  // P3: Safety config, allocation, dead stock (#201)
  const qc = useQueryClient();
  const { data: safetyConfigs } = useQuery({
    queryKey: ['inventory-safety-configs'],
    queryFn: async () => { const res = await apiClient.get<any[]>('/v1/inventory/safety-configs'); return res.data; },
  });
  const safetyItems: any[] = (safetyConfigs as any) ?? [];

  const upsertSafetyMut = useMutation({
    mutationFn: (values: any) => apiClient.put<any>('/v1/inventory/safety-config/'+values.sku_id, values),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['inventory-safety-configs'] }); message.success('安全库存配置已保存'); },
  });

  const { data: deadStockRes } = useQuery({
    queryKey: ['inventory-dead-stock'],
    queryFn: async () => { const res = await apiClient.post<any[]>('/v1/inventory/dead-stock/analyze', {}); return res; },
  });
  const deadStockItems: any[] = (deadStockRes as any) ?? [];

  const { mutate: analyzeDeadStock, isPending: deadLoading } = useMutation({
    mutationFn: () => apiClient.post<any>('/v1/inventory/dead-stock/analyze', {}),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['inventory-dead-stock'] }); message.success('死库存分析完成'); },
  });

  const safetyColumns = [
    { title: 'SKU ID', dataIndex: 'sku_id', key: 'sku_id' },
    { title: '最低库存', dataIndex: 'min_stock_level', key: 'min_stock_level' },
    { title: '最高库存', dataIndex: 'max_stock_level', key: 'max_stock_level' },
    { title: '提前期(天)', dataIndex: 'lead_time_days', key: 'lead_time_days' },
    { title: '安全天数', dataIndex: 'safety_days', key: 'safety_days' },
    { title: '日均销量', dataIndex: 'daily_avg_sales', key: 'daily_avg_sales' },
    { title: '自动补货', dataIndex: 'auto_reorder', key: 'auto_reorder', render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '开启' : '关闭'}</Tag> },
  ];

  const deadStockColumns = [
    { title: 'SKU ID', dataIndex: 'sku_id', key: 'sku_id' },
    { title: 'SKU编码', dataIndex: 'sku_code', key: 'sku_code' },
    { title: '产品名', dataIndex: 'product_name', key: 'product_name' },
    { title: '仓库', dataIndex: 'warehouse', key: 'warehouse' },
    { title: '数量', dataIndex: 'current_qty', key: 'current_qty' },
    { title: '未动天数', dataIndex: 'days_since_move', key: 'days_since_move' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (v: string) => <Tag color={v === 'dead' ? 'red' : 'orange'}>{v === 'dead' ? '死库存' : '滞销'}</Tag> },
    { title: '建议', dataIndex: 'suggestion', key: 'suggestion' },
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
            {
              key: 'safety',
              label: '安全库存',
              children: (
                <div>
                  <Table dataSource={safetyItems} columns={safetyColumns} rowKey="sku_id" size="small" pagination={false} style={{ marginBottom: 16 }} />
                  <p style={{ color: '#888', fontSize: 12 }}>此为只读视图。安全库存配置通过 SKU 详情编辑。</p>
                </div>
              ),
            },
            {
              key: 'deadstock',
              label: '死库存分析',
              children: (
                <div>
                  <Space style={{ marginBottom: 16 }}>
                    <Button type="primary" onClick={() => analyzeDeadStock()} loading={deadLoading}>运行死库存分析</Button>
                  </Space>
                  <Table dataSource={deadStockItems} columns={deadStockColumns} rowKey="sku_id" size="small" pagination={{ pageSize: 10 }} />
                </div>
              ),
            },
            {
              key: 'allocation',
              label: '多平台分配',
              children: (
                <div>
                  <Input.Search placeholder="输入SKU ID查看分配建议" style={{ width: 300, marginBottom: 16 }} onSearch={async (val) => {
                    if (!val) return;
                    try {
                      const res = await apiClient.get<any>('/v1/inventory/allocate/' + val);
                      Modal.info({
                        title: '分配建议',
                        width: 600,
                        content: (
                          <div>
                            <p>总可用: {res.data.total_available}</p>
                            <p>已分配: {res.data.reserved_total}</p>
                            <p>未分配: {res.data.unallocated}</p>
                            <Table dataSource={res.data.recommendations || []} rowKey="platform_id" size="small" pagination={false}
                              columns={[
                                { title: '平台ID', dataIndex: 'platform_id', key: 'platform_id' },
                                { title: '平台名称', dataIndex: 'platform_name', key: 'platform_name' },
                                { title: '销售占比(%)', dataIndex: 'sales_share', key: 'sales_share' },
                                { title: '当前库存', dataIndex: 'current_stock', key: 'current_stock' },
                                { title: '建议分配', dataIndex: 'recommended', key: 'recommended' },
                                { title: '优先级', dataIndex: 'priority', key: 'priority' },
                              ]}
                            />
                          </div>
                        ),
                      });
                    } catch(e: any) {
                      message.error(e?.message || '查询失败');
                    }
                  }} />
                </div>
              ),
            },
          ]}
        />
      </div>
    </>
  );
}
