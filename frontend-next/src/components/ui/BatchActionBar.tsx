import { ReactNode } from 'react';
import { Button, Space, Typography } from 'antd';
import { ClearOutlined } from '@ant-design/icons';

const { Text } = Typography;

export interface BatchActionItem {
  key: string;
  label: string;
  /** @default 'default' */
  type?: 'primary' | 'default' | 'dashed' | 'link' | 'text';
  danger?: boolean;
  icon?: ReactNode;
  onClick: () => void;
}

export interface BatchActionBarProps {
  /** Number of selected items. */
  selectedCount: number;
  /** Available batch actions. */
  actions: BatchActionItem[];
  /** Callback to clear selection. */
  onClearSelection: () => void;
}

export default function BatchActionBar({
  selectedCount,
  actions,
  onClearSelection,
}: BatchActionBarProps) {
  if (selectedCount === 0) return null;

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '8px 16px',
        backgroundColor: '#e6f4ff',
        borderRadius: 6,
        marginBottom: 12,
        flexWrap: 'wrap',
      }}
    >
      <Text style={{ fontSize: 14 }}>
        已选 <Text strong>{selectedCount}</Text> 项
      </Text>

      <Space size="small">
        {actions.map((action) => (
          <Button
            key={action.key}
            size="small"
            type={action.type ?? 'default'}
            danger={action.danger}
            icon={action.icon}
            onClick={action.onClick}
          >
            {action.label}
          </Button>
        ))}
      </Space>

      <Button
        size="small"
        type="text"
        icon={<ClearOutlined />}
        onClick={onClearSelection}
        style={{ marginLeft: 'auto' }}
      >
        清除选择
      </Button>
    </div>
  );
}
