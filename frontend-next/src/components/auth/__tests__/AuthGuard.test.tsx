import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import React from 'react';

// Mock next/navigation
const mockReplace = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace: mockReplace, push: mockReplace }),
}));

// Mock permission store
const mockFetchPermissions = vi.fn();

type PermissionStoreState = {
  fetched: boolean;
  fetchPermissions: typeof mockFetchPermissions;
  permissions: string[];
};

describe('AuthGuard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('redirects to login when no token', async () => {
    vi.doMock('@/stores/permission-store', () => ({
      usePermissionStore: <T,>(selector: (state: PermissionStoreState) => T) => {
        const state = { fetched: false, fetchPermissions: mockFetchPermissions, permissions: [] };
        return selector(state);
      },
    }));

    const AuthGuard = (await import('@/components/auth/AuthGuard')).default;
    render(React.createElement(AuthGuard, null, 'Content'));
    expect(mockReplace).toHaveBeenCalledWith('/login');
  });

  it('fetches permissions when token exists', async () => {
    localStorage.setItem('token', 'test-token');

    vi.doMock('@/stores/permission-store', () => ({
      usePermissionStore: <T,>(selector: (state: PermissionStoreState) => T) => {
        const state = { fetched: true, fetchPermissions: mockFetchPermissions, permissions: ['admin'] };
        return selector(state);
      },
    }));

    const AuthGuard = (await import('@/components/auth/AuthGuard')).default;
    render(React.createElement(AuthGuard, null, 'Protected Content'));

    expect(mockFetchPermissions).toHaveBeenCalled();
  });
});
