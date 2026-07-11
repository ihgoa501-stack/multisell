import { describe, it, expect, vi } from 'vitest';
import { renderToString } from 'react-dom/server';
import AuthGuard from '@/components/auth/AuthGuard';

vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace: vi.fn() }),
}));

vi.mock('@/stores/permission-store', () => ({
  usePermissionStore: (selector: (state: unknown) => unknown) => selector({
    fetchPermissions: vi.fn(),
    clearPermissions: vi.fn(),
    fetched: false,
  }),
}));

describe('AuthGuard hydration', () => {
  it('renders the same empty boundary during SSR and initial hydration', () => {
    const html = renderToString(<AuthGuard><div>受保护内容</div></AuthGuard>);
    expect(html).toBe('');
  });
});
