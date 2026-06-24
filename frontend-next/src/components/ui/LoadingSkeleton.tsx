import { Skeleton, Card, Table } from 'antd';

export interface SkeletonTableProps {
  /** Number of rows to render. Default 5. */
  rows?: number;
  /** Number of columns. Default 4. */
  columns?: number;
  /** Whether to show the table header. Default true. */
  showHeader?: boolean;
}

export function SkeletonTable({ rows = 5, columns = 4, showHeader = true }: SkeletonTableProps) {
  const cols = Array.from({ length: columns }, (_, i) => ({
    key: String(i),
    title: showHeader ? <Skeleton.Input active size="small" style={{ width: 80 }} /> : undefined,
    dataIndex: String(i),
    render: () => (
      <Skeleton.Input active size="small" style={{ width: `${60 + (i * 7) % 30}%` }} />
    ),
  }));

  const data = Array.from({ length: rows }, (_, i) => ({
    key: String(i),
    ...Object.fromEntries(cols.map((_, j) => [String(j), null])),
  }));

  return (
    <Table
      columns={cols as never}
      dataSource={data}
      pagination={false}
      size="small"
      style={{ pointerEvents: 'none' }}
    />
  );
}

export interface SkeletonCardProps {
  /** Number of cards to render. Default 3. */
  count?: number;
}

export function SkeletonCard({ count = 3 }: SkeletonCardProps) {
  return (
    <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
      {Array.from({ length: count }, (_, i) => (
        <Card key={i} style={{ flex: '1 1 240px', minWidth: 200 }}>
          {i === 0 && (
            <Skeleton.Input active size="small" style={{ width: 100, marginBottom: 12 }} />
          )}
          <Skeleton active paragraph={{ rows: 2 }} />
        </Card>
      ))}
    </div>
  );
}

export interface SkeletonListProps {
  /** Number of items to render. Default 5. */
  count?: number;
}

export function SkeletonList({ count = 5 }: SkeletonListProps) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {Array.from({ length: count }, (_, i) => (
        <Card key={i} size="small">
          <Skeleton active avatar paragraph={{ rows: 1 }} />
        </Card>
      ))}
    </div>
  );
}
