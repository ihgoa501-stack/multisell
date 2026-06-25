import { Button, Empty, Space, Typography } from 'antd';
import { ReactNode } from 'react';

const { Text } = Typography;

export interface EmptyStateProps {
  /** Primary description — keep it short, actionable when possible. */
  description?: string;
  /** Extra context shown below the description. */
  subtitle?: string;
  /** Call-to-action button config. */
  action?: {
    label: string;
    onClick: () => void;
    /** @default 'primary' */
    type?: 'primary' | 'default' | 'dashed' | 'link' | 'text';
    icon?: ReactNode;
  };
  /** Visual variant. Use 'large' for landing-style pages. */
  size?: 'default' | 'large';
  /** Override Ant Design's default image. Set to false to hide entirely. */
  image?: 'default' | false;
  children?: ReactNode;
}

const sizeMap = {
  default: { paddingV: 48, fontSize: 14, gap: 8 },
  large: { paddingV: 80, fontSize: 16, gap: 12 },
} as const;

export default function EmptyState({
  description = '暂无数据',
  subtitle,
  action,
  size = 'default',
  image = 'default',
  children,
}: EmptyStateProps) {
  const s = sizeMap[size];

  return (
    <div
      style={{
        padding: `${s.paddingV}px 24px`,
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: 200,
      }}
    >
      <Empty
        image={image === false ? false : Empty.PRESENTED_IMAGE_SIMPLE}
        description={null}
      >
        <Space direction="vertical" align="center" style={{ gap: s.gap, textAlign: 'center' }}>
          <Text style={{ fontSize: s.fontSize, fontWeight: 500, color: 'var(--t2)' }}>
            {description}
          </Text>
          {subtitle && (
            <Text type="secondary" style={{ fontSize: 13, maxWidth: 360 }}>
              {subtitle}
            </Text>
          )}
          {action && (
            <Button type={action.type ?? 'primary'} icon={action.icon} onClick={action.onClick}>
              {action.label}
            </Button>
          )}
          {children}
        </Space>
      </Empty>
    </div>
  );
}
