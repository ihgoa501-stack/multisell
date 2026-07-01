import type { ReactNode } from 'react';

interface SectionCardProps {
  title: ReactNode;
  children: ReactNode;
  actions?: ReactNode;
  /** @default false */
  noPadding?: boolean;
  style?: React.CSSProperties;
}

/**
 * Card with styled header + body, using design-token CSS variables.
 * Replaces repeating div.card + div.header + div.body patterns.
 */
export default function SectionCard({
  title,
  children,
  actions,
  noPadding = false,
  style,
}: SectionCardProps) {
  return (
    <div
      className="section-card"
      style={{
        background: 'var(--s1)',
        border: '1px solid var(--bd)',
        borderRadius: 8,
        transition: 'background var(--dur-micro) ease, border-color var(--dur-short) ease',
        ...style,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: 'var(--space-md) var(--space-lg)',
          borderBottom: '1px solid var(--bd)',
          fontFamily: 'var(--ds)',
          fontWeight: 600,
          fontSize: '0.875rem',
          color: 'var(--t1)',
          transition: 'border-color var(--dur-short) ease',
        }}
      >
        <span>{title}</span>
        {actions && <div>{actions}</div>}
      </div>
      <div style={{ padding: noPadding ? 0 : 'var(--space-lg)' }}>
        {children}
      </div>
    </div>
  );
}
