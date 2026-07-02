import { ReactNode } from 'react';
import { Card, Statistic, Typography } from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined, MinusOutlined } from '@ant-design/icons';

const { Text } = Typography;

export interface StatCardProps {
  title: string;
  value: number | string | undefined | null;
  /** Prefix before the value (can be icon or currency symbol). */
  prefix?: ReactNode;
  /** Suffix after the value (e.g. "%"). */
  suffix?: string;
  /** Precision for decimal values. Default 0. */
  precision?: number;
  /** Inline style for the value text. */
  valueStyle?: React.CSSProperties;
  /** Trend indicator. */
  trend?: {
    value: number;
    /** @default 'up' — means increase is good */
    direction?: 'up' | 'down';
    label?: string;
  };
  /** Icon shown on the left. */
  icon?: ReactNode;
  /** Background color tint for the icon area. */
  iconBgColor?: string;
  /** Whether the value is loading. */
  loading?: boolean;
  /** Click handler. */
  onClick?: () => void;
  /** Additional styles. */
  style?: React.CSSProperties;
}

export default function StatCard({
  title,
  value,
  prefix,
  suffix,
  precision,
  valueStyle,
  trend,
  icon,
  iconBgColor = '#f0f5ff',
  loading,
  onClick,
  style,
}: StatCardProps) {
  const isGoodTrend = trend?.value !== undefined
      ? (trend.value > 0 && trend.direction !== 'down') ||
        (trend.value < 0 && trend.direction === 'down')
      : true;

  const trendColor =
    !trend || trend.value === 0
      ? 'var(--t4)'
      : isGoodTrend
        ? 'var(--g4)'
        : 'var(--r4)';

  const TrendIcon =
    trend && trend.value > 0
      ? ArrowUpOutlined
      : trend && trend.value < 0
        ? ArrowDownOutlined
        : trend
          ? MinusOutlined
          : null;

  return (
    <Card
      hoverable={!!onClick}
      onClick={onClick}
      style={style}
      styles={{ body: { padding: 20 } }}
    >
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 16 }}>
        {icon && (
          <div
            style={{
              width: 48,
              height: 48,
              borderRadius: 12,
              backgroundColor: iconBgColor === '#f0f5ff' ? 'var(--s2)' : iconBgColor,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
              fontSize: 24,
            }}
          >
            {icon}
          </div>
        )}
        <div style={{ flex: 1, minWidth: 0 }}>
          <Statistic
            title={title}
            value={value ?? '-'}
            precision={precision}
            prefix={prefix}
            suffix={suffix}
            loading={loading}
            valueStyle={valueStyle}
          />
          {trend && TrendIcon && (
            <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 4 }}>
              <Text
                style={{
                  color: trendColor,
                  fontSize: 13,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 2,
                }}
              >
                <TrendIcon style={{ fontSize: 12 }} />
                {Math.abs(trend.value).toFixed(1)}%
              </Text>
              {trend.label && (
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {trend.label}
                </Text>
              )}
            </div>
          )}
        </div>
      </div>
    </Card>
  );
}
