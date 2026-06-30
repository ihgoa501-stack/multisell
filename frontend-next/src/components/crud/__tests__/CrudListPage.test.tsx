import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import CrudListPage, { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

// Mock next/navigation
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
  usePathname: () => '/test',
}));

// Mock @tanstack/react-query
vi.mock('@tanstack/react-query', () => ({
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
  useQuery: () => ({
    data: { data: [], total: 0, page: 1, size: 10 },
    isLoading: false,
    refetch: vi.fn(),
  }),
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}));

// Mock api-client
vi.mock('@/lib/api-client', () => ({
  default: {
    getPage: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('CrudListPage', () => {
  const baseProps = {
    resource: '/test',
    title: '测试管理',
    singular: '测试',
    columns: [
      { title: 'ID', dataIndex: 'id', width: 70 },
      { title: '名称', dataIndex: 'name' },
    ],
    fields: [
      { name: 'name', label: '名称', required: true },
    ],
  };

  it('renders title', () => {
    render(<CrudListPage {...baseProps} />);
    expect(screen.getByText('测试管理')).toBeInTheDocument();
  });

  it('renders create button', () => {
    render(<CrudListPage {...baseProps} />);
    expect(screen.getByText((content) => content.replace(/\s/g, '').includes('新建'))).toBeInTheDocument();
  });

  it('renders refresh button', () => {
    render(<CrudListPage {...baseProps} />);
    expect(screen.getByText((content) => content.replace(/\s/g, '').includes('刷新'))).toBeInTheDocument();
  });

  it('hides create button when editable=false', () => {
    render(<CrudListPage {...baseProps} editable={false} />);
    expect(screen.queryByText((content) => content.replace(/\s/g, '').includes('新建'))).not.toBeInTheDocument();
  });

  it('renders search input', () => {
    render(<CrudListPage {...baseProps} />);
    expect(screen.getByPlaceholderText('搜索...')).toBeInTheDocument();
  });

  it('renders export button when showExport=true', () => {
    render(<CrudListPage {...baseProps} showExport />);
    expect(screen.getByText((content) => content.replace(/\s/g, '').includes('导出'))).toBeInTheDocument();
  });

  it('renders filters when provided', () => {
    const filters = [
      {
        key: 'status',
        label: '状态',
        options: [{ label: '启用', value: 'active' }],
      },
    ];
    render(<CrudListPage {...baseProps} filters={filters} />);
    expect(screen.getByText((content) => content.replace(/\s/g, '').includes('状态'))).toBeInTheDocument();
  });
});

describe('fmtDate', () => {
  it('formats ISO date', () => {
    const result = fmtDate('2024-01-15T10:30:00Z');
    expect(typeof result).toBe('string');
    expect(result).toContain('2024-01-15');
  });

  it('returns dash for null', () => {
    expect(fmtDate(null)).toBe('-');
  });

  it('returns dash for undefined', () => {
    expect(fmtDate(undefined)).toBe('-');
  });

  it('returns raw string for non-date', () => {
    expect(fmtDate('not-a-date')).toBe('not-a-date');
  });
});

describe('fmtMoney', () => {
  it('formats number with ¥', () => {
    expect(fmtMoney(100)).toBe('¥100.00');
  });

  it('returns dash for null', () => {
    expect(fmtMoney(null)).toBe('-');
  });

  it('returns dash for undefined', () => {
    expect(fmtMoney(undefined)).toBe('-');
  });

  it('handles zero', () => {
    expect(fmtMoney(0)).toBe('¥0.00');
  });
});
