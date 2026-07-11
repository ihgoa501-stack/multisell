import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import CrudListPage from '@/components/crud/CrudListPage';

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
  usePathname: () => '/test',
}));

vi.mock('@tanstack/react-query', () => ({
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
  useQuery: () => ({
    data: undefined,
    isLoading: true,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock('@/lib/api-client', () => ({
  default: { getPage: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

describe('CrudListPage loading timeout', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it('turns a stalled loader into a recoverable warning', async () => {
    render(
      <CrudListPage
        resource="/test"
        title="测试管理"
        singular="测试"
        columns={[{ title: '名称', dataIndex: 'name' }]}
        fields={[{ name: 'name', label: '名称' }]}
      />,
    );

    await act(async () => {
      vi.advanceTimersByTime(8000);
    });

    expect(screen.getByText('测试管理加载时间过长')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /重新加载/ })).toBeInTheDocument();
  });
});
