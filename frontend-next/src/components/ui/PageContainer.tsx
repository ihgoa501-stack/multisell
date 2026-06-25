import { ReactNode } from 'react';
import { Spin, Typography } from 'antd';
import EmptyState from './EmptyState';

const { Text } = Typography;

export interface PageContainerProps {
  title: string;
  /** Optional subtitle shown below the title. */
  subtitle?: string;
  /** Action buttons rendered on the right side of the header. */
  extra?: ReactNode;
  /** Show a full-page loading spinner. */
  loading?: boolean;
  /** Loading description text. */
  loadingDesc?: string;
  /** Show empty state (takes priority over error and children). */
  empty?: boolean;
  /** Empty state description. */
  emptyDesc?: string;
  /** Empty state call-to-action. */
  emptyAction?: {
    label: string;
    onClick: () => void;
  };
  /** Show error state (takes priority over children, behind empty). */
  error?: boolean;
  /** Error message. */
  errorMsg?: string;
  /** Retry callback for error state. */
  onRetry?: () => void;
  children?: ReactNode;
}

export default function PageContainer({
  title,
  subtitle,
  extra,
  loading,
  loadingDesc,
  empty,
  emptyDesc,
  emptyAction,
  error,
  errorMsg,
  onRetry,
  children,
}: PageContainerProps) {
  // --- Loading state ---
  if (loading) {
    return (
      <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
        <Spin size="large" tip={loadingDesc ?? '加载中...'}>
          <div style={{ padding: 50 }} />
        </Spin>
      </div>
    );
  }

  // --- Empty state ---
  if (empty) {
    return (
      <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
        <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '1rem', color: 'var(--t1)' }}>{title}</h1>
        <div style={{ marginTop: 16 }} />
        <EmptyState
          description={emptyDesc ?? '暂无数据'}
          action={emptyAction}
        />
      </div>
    );
  }

  // --- Error state ---
  if (error) {
    return (
      <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
        <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '1rem', color: 'var(--t1)' }}>{title}</h1>
        <div style={{ marginTop: 16 }} />
        <EmptyState
          description={errorMsg ?? '加载失败'}
          subtitle="请检查网络连接后重试"
          action={onRetry ? { label: '重试', onClick: onRetry } : undefined}
        />
      </div>
    );
  }

  // --- Normal state ---
  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: subtitle ? 'flex-start' : 'center',
          marginBottom: 24,
          gap: 16,
        }}
      >
        <div style={{ flex: 1, minWidth: 0 }}>
          <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '1rem', color: 'var(--t1)', margin: 0 }}>{title}</h1>
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
