import { create } from 'zustand';
import apiClient from '@/lib/api-client';

interface PermissionState {
  permissions: string[];
  loading: boolean;
  fetched: boolean;
  fetchPermissions: () => Promise<void>;
  hasPermission: (code: string) => boolean;
  hasAnyPermission: (codes: string[]) => boolean;
  clearPermissions: () => void;
}

export const usePermissionStore = create<PermissionState>((set, get) => ({
  permissions: [],
  loading: false,
  fetched: false,

  fetchPermissions: async () => {
    // Skip if already fetched or currently loading
    if (get().fetched || get().loading) return;

    set({ loading: true });

    try {
      const res = await apiClient.get<{ permissions: string[] }>('/v1/rbac/current/permissions');
      if (res.data?.permissions) {
        set({ permissions: res.data.permissions, fetched: true, loading: false });
      } else {
        set({ permissions: [], fetched: true, loading: false });
      }
    } catch {
      // On error, keep empty permissions
      set({ permissions: [], fetched: true, loading: false });
    }
  },

  hasPermission: (code: string) => {
    return get().permissions.includes(code);
  },

  hasAnyPermission: (codes: string[]) => {
    if (codes.length === 0) return true;
    const perms = get().permissions;
    return codes.some((code) => perms.includes(code));
  },

  clearPermissions: () => {
    set({ permissions: [], loading: false, fetched: false });
  },
}));
