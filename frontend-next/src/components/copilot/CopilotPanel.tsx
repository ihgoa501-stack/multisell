'use client';

import { Drawer, Button, Input, Space } from 'antd';
import { RobotOutlined, SendOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useAppStore } from '@/stores/app-store';

interface CopilotPanelProps {
  open: boolean;
}

export default function CopilotPanel({ open }: CopilotPanelProps) {
  const { setCopilotOpen } = useAppStore();
  const [inputValue, setInputValue] = useState('');

  return (
    <Drawer
      title={
        <Space>
          <RobotOutlined />
          <span>AI Copilot</span>
        </Space>
      }
      placement="right"
      open={open}
      onClose={() => setCopilotOpen(false)}
      width={380}
      styles={{
        body: {
          padding: 0,
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
        },
      }}
    >
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
        }}
      >
        <div style={{ padding: 16, borderBottom: '1px solid #f0f0f0' }}>
          <Button block icon={<RobotOutlined />}>
            Explain this page
          </Button>
        </div>

        <div
          style={{
            flex: 1,
            padding: 16,
            overflowY: 'auto',
            display: 'flex',
            flexDirection: 'column',
            gap: 12,
          }}
        >
          <div
            style={{
              padding: 12,
              background: '#f6f6f6',
              borderRadius: 8,
              fontSize: 14,
              color: '#666',
            }}
          >
            Hi! I&apos;m your AI Copilot. Ask me anything about this page or
            your business data.
          </div>
        </div>

        <div
          style={{
            padding: 16,
            borderTop: '1px solid #f0f0f0',
            display: 'flex',
            gap: 8,
          }}
        >
          <Input
            placeholder="Ask a question..."
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onPressEnter={() => setInputValue('')}
            style={{ flex: 1 }}
          />
          <Button type="primary" icon={<SendOutlined />} />
        </div>
      </div>
    </Drawer>
  );
}
