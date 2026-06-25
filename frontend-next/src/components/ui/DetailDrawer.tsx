import { ReactNode } from 'react';
import { Drawer, Typography, Space, Spin, Empty } from 'antd';

const { Text } = Typography;

export interface DetailField {
  label: string;
  value: ReactNode;
  /** Span across full width. Default false. */
  fullWidth?: boolean;
  /** Hide if value is null/undefined. */
  hidden?: boolean;
}

export interface DetailDrawerProps {
  /** Whether the drawer is visible. */
  open: boolean;
  /** Drawer title. */
  title: string;
  /** Fields to display as a key-value list. */
  fields?: DetailField[];
  /** Custom content (overrides fields when provided). */
  children?: ReactNode;
  /** Callback to close the drawer. */
  onClose: () => void;
  /** Drawer width. @default 480 */
  width?: number | string;
  /** Show loading spinner. */
  loading?: boolean;
  /** Show empty state when fields are empty. */
  empty?: boolean;
  /** Extra content in the footer. */
  footer?: ReactNode;
}

export default function DetailDrawer({
  open,
  title,
  fields,
  children,
  onClose,
  width = 480,
  loading,
  empty,
  footer,
}: DetailDrawerProps) {
  return (
    <Drawer
      title={title}
      placement="right"
      open={open}
      onClose={onClose}
      width={width}
      styles={{
        body: { padding: 24, display: 'flex', flexDirection: 'column' },
      }}
    >
      {loading ? (
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Spin size="large" />
        </div>
      ) : empty ? (
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Empty description="暂无详情" />
        </div>
      ) : children ? (
        <div style={{ flex: 1 }}>{children}</div>
      ) : (
        <div style={{ flex: 1 }}>
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            {(fields ?? []).map(
              (field) =>
                !field.hidden && (
                  <div
                    key={field.label}
                    style={{
                      display: 'flex',
                      flexDirection: field.fullWidth ? 'column' : 'row',
                      gap: field.fullWidth ? 4 : 8,
                    }}
                  >
                    <Text
                      type="secondary"
                      style={{
                        minWidth: field.fullWidth ? undefined : 100,
                        fontSize: 13,
                        flexShrink: 0,
                      }}
                    >
                      {field.label}
                    </Text>
                    <div style={{ flex: 1, wordBreak: 'break-word' }}>
                      {field.value ?? '-'}
                    </div>
                  </div>
                )
            )}
          </Space>
        </div>
      )}

      {footer && (
        <div
          style={{
            borderTop: '1px solid var(--bd)',
            paddingTop: 16,
            marginTop: 16,
          }}
        >
          {footer}
        </div>
      )}
    </Drawer>
  );
}
