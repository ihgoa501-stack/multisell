'use client';

import { Card, Col, Row } from 'antd';
import {
  RobotOutlined,
  SafetyCertificateOutlined,
  ApiOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';

interface SettingCard {
  key: string;
  title: string;
  desc: string;
  icon: React.ReactNode;
  path: string;
  color: string;
}

const CARDS: SettingCard[] = [
  {
    key: 'llm',
    title: 'LLM 配置',
    desc: '管理各 Agent 的模型配置与参数',
    icon: <RobotOutlined style={{ fontSize: 32, color: '#1677ff' }} />,
    path: '/settings/llm',
    color: '#1677ff',
  },
  {
    key: 'rbac',
    title: 'RBAC 权限',
    desc: '角色与权限管理',
    icon: <SafetyCertificateOutlined style={{ fontSize: 32, color: '#722ed1' }} />,
    path: '/settings/rbac',
    color: '#722ed1',
  },
  {
    key: 'platform',
    title: '平台对接',
    desc: '电商平台集成与授权',
    icon: <ApiOutlined style={{ fontSize: 32, color: '#13c2c2' }} />,
    path: '/platform-integrations',
    color: '#13c2c2',
  },
  {
    key: 'system',
    title: '系统信息',
    desc: '版本、健康检查与运行状态',
    icon: <InfoCircleOutlined style={{ fontSize: 32, color: '#fa8c16' }} />,
    path: '/settings/system',
    color: '#fa8c16',
  },
];

export default function SettingsPage() {
  const router = useRouter();

  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '1rem', color: 'var(--t1)', margin: '0 0 16px 0' }}>设置</h1>
      <Row gutter={[16, 16]}>
        {CARDS.map((c) => (
          <Col key={c.key} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              onClick={() => router.push(c.path)}
              style={{ height: '100%' }}
            >
              <div style={{ marginBottom: 12 }}>{c.icon}</div>
              <h3 style={{ fontWeight: 600, marginBottom: 8 }}>{c.title}</h3>
              <div style={{ color: 'var(--t3)', fontSize: 13 }}>{c.desc}</div>
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  );
}
