import { ReactNode } from 'react';
import { Typography } from 'antd';
import Breadcrumbs from '@/components/layout/Breadcrumbs';

const { Text } = Typography;

export interface PageHeaderProps {
  title: string;
  /** Optional subtitle below the title. */
  subtitle?: string;
  /** Action buttons rendered on the right. */
  extra?: ReactNode;
  /** Whether to show breadcrumbs above the title. Default true. */
  showBreadcrumb?: boolean;
  children?: ReactNode;
}

export default function PageHeader({
  title,
  subtitle,
  extra,
  showBreadcrumb = true,
  children,
}: PageHeaderProps) {
  return (
    <div style={{ marginBottom: 24 }}>
      {showBreadcrumb && (
        <div style={{ marginBottom: 8 }}>
          <Breadcrumbs />
        </div>
      )}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: subtitle ? 'flex-start' : 'center',
          gap: 16,
        }}
      >
        <div style={{ flex: 1, minWidth: 0 }}>
          <h1 style={{ fontSize: 24, fontWeight: 600, margin: 0 }}>{title}</h1>
          {subtitle && (
            <Text type="secondary" style={{ display: 'block', marginTop: 4 }}>
              {subtitle}
            </Text>
          )}
        </div>
        {extra && <div style={{ flexShrink: 0 }}>{extra}</div>}
      </div>
      {children}
    </div>
  );
}
