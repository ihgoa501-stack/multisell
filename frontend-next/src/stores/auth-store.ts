import { create } from 'zustand';
import type { User } from '@/types/api';
import { getToken, setToken, removeToken, getStoredUser, setStoredUser, removeStoredUser } from '@/lib/auth';
import apiClient from '@/lib/api-client';

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  hydrate: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  isAuthenticated: false,

  login: async (email: string, password: string) => {
    const result = await apiClient.post<{ token: string; user: User }>('/auth/login', {
      username: email,
      password,
    });
    if (result.data) {
      const { token, user } = result.data;
      setToken(token);
      setStoredUser(user);
      set({ user, token, isAuthenticated: true });
    }
  },

  logout: () => {
    removeToken();
    removeStoredUser();
    set({ user: null, token: null, isAuthenticated: false });
  },

  hydrate: () => {
    const token = getToken();
    const user = getStoredUser();
    if (token && user) {
      set({ user, token, isAuthenticated: true });
    }
  },
}));
