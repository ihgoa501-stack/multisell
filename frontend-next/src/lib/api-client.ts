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
  private inflightRequests: Map<string, Promise<unknown>> = new Map();

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

  private async refreshAccessToken(): Promise<string> {
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

      this.refreshQueue.forEach((item) => item.resolve(access_token));
      this.refreshQueue = [];

      return access_token;
    } catch (error) {
      this.refreshQueue.forEach((item) => item.reject(error));
      this.refreshQueue = [];

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

    if (method === 'GET') {
      const urlStr = url.toString();
      const existing = this.inflightRequests.get(urlStr);
      if (existing) return existing as Promise<T>;

      const promise = this.executeWithRefresh<T>(url, options);
      this.inflightRequests.set(urlStr, promise);
      promise.finally(() => {
        if (this.inflightRequests.get(urlStr) === promise) {
          this.inflightRequests.delete(urlStr);
        }
      });
      return promise;
    }

    return this.executeWithRefresh<T>(url, options);
  }

  private async executeWithRefresh<T>(
    url: URL,
    options: RequestInit,
  ): Promise<T> {
    const response = await fetch(url.toString(), options);

    if (response.status !== 401) {
      if (!response.ok) {
        throw new Error(`API error: ${response.status} ${response.statusText}`);
      }
      return response.json();
    }

    if (typeof window === 'undefined') {
      throw new Error(`API error: ${response.status} ${response.statusText}`);
    }

    const originalToken = getToken();
    if (!originalToken) {
      window.location.href = '/login';
      throw new Error('Not authenticated');
    }

    const newToken = await this.refreshAccessToken();

    const retryOptions: RequestInit = {
      method: options.method || 'GET',
      headers: this.getHeaders(newToken),
      body: options.body,
    };

    const retryResponse = await fetch(url.toString(), retryOptions);
    if (!retryResponse.ok) {
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
