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

interface QueueItem {
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
}

class ApiClient {
  private baseUrl: string;
  private refreshing = false;
  private refreshQueue: QueueItem[] = [];

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
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

      const response = await fetch(`${this.baseUrl}/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

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

    const response = await fetch(url.toString(), options);

    // If not a 401 or if we're server-side, just handle the response
    if (response.status !== 401) {
      if (!response.ok) {
        throw new Error(`API error: ${response.status} ${response.statusText}`);
      }
      return response.json();
    }

    // 401 handling — only on client side with a stored token
    if (typeof window === 'undefined') {
      throw new Error(`API error: ${response.status} ${response.statusText}`);
    }

    const originalToken = getToken();
    if (!originalToken) {
      // No token at all — redirect to login
      window.location.href = '/login';
      throw new Error('Not authenticated');
    }

    // Attempt token refresh
    const newToken = await this.refreshAccessToken();

    // Retry the original request with the new token
    const retryOptions: RequestInit = {
      method,
      headers: this.getHeaders(newToken),
      body: body ? JSON.stringify(body) : undefined,
    };

    const retryResponse = await fetch(url.toString(), retryOptions);
    if (!retryResponse.ok) {
      // Retry also failed — refresh token might be valid but access is denied
      throw new Error(`API error: ${retryResponse.status} ${retryResponse.statusText}`);
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
}

export const apiClient = new ApiClient(API_BASE);
export default apiClient;
