import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  getToken,
  setToken,
  removeToken,
  getRefreshToken,
  setRefreshToken,
  removeRefreshToken,
  getStoredUser,
  setStoredUser,
  removeStoredUser,
  isAuthenticated,
} from '@/lib/auth';

let store: Record<string, string>;

beforeEach(() => {
  store = {};
  vi.stubGlobal('window', {
    localStorage: {
      getItem: vi.fn((key: string) => store[key] ?? null),
      setItem: vi.fn((key: string, value: string) => {
        store[key] = value;
      }),
      removeItem: vi.fn((key: string) => {
        delete store[key];
      }),
      clear: vi.fn(() => {
        store = {};
      }),
      get length() {
        return Object.keys(store).length;
      },
      key: vi.fn((i: number) => Object.keys(store)[i] ?? null),
    },
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('auth token lifecycle', () => {
  it('returns null when no token is stored', () => {
    expect(getToken()).toBeNull();
  });

  it('stores and retrieves a token', () => {
    setToken('jwt-abc');
    expect(getToken()).toBe('jwt-abc');
  });

  it('removes a token', () => {
    setToken('jwt-abc');
    removeToken();
    expect(getToken()).toBeNull();
  });
});

describe('auth refresh token lifecycle', () => {
  it('returns null when no refresh token is stored', () => {
    expect(getRefreshToken()).toBeNull();
  });

  it('stores and retrieves a refresh token', () => {
    setRefreshToken('refresh-123');
    expect(getRefreshToken()).toBe('refresh-123');
  });

  it('removes a refresh token', () => {
    setRefreshToken('refresh-123');
    removeRefreshToken();
    expect(getRefreshToken()).toBeNull();
  });
});

describe('auth stored user', () => {
  it('returns null when no user is stored', () => {
    expect(getStoredUser()).toBeNull();
  });

  it('stores and retrieves a user as JSON', () => {
    const user = { id: '1', email: 'a@b.com', name: 'Test' };
    setStoredUser(user);
    expect(getStoredUser()).toEqual(user);
  });

  it('returns null for invalid JSON', () => {
    localStorage.setItem('user', '{bad json');
    expect(getStoredUser()).toBeNull();
  });

  it('removes a stored user', () => {
    setStoredUser({ id: '1', email: 'a@b.com', name: 'Test' });
    removeStoredUser();
    expect(getStoredUser()).toBeNull();
  });
});

describe('isAuthenticated', () => {
  it('returns false when no token is set', () => {
    expect(isAuthenticated()).toBe(false);
  });

  it('returns true when a token is set', () => {
    setToken('jwt-xyz');
    expect(isAuthenticated()).toBe(true);
  });
});
