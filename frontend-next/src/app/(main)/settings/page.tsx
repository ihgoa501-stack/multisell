'use client';

import { Card, Col, Row } from 'antd';
import {
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
    key: 'rbac',
    title: 'RBAC 权限',
    desc: '角色与权限管理',
    icon: <SafetyCertificateOutlined style={{ fontSize: 32, color: 'var(--i6)' }} />,
    path: '/settings/rbac',
    color: 'var(--i6)',
  },
  {
    key: 'platform',
    title: '平台对接',
    desc: '电商平台集成与授权',
    icon: <ApiOutlined style={{ fontSize: 32, color: 'var(--c4)' }} />,
    path: '/platform-integrations',
    color: 'var(--c4)',
  },
  {
    key: 'system',
    title: '系统信息',
    desc: '版本、健康检查与运行状态',
    icon: <InfoCircleOutlined style={{ fontSize: 32, color: 'var(--y4)' }} />,
    path: '/settings/system',
    color: 'var(--y4)',
  },
];

export default function SettingsPage() {
  const router = useRouter();

  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', color: 'var(--t1)', margin: '0 0 16px 0' }}>设置</h1>
      <Row gutter={[16, 16]}>
        {CARDS.map((c) => (
          <Col key={c.key} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              onClick={() => router.push(c.path)}
              style={{ height: '100%' }}
            >
              <div style={{ marginBottom: 'var(--space-md)' }}>{c.icon}</div>
              <h2 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: '1rem', marginBottom: 'var(--space-sm)' }}>{c.title}</h2>
              <div style={{ color: 'var(--t3)', fontSize: 'var(--text-body)' }}>{c.desc}</div>
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  );
}
