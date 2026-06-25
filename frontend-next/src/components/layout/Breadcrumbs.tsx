'use client';

import { Breadcrumb } from 'antd';
import { usePathname } from 'next/navigation';
import Link from 'next/link';

const labelMap: Record<string, string> = {
  dashboard: 'Dashboard',
  ai: 'AI 指挥中心',
  agentos: 'AgentOS',
  'work-items': '工作队列',
  products: '商品',
  create: '创建',
  categories: '类目',
  brands: '品牌',
  sku: 'SKU',
  inventory: '库存',
  suppliers: '供应商',
  platforms: '平台',
  'platform-integrations': '平台集成',
  listings: '刊登',
  'listing-tasks': '刊登任务',
  workbench: '工作台',
  orders: '订单',
  'order-import': '订单导入',
  shipping: '物流',
  'platform-fees': '平台费用',
  finance: '财务总览',
  settlement: '结算',
  decision: '决策',
  allocation: '分配',
  cost: '成本分摊',
  agents: 'Agent 列表',
  actions: 'Action 中心',
  evolution: '进化',
  entropy: '熵监控',
  trace: 'Trace',
  exceptions: '异常',
  notifications: '通知',
  'image-gen': '图片生成',
  canvas: '画布',
  'import-batches': '批量导入',
  'operation-logs': '操作日志',
  search: '搜索',
  reports: '报表',
  aftersales: '售后',
  sourcing1688: '1688采购',
  settings: '系统设置',
  llm: 'LLM 配置',
  rbac: '权限管理',
  login: '登录',
};

export default function Breadcrumbs() {
  const pathname = usePathname();
  const segments = pathname.split('/').filter(Boolean);

  const items = [
    { title: <Link href="/">Home</Link> },
    ...segments.map((segment, index) => {
      const href = '/' + segments.slice(0, index + 1).join('/');
      const label = labelMap[segment] || segment;
      const isLast = index === segments.length - 1;
      return {
        title: isLast ? label : <Link href={href}>{label}</Link>,
      };
    }),
  ];

  return <Breadcrumb items={items} />;
}
