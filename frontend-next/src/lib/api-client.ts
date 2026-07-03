import type { Result, PageResult } from '@/types/api';
import {
  getToken,
  setToken,
  getRefreshToken,
  setRefreshToken,
  removeToken,
  removeRefreshToken,
} from '@/lib/auth';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

/** Timeout for all fetch requests in milliseconds. */
const REQUEST_TIMEOUT_MS = 30_000;

/** Category labels for structured error reporting. */
export type ApiErrorCategory =
  | 'auth'
  | 'validation'
  | 'server'
  | 'network'
  | 'timeout'
  | 'unknown';

/**
 * Structured error type returned by the API client.
 * Components can inspect `status`, `message`, and `category` to show
 * user-appropriate error messages.
 */
export class ApiError extends Error {
  override name = 'ApiError';

  constructor(
    public readonly status: number,
    message: string,
    public readonly category: ApiErrorCategory = 'unknown',
  ) {
    super(message);
  }

  static fromStatus(status: number, statusText?: string): ApiError {
    let category: ApiErrorCategory;
    if (status === 401 || status === 403) category = 'auth';
    else if (status >= 400 && status < 500) category = 'validation';
    else if (status >= 500) category = 'server';
    else category = 'unknown';
    return new ApiError(status, statusText || `HTTP ${status}`, category);
  }
}

interface QueueItem {
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
}

export class ApiClient {
  private baseUrl: string;
  private refreshing = false;
  private refreshQueue: QueueItem[] = [];
  /** In-flight GET request deduplication map. Keys are full URL strings. */
  private inflightRequests: Map<string, Promise<unknown>> = new Map();

  /** Optional callback invoked on 403 responses. */
  private static forbiddenHandler: (() => void) | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  /**
   * Register a callback that fires whenever a 403 response is received.
   * The permission store uses this to re-fetch permissions automatically.
   */
  static setForbiddenHandler(handler: (() => void) | null): void {
    ApiClient.forbiddenHandler = handler;
  }

  private getHeaders(token?: string): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    } else if (typeof window !== 'undefined') {
      const storedToken = getToken();
      if (storedToken) {
        headers['Authorization'] = `Bearer ${storedToken}`;
      }
    }
    return headers;
  }

  /**
   * Wraps `fetch` with a 30-second timeout and converts transport/abort
   * errors into structured {@link ApiError} instances.
   */
  private async fetchWithTimeout(
    url: string,
    options: RequestInit = {},
  ): Promise<Response> {
    try {
      const response = await fetch(url, {
        ...options,
        signal: options.signal ?? AbortSignal.timeout(REQUEST_TIMEOUT_MS),
      });
      return response;
    } catch (err: unknown) {
      if (err instanceof Error && err.name === 'AbortError') {
        throw new ApiError(0, 'Request timeout', 'timeout');
      }
      throw new ApiError(
        0,
        err instanceof Error ? err.message : 'Network error',
        'network',
      );
    }
  }

  /**
   * Attempt to refresh the access token using the stored refresh token.
   * Uses a queue to prevent multiple concurrent refresh calls.
   */
  private async refreshAccessToken(): Promise<string> {
    // If a refresh is already in progress, queue this request to wait for it
    if (this.refreshing) {
      return new Promise<string>((resolve, reject) => {
        this.refreshQueue.push({ resolve, reject });
      });
    }

    this.refreshing = true;

    try {
      const refreshToken = getRefreshToken();
      if (!refreshToken) {
        throw new Error('No refresh token available');
      }

      const response = await this.fetchWithTimeout(
        `${this.baseUrl}/v1/auth/refresh`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken }),
        },
      );

      if (!response.ok) {
        throw new Error(`Refresh failed: ${response.status}`);
      }

      const result: Result<{
        access_token: string;
        refresh_token: string;
      }> = await response.json();

      if (!result.data) {
        throw new Error('Refresh response missing data');
      }

      const { access_token, refresh_token } = result.data;
      setToken(access_token);
      setRefreshToken(refresh_token);

      // Resolve all queued requests with the new token
      this.refreshQueue.forEach((item) => item.resolve(access_token));
      this.refreshQueue = [];

      return access_token;
    } catch (error) {
      // Reject all queued requests
      this.refreshQueue.forEach((item) => item.reject(error));
      this.refreshQueue = [];

      // Clear auth data and redirect
      removeToken();
      removeRefreshToken();

      if (typeof window !== 'undefined') {
        window.location.href = '/login';
      }

      throw new Error('Session expired — please log in again');
    } finally {
      this.refreshing = false;
    }
  }

  /**
   * Perform a fetch request with automatic 401 handling and token refresh.
   * For GET requests, concurrent identical calls are deduplicated.
   */
  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    params?: Record<string, string>,
  ): Promise<T> {
    const url = new URL(`${this.baseUrl}${path}`);
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        url.searchParams.append(key, value);
      });
    }

    const options: RequestInit = {
      method,
      headers: this.getHeaders(),
      body: body ? JSON.stringify(body) : undefined,
    };

    // GET deduplication: if the same URL is already in-flight, reuse the promise.
    if (method === 'GET') {
      const urlStr = url.toString();
      const existing = this.inflightRequests.get(urlStr);
      if (existing) return existing as Promise<T>;

      const promise = this.executeWithRefresh<T>(url, options);
      this.inflightRequests.set(urlStr, promise);
      promise.finally(() => {
        // Only remove if ours is still the active promise (avoid clearing a
        // newer identical request's promise).
        if (this.inflightRequests.get(urlStr) === promise) {
          this.inflightRequests.delete(urlStr);
        }
      });
      return promise;
    }

    return this.executeWithRefresh<T>(url, options);
  }

  /**
   * Execute the fetch and handle 401 → token refresh → retry.
   * Also catches 403 to trigger the permission-reload callback.
   */
  private async executeWithRefresh<T>(
    url: URL,
    options: RequestInit,
  ): Promise<T> {
    const response = await this.fetchWithTimeout(url.toString(), options);

    // 403 — trigger the forbidden handler (typically re-fetches permissions)
    if (response.status === 403) {
      ApiClient.forbiddenHandler?.();
      throw ApiError.fromStatus(403);
    }

    // If not a 401 or if we're server-side, just handle the response
    if (response.status !== 401) {
      if (!response.ok) {
        throw ApiError.fromStatus(response.status, response.statusText);
      }
      return response.json();
    }

    // 401 handling — only on client side with a stored token
    if (typeof window === 'undefined') {
      throw ApiError.fromStatus(401, response.statusText);
    }

    const originalToken = getToken();
    if (!originalToken) {
      // No token at all — redirect to login
      window.location.href = '/login';
      throw new ApiError(401, 'Not authenticated', 'auth');
    }

    // Attempt token refresh
    const newToken = await this.refreshAccessToken();

    // Retry the original request with the new token
    const retryOptions: RequestInit = {
      method: options.method || 'GET',
      headers: this.getHeaders(newToken),
      body: options.body,
    };

    const retryResponse = await this.fetchWithTimeout(
      url.toString(),
      retryOptions,
    );
    if (!retryResponse.ok) {
      // Retry also failed — refresh token might be valid but access is denied
      throw ApiError.fromStatus(retryResponse.status, retryResponse.statusText);
    }

    return retryResponse.json();
  }

  async get<T>(path: string, params?: Record<string, string>): Promise<Result<T>> {
    return this.request<Result<T>>('GET', path, undefined, params);
  }

  async getPage<T>(path: string, params?: Record<string, string>): Promise<PageResult<T>> {
    return this.request<PageResult<T>>('GET', path, undefined, params);
  }

  async post<T>(path: string, body?: unknown): Promise<Result<T>> {
    return this.request<Result<T>>('POST', path, body);
  }

  async put<T>(path: string, body?: unknown): Promise<Result<T>> {
    return this.request<Result<T>>('PUT', path, body);
  }

  async delete<T>(path: string): Promise<Result<T>> {
    return this.request<Result<T>>('DELETE', path);
  }

  /**
   * Upload a file via multipart/form-data (FormData).
   * Unlike JSON methods, this does NOT set Content-Type — the browser
   * sets the correct multipart boundary automatically.
   */
  async upload<T>(path: string, formData: FormData): Promise<Result<T>> {
    const url = new URL(`${this.baseUrl}${path}`);
    const token = typeof window !== 'undefined' ? getToken() : null;
    const headers: HeadersInit = {};
    if (token) headers['Authorization'] = `Bearer ${token}`;

    const options: RequestInit = {
      method: 'POST',
      headers,
      body: formData,
    };

    return this.executeWithRefresh<Result<T>>(url, options);
  }
}

export const apiClient = new ApiClient(API_BASE);
export default apiClient;
