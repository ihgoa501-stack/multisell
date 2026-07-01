'use client';

/**
 * Skeleton loading placeholders that match page layout structure.
 * Replaces bare <Spin> with content-aware loading shapes.
 */

export function CardSkeleton({ rows = 1 }: { rows?: number }) {
  return (
    <div
      style={{
        background: 'var(--s1)',
        border: '1px solid var(--bd)',
        borderRadius: 8,
        padding: 'var(--space-lg)',
      }}
      aria-busy="true"
      aria-label="Loading"
    >
      <div
        style={{
          height: 14,
          width: '40%',
          background: 'var(--s3)',
          borderRadius: 4,
          marginBottom: 'var(--space-sm)',
        }}
      />
      <div
        style={{
          height: 28,
          width: '60%',
          background: 'var(--s2)',
          borderRadius: 4,
        }}
      />
      {Array.from({ length: rows }).map((_, i) => (
        <div
          key={i}
          style={{
            height: 14,
            width: '80%',
            background: 'var(--s3)',
            borderRadius: 4,
            marginTop: 'var(--space-sm)',
          }}
        />
      ))}
    </div>
  );
}

export function TableSkeleton({ rows = 5, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <div aria-busy="true" aria-label="Loading table">
      {/* Header */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: `repeat(${cols}, 1fr)`,
          gap: 'var(--space-md)',
          padding: 'var(--space-md) var(--space-lg)',
          borderBottom: '1px solid var(--bd)',
        }}
      >
        {Array.from({ length: cols }).map((_, i) => (
          <div key={i} style={{ height: 12, background: 'var(--s3)', borderRadius: 4, width: '60%' }} />
        ))}
      </div>
      {/* Rows */}
      {Array.from({ length: rows }).map((_, r) => (
        <div
          key={r}
          style={{
            display: 'grid',
            gridTemplateColumns: `repeat(${cols}, 1fr)`,
            gap: 'var(--space-md)',
            padding: 'var(--space-md) var(--space-lg)',
            borderBottom: r < rows - 1 ? '1px solid var(--bd)' : 'none',
          }}
        >
          {Array.from({ length: cols }).map((_, c) => (
            <div
              key={c}
              style={{
                height: 12,
                background: 'var(--s2)',
                borderRadius: 4,
                width: `${50 + Math.random() * 40}%`,
              }}
            />
          ))}
        </div>
      ))}
    </div>
  );
}

export function StatRowSkeleton({ count = 6 }: { count?: number }) {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: `repeat(${Math.min(count, 6)}, 1fr)`,
        gap: 'var(--space-md)',
      }}
      aria-busy="true"
      aria-label="Loading stats"
    >
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={i}
          style={{
            background: 'var(--s1)',
            border: '1px solid var(--bd)',
            borderRadius: 8,
            padding: 'var(--space-lg)',
          }}
        >
          <div style={{ height: 12, width: '50%', background: 'var(--s3)', borderRadius: 4, marginBottom: 'var(--space-sm)' }} />
          <div style={{ height: 24, width: '70%', background: 'var(--s2)', borderRadius: 4 }} />
        </div>
      ))}
    </div>
  );
}
