import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { Mock } from 'vitest';

// Mock auth module before any imports that use it
vi.mock('@/lib/auth', () => ({
  getToken: vi.fn(),
  setToken: vi.fn(),
  getRefreshToken: vi.fn(),
  setRefreshToken: vi.fn(),
  removeToken: vi.fn(),
  removeRefreshToken: vi.fn(),
}));

import { getToken, getRefreshToken, removeToken, removeRefreshToken } from '@/lib/auth';
import { ApiClient, ApiError } from '@/lib/api-client';

describe('ApiError', () => {
  it('classifies status codes into categories', () => {
    expect(ApiError.fromStatus(401).category).toBe('auth');
    expect(ApiError.fromStatus(403).category).toBe('auth');
    expect(ApiError.fromStatus(400).category).toBe('validation');
    expect(ApiError.fromStatus(422).category).toBe('validation');
    expect(ApiError.fromStatus(500).category).toBe('server');
    expect(ApiError.fromStatus(502).category).toBe('server');
    expect(ApiError.fromStatus(200).category).toBe('unknown');
  });

  it('includes status text when provided', () => {
    const err = ApiError.fromStatus(404, 'Not Found');
    expect(err.message).toBe('Not Found');
    expect(err.status).toBe(404);
  });
});

describe('ApiClient', () => {
  let client: ApiClient;
  let fetchMock: Mock;
  const BASE = 'http://test.api';

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = vi.fn();
    global.fetch = fetchMock;
    client = new ApiClient(BASE);
    Object.defineProperty(window, 'location', {
      value: { href: '' },
      writable: true,
    });
    // Default: no auth token
    (getToken as Mock).mockReturnValue(null);
  });

  afterEach(() => {
    ApiClient.setForbiddenHandler(null);
  });

  // --- GET request URL construction ---

  it('sends GET request with correct URL and query params', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: 0, data: { items: [] } }),
    });

    await client.get('/v1/products', { page: '1', sort: 'name' });

    const calledUrl = new URL(fetchMock.mock.calls[0][0] as string);
    expect(calledUrl.pathname).toBe('/v1/products');
    expect(calledUrl.searchParams.get('page')).toBe('1');
    expect(calledUrl.searchParams.get('sort')).toBe('name');
    expect(fetchMock).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ method: 'GET' }),
    );
  });

  // --- Authorization header ---

  it('includes Authorization header when token is available', async () => {
    (getToken as Mock).mockReturnValue('my-jwt');
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: 0, data: { id: 1 } }),
    });

    await client.get('/v1/me');

    expect(fetchMock).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer my-jwt',
        }),
      }),
    );
  });

  // --- POST request ---

  it('sends POST request with JSON body', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: 0, data: { id: 42 } }),
    });

    await client.post('/v1/products', { name: 'Widget', price: 9.99 });

    expect(fetchMock).toHaveBeenCalledWith(
      'http://test.api/v1/products',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
        body: JSON.stringify({ name: 'Widget', price: 9.99 }),
      }),
    );
  });

  it('sends explicit approval and idempotency headers for controlled writes', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: 0, data: { id: 42 } }),
    });

    await client.postApproved('/v1/listings/42/publish', {}, {
      approvalId: 7,
      idempotencyKey: 'publish-listing-42',
    });

    expect(fetchMock).toHaveBeenCalledWith(
      'http://test.api/v1/listings/42/publish',
      expect.objectContaining({
        headers: expect.objectContaining({
          'X-Approval-ID': '7',
          'Idempotency-Key': 'publish-listing-42',
        }),
      }),
    );
  });

  it('rejects invalid approval execution context before network access', async () => {
    await expect(client.postApproved('/v1/listings/42/publish', {}, {
      approvalId: 0,
      idempotencyKey: 'short',
    })).rejects.toBeInstanceOf(ApiError);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('surfaces the backend business error message', async () => {
    const body = { code: 502, message: 'TAB_NOT_FOUND: 请先打开完全相同的 1688 商品页面' };
    fetchMock.mockResolvedValue({
      ok: false,
      status: 502,
      statusText: 'Bad Gateway',
      clone: () => ({ json: () => Promise.resolve(body) }),
    });

    await expect(client.post('/v1/sourcing-1688/fetch', {})).rejects.toThrow(body.message);
  });

  // --- GET request deduplication ---

  it('deduplicates concurrent identical GET requests', async () => {
    let resolveFirst: (v: unknown) => void;
    const firstPromise = new Promise((resolve) => {
      resolveFirst = resolve;
    });
    fetchMock.mockReturnValue(firstPromise);

    const result1 = client.get('/v1/products');
    const result2 = client.get('/v1/products');

    resolveFirst!({
      ok: true,
      json: () => Promise.resolve({ code: 0, data: null }),
    });

    await Promise.all([result1, result2]);
    // fetch should be called only once for the deduped requests
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  // --- GET dedup cleanup ---

  it('clears inflightRequests Map after GET completes', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: 0, data: null }),
    });

    await client.get('/v1/products');
    await client.get('/v1/products');

    // Second call is NOT deduped because first call's inflight entry was cleaned
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  // --- 401 auto-refresh and retry ---

  it('refreshes token on 401 and retries the request', async () => {
    let callCount = 0;
    fetchMock.mockImplementation(async (url: string, options: any) => {
      callCount++;
      const s = url.toString();
      // Refresh endpoint
      if (s.includes('/v1/auth/refresh')) {
        return {
          ok: true,
          json: () =>
            Promise.resolve({
              code: 0,
              data: { access_token: 'new-token', refresh_token: 'new-refresh' },
            }),
        };
      }
      // First call to original endpoint returns 401
      if (callCount === 1) {
        return { ok: false, status: 401, statusText: 'Unauthorized' };
      }
      // Retry with new token succeeds
      return {
        ok: true,
        json: () => Promise.resolve({ code: 0, data: { id: 1 } }),
      };
    });

    (getToken as Mock).mockReturnValue('old-token');
    (getRefreshToken as Mock).mockReturnValue('old-refresh');

    const result = await client.post('/v1/products', { name: 'X' });

    // fetch called: original (401) + refresh + retry = 3
    expect(callCount).toBe(3);
    expect(result).toEqual({ code: 0, data: { id: 1 } });
  });

  // --- Token refresh failure --> logout ---

  it('clears tokens and redirects to /login when refresh fails', async () => {
    fetchMock.mockImplementation(async (url: string) => {
      if (url.toString().includes('/v1/auth/refresh')) {
        return { ok: false, status: 401, statusText: 'Unauthorized' };
      }
      return { ok: false, status: 401, statusText: 'Unauthorized' };
    });

    (getToken as Mock).mockReturnValue('old-token');
    (getRefreshToken as Mock).mockReturnValue('old-refresh');

    await expect(client.post('/v1/products')).rejects.toThrow(
      'Session expired',
    );

    expect(removeToken).toHaveBeenCalled();
    expect(removeRefreshToken).toHaveBeenCalled();
    expect(window.location.href).toBe('/login');
  });

  // --- Refresh queue ---

  it('makes only one refresh call for concurrent 401s', async () => {
    let refreshResolve!: () => void;
    const refreshBlock = new Promise<void>((r) => {
      refreshResolve = r;
    });
    let refreshCallCount = 0;

    fetchMock.mockImplementation(async (url: string, options: any) => {
      const s = url.toString();
      if (s.includes('/v1/auth/refresh')) {
        refreshCallCount++;
        await refreshBlock;
        return {
          ok: true,
          json: () =>
            Promise.resolve({
              code: 0,
              data: { access_token: 'new-token', refresh_token: 'new-refresh' },
            }),
        };
      }

      // Initial request: 401; retry with new token succeeds
      if (
        options?.headers &&
        (options.headers as Record<string, string>).Authorization ===
          'Bearer new-token'
      ) {
        return {
          ok: true,
          json: () => Promise.resolve({ code: 0, data: null }),
        };
      }
      return { ok: false, status: 401, statusText: 'Unauthorized' };
    });

    (getToken as Mock).mockReturnValue('old-token');
    (getRefreshToken as Mock).mockReturnValue('old-refresh');

    const allDone = Promise.allSettled([
      client.post('/v1/orders'),
      client.post('/v1/products'),
    ]);

    // Wait for both requests to receive their 401 and enter the refresh flow
    await vi.waitFor(() => {
      expect(refreshCallCount).toBe(1);
    });

    // Unblock the refresh
    refreshResolve!();
    await allDone;
  });

  // --- 403 triggers forbiddenHandler ---

  it('calls forbiddenHandler on 403 response', async () => {
    const handler = vi.fn();
    ApiClient.setForbiddenHandler(handler);

    fetchMock.mockResolvedValue({
      ok: false,
      status: 403,
      statusText: 'Forbidden',
      json: () => Promise.resolve({ code: 403, message: 'Forbidden' }),
    });

    await expect(client.get('/v1/admin')).rejects.toThrow();

    expect(handler).toHaveBeenCalledTimes(1);
  });
});
