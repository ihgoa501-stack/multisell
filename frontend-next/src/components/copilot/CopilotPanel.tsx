'use client';

import { Button, Typography } from 'antd';
import { useRouter } from 'next/navigation';

export default function CopilotPanel() {
  const router = useRouter();

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, padding: 20 }}>
      <Typography.Title level={4} style={{ margin: 0 }}>小Q</Typography.Title>
      <Typography.Paragraph type="secondary" style={{ margin: 0 }}>
        小Q是当前唯一的 Owner 经营 Agent。它只在明确的经营对象上读取事实、解释证据和提出可追溯建议，不提供无目标的旧式通用聊天。
      </Typography.Paragraph>
      <Button type="primary" onClick={() => router.push('/xiaoq')}>
        打开小Q工作台
      </Button>
    </div>
  );
}
