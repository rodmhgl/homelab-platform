import { config } from '../utils/config';

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private buildUrl(path: string, params?: Record<string, string>): string {
    const fullPath = `${this.baseUrl}${path}`;

    // If baseUrl is absolute (http/https), use URL constructor
    // Otherwise, for relative URLs (same-origin), use plain string concatenation
    if (this.baseUrl.startsWith('http://') || this.baseUrl.startsWith('https://')) {
      const url = new URL(fullPath);
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          url.searchParams.append(key, value);
        });
      }
      return url.toString();
    }

    // For relative URLs (empty baseUrl or path-only), build query string manually
    if (!params || Object.keys(params).length === 0) {
      return fullPath;
    }

    const queryString = new URLSearchParams(params).toString();
    return `${fullPath}?${queryString}`;
  }

  async request<T>(
    path: string,
    options: RequestInit & { params?: Record<string, string> } = {}
  ): Promise<T> {
    const { params, ...fetchOptions } = options;
    const url = this.buildUrl(path, params);

    const response = await fetch(url, {
      ...fetchOptions,
      headers: {
        'Content-Type': 'application/json',
        ...fetchOptions.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: response.statusText }));
      throw new Error(error.message || 'Request failed');
    }

    return response.json();
  }

  get<T>(path: string, params?: Record<string, string>): Promise<T> {
    return this.request<T>(path, { method: 'GET', params });
  }

  post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  delete<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: 'DELETE' });
  }
}

export const apiClient = new ApiClient(config.apiUrl + '/api/v1');
