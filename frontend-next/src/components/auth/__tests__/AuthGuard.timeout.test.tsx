import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import AuthGuard from '@/components/auth/AuthGuard';

const fetchPermissions = vi.fn();
const clearPermissions = vi.fn();

vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace: vi.fn() }),
}));

vi.mock('@/stores/permission-store', () => ({
  usePermissionStore: (selector: (state: unknown) => unknown) => selector({
    fetchPermissions,
    clearPermissions,
    fetched: false,
  }),
}));

describe('AuthGuard permission timeout', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.setItem('token', 'test-token');
  });

  afterEach(() => {
    vi.useRealTimers();
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('offers recovery instead of showing an indefinite spinner', async () => {
    render(<AuthGuard><div>受保护内容</div></AuthGuard>);

    await act(async () => {
      vi.advanceTimersByTime(6000);
    });

    expect(screen.getByText('权限信息加载超时')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '重新加载权限' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '返回登录' })).toBeInTheDocument();
  });
});
