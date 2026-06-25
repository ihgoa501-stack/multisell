'use client';

import { Card, Descriptions, Table, Timeline, Spin, Result, Button, Space, Tag } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeftOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';

interface OrderItem {
  id: number;
  sku_id: number;
  product_name: string;
  sku_code: string;
  spec_desc: string;
  unit_price: number;
  quantity: number;
  subtotal: number;
}

interface OrderStatusLog {
  id: number;
  from_status: string;
  to_status: string;
  created_at: string;
}

interface Order {
  id: number;
  order_no: string;
  platform_id?: number;
  status: string;
  tracking_number: string;
  recipient_name: string;
  recipient_phone: string;
  shipping_address: string;
  total_amount: number;
  shipping_fee: number;
  pay_amount: number;
  platform_fee: number;
  payment_fee: number;
  other_fee: number;
  product_cost: number;
  profit_amount: number;
  profit_margin: number;
  payment_method: string;
  remark: string;
  created_at: string;
}

interface OrderDetailResponse {
  order: Order;
  items: OrderItem[];
  status_logs: OrderStatusLog[];
}

export default function OrderDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;

  const { data, isLoading, isError } = useQuery({
    queryKey: ['order', id],
    queryFn: async () => {
      const res = await apiClient.get<OrderDetailResponse>(`/v1/order/${id}`);
      return res.data;
    },
    retry: false,
  });

  const order = data?.order;
  const items = data?.items || [];
  const logs = data?.status_logs || [];

  const columns = [
    { title: '商品名称', dataIndex: 'product_name', key: 'product_name' },
    { title: 'SKU 编码', dataIndex: 'sku_code', key: 'sku_code' },
    { title: '规格', dataIndex: 'spec_desc', key: 'spec_desc' },
    {
      title: '单价',
      dataIndex: 'unit_price',
      key: 'unit_price',
      render: (v: number) => `¥${v.toFixed(2)}`,
    },
    { title: '数量', dataIndex: 'quantity', key: 'quantity' },
    {
      title: '小计',
      dataIndex: 'subtotal',
      key: 'subtotal',
      render: (v: number) => `¥${v.toFixed(2)}`,
    },
  ];

  return (
    <PageContainer title="订单详情">
      <Button
        icon={<ArrowLeftOutlined />}
        onClick={() => router.push('/orders')}
        style={{ marginBottom: 16 }}
      >
        返回列表
      </Button>

      {isLoading ? (
        <Card>
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin tip="加载中..." />
          </div>
        </Card>
      ) : isError || !data ? (
        <Card>
          <Result status="info" title="订单详情" subTitle="暂无详情数据或订单不存在" />
        </Card>
      ) : (
        <Space direction="vertical" size="middle" style={{ display: 'flex' }}>
          <Card title="基本信息">
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="订单号">{order?.order_no}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={order?.status === 'completed' ? 'green' : 'blue'}>{order?.status}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="收件人姓名">{order?.recipient_name}</Descriptions.Item>
              <Descriptions.Item label="收件人电话">{order?.recipient_phone}</Descriptions.Item>
              <Descriptions.Item label="收货地址" span={2}>
                {order?.shipping_address}
              </Descriptions.Item>
              <Descriptions.Item label="支付方式">{order?.payment_method || '-'}</Descriptions.Item>
              <Descriptions.Item label="运单号">{order?.tracking_number || '-'}</Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {order?.created_at ? dayjs(order.created_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="备注" span={2}>
                {order?.remark || '-'}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="费用与利润">
            <Descriptions bordered column={3} size="small">
              <Descriptions.Item label="商品总额">¥{order?.total_amount?.toFixed(2)}</Descriptions.Item>
              <Descriptions.Item label="运费">¥{order?.shipping_fee?.toFixed(2)}</Descriptions.Item>
              <Descriptions.Item label="实付金额">¥{order?.pay_amount?.toFixed(2)}</Descriptions.Item>
              <Descriptions.Item label="平台扣费">¥{order?.platform_fee?.toFixed(2)}</Descriptions.Item>
              <Descriptions.Item label="支付扣费">¥{order?.payment_fee?.toFixed(2)}</Descriptions.Item>
              <Descriptions.Item label="其它费用">¥{order?.other_fee?.toFixed(2)}</Descriptions.Item>
              <Descriptions.Item label="商品成本">¥{order?.product_cost?.toFixed(2)}</Descriptions.Item>
              <Descriptions.Item label="预估利润" span={2}>
                <span style={{ color: (order?.profit_amount || 0) >= 0 ? '#52c41a' : '#ff4d4f', fontWeight: 'bold' }}>
                  ¥{order?.profit_amount?.toFixed(2)} (利润率: {order?.profit_margin?.toFixed(2)}%)
                </span>
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="商品列表">
            <Table
              dataSource={items}
              columns={columns}
              rowKey="id"
              pagination={false}
              size="small"
            />
          </Card>

          <Card title="状态时间线">
            {logs.length === 0 ? (
              <div style={{ color: 'rgba(0,0,0,0.45)' }}>暂无状态变更记录</div>
            ) : (
              <Timeline
                items={logs.map((log) => ({
                  children: (
                    <div>
                      <strong>
                        {log.from_status || 'INIT'} &rarr; {log.to_status}
                      </strong>
                      <div style={{ color: 'rgba(0,0,0,0.45)', fontSize: 12, marginTop: 4 }}>
                        {dayjs(log.created_at).format('YYYY-MM-DD HH:mm:ss')}
                      </div>
                    </div>
                  ),
                }))}
              />
            )}
          </Card>
        </Space>
      )}
    </PageContainer>
  );
}
