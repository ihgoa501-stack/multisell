import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { apiClient } from '@/lib/api-client';

const mockGetToken = vi.fn<string | null>();
const mockSetToken = vi.fn<void>();
const mockGetRefreshToken = vi.fn<string | null>();
const mockSetRefreshToken = vi.fn<void>();
const mockRemoveToken = vi.fn<void>();
const mockRemoveRefreshToken = vi.fn<void>();

vi.mock('@/lib/auth', () => ({
  getToken: () => mockGetToken(),
  setToken: (t: string) => mockSetToken(t),
  getRefreshToken: () => mockGetRefreshToken(),
  setRefreshToken: (t: string) => mockSetRefreshToken(t),
  removeToken: () => mockRemoveToken(),
  removeRefreshToken: () => mockRemoveRefreshToken(),
}));

let fetchSpy: ReturnType<typeof vi.fn>;

beforeEach(() => {
  vi.resetModules();
  fetchSpy = vi.fn();
  vi.stubGlobal('fetch', fetchSpy);
  mockGetToken.mockReturnValue(null);
  mockGetRefreshToken.mockReturnValue(null);
  Object.defineProperty(window, 'location', {
    value: { href: '' },
    writable: true,
    configurable: true,
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

function jsonOk<T>(data: T, status = 200): Response {
  return {
    ok: true,
    status,
    statusText: 'OK',
    json: () => Promise.resolve(data),
  } as unknown as Response;
}

function jsonError(status: number, statusText = 'Error'): Response {
  return {
    ok: false,
    status,
    statusText,
    json: () => Promise.resolve({ code: status, message: statusText }),
  } as unknown as Response;
}

describe('apiClient', () => {
  it('sends a successful GET request', async () => {
    const body = { code: 0, message: 'ok', data: { id: 1 } };
    fetchSpy.mockResolvedValue(jsonOk(body));

    const result = await apiClient.get('/v1/items/1');

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    expect(fetchSpy.mock.calls[0][0]).toBe('http://localhost:8080/api/v1/items/1');
    expect(fetchSpy.mock.calls[0][1].method).toBe('GET');
    expect(result).toEqual(body);
  });

  it('sends a successful POST with body', async () => {
    const sent = { code: 0, message: 'created', data: { id: 10 } };
    fetchSpy.mockResolvedValue(jsonOk(sent, 201));

    const result = await apiClient.post('/v1/items', { name: 'Widget' });

    const [, opts] = fetchSpy.mock.calls[0];
    expect(opts.method).toBe('POST');
    expect(opts.body).toBe(JSON.stringify({ name: 'Widget' }));
    expect(result).toEqual(sent);
  });

  it('appends query params to URL', async () => {
    fetchSpy.mockResolvedValue(jsonOk({ code: 0, message: 'ok', data: [] }));

    await apiClient.get('/v1/items', { page: '2', size: '10', q: 'hello world' });

    const url = new URL(fetchSpy.mock.calls[0][0]);
    expect(url.searchParams.get('page')).toBe('2');
    expect(url.searchParams.get('size')).toBe('10');
    expect(url.searchParams.get('q')).toBe('hello world');
  });

  it('throws on non-200 status', async () => {
    fetchSpy.mockResolvedValue(jsonError(500, 'Internal Server Error'));

    await expect(apiClient.get('/v1/fail')).rejects.toThrow('API error: 500');
  });

  it('includes Authorization header when token exists', async () => {
    mockGetToken.mockReturnValue('abc-123');
    fetchSpy.mockResolvedValue(jsonOk({ code: 0, message: 'ok' }));

    await apiClient.get('/v1/me');

    const headers = fetchSpy.mock.calls[0][1].headers;
    expect(headers['Authorization']).toBe('Bearer abc-123');
  });

  it('does not include Authorization header when no token', async () => {
    mockGetToken.mockReturnValue(null);
    fetchSpy.mockResolvedValue(jsonOk({ code: 0, message: 'ok' }));

    await apiClient.get('/v1/me');

    const headers = fetchSpy.mock.calls[0][1].headers;
    expect(headers['Authorization']).toBeUndefined();
  });

  it('triggers token refresh on 401 and retries', async () => {
    mockGetToken.mockReturnValue('expired');
    mockGetRefreshToken.mockReturnValue('refresh-xyz');

    const firstCall = jsonError(401, 'Unauthorized');
    const refreshResp = jsonOk({
      code: 0,
      message: 'ok',
      data: { access_token: 'new-acc', refresh_token: 'new-ref' },
    });
    const retryResp = jsonOk({ code: 0, message: 'ok', data: 'retried' });

    fetchSpy
      .mockResolvedValueOnce(firstCall)
      .mockResolvedValueOnce(refreshResp)
      .mockResolvedValueOnce(retryResp);

    const result = await apiClient.get('/v1/protected');

    expect(fetchSpy).toHaveBeenCalledTimes(3);
    expect(fetchSpy.mock.calls[1][0]).toContain('/v1/auth/refresh');
    expect(fetchSpy.mock.calls[1][1].method).toBe('POST');
    expect(mockSetToken).toHaveBeenCalledWith('new-acc');
    expect(mockSetRefreshToken).toHaveBeenCalledWith('new-ref');
    expect(result).toEqual({ code: 0, message: 'ok', data: 'retried' });
  });

  it('redirects to /login when refresh fails', async () => {
    mockGetToken.mockReturnValue('expired');
    mockGetRefreshToken.mockReturnValue('bad-refresh');

    fetchSpy
      .mockResolvedValueOnce(jsonError(401))
      .mockResolvedValueOnce(jsonError(500));

    await expect(apiClient.get('/v1/secret')).rejects.toThrow('Session expired');
    expect(window.location.href).toBe('/login');
    expect(mockRemoveToken).toHaveBeenCalled();
    expect(mockRemoveRefreshToken).toHaveBeenCalled();
  });

  it('deduplicates concurrent GET requests for the same URL', async () => {
    fetchSpy.mockResolvedValue(jsonOk({ code: 0, message: 'ok', data: [1, 2] }));

    const [a, b] = await Promise.all([
      apiClient.get('/v1/items', { page: '1' }),
      apiClient.get('/v1/items', { page: '1' }),
    ]);

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    expect(a).toEqual(b);
  });

  it('does not deduplicate GET requests with different params', async () => {
    fetchSpy.mockResolvedValue(jsonOk({ code: 0, message: 'ok' }));

    await Promise.all([
      apiClient.get('/v1/items', { page: '1' }),
      apiClient.get('/v1/items', { page: '2' }),
    ]);

    expect(fetchSpy).toHaveBeenCalledTimes(2);
  });

  it('sends PUT request with body', async () => {
    fetchSpy.mockResolvedValue(jsonOk({ code: 0, message: 'ok' }));

    await apiClient.put('/v1/items/1', { name: 'Updated' });

    const [, opts] = fetchSpy.mock.calls[0];
    expect(opts.method).toBe('PUT');
    expect(opts.body).toBe(JSON.stringify({ name: 'Updated' }));
  });

  it('sends DELETE request', async () => {
    fetchSpy.mockResolvedValue(jsonOk({ code: 0, message: 'deleted' }));

    await apiClient.delete('/v1/items/1');

    expect(fetchSpy.mock.calls[0][1].method).toBe('DELETE');
  });
});
