import { ApiError, networkError, toApiError } from "./errors";
import type { paths } from "./gen";

export type ApiPath = keyof paths;

export type QueryValue = string | number | boolean | null | undefined;

export interface RequestOptions {
  query?: Record<string, QueryValue>;
  signal?: AbortSignal;
  headers?: HeadersInit;
}

export interface ApiClientOptions {
  baseUrl: string;
  getAuthToken?: () => string | undefined;
  fetchImpl?: typeof fetch;
}

type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

export function buildPath(template: ApiPath, params: Record<string, string | number>): ApiPath {
  return template.replace(/\{(\w+)\}/g, (_match, key: string) =>
    encodeURIComponent(String(params[key] ?? "")),
  ) as ApiPath;
}

export class ApiClient {
  private readonly baseUrl: string;
  private readonly getAuthToken: (() => string | undefined) | undefined;
  private readonly fetchImpl: typeof fetch;

  constructor(options: ApiClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, "");
    this.getAuthToken = options.getAuthToken;
    this.fetchImpl = options.fetchImpl ?? ((input, init) => fetch(input, init));
  }

  get<T>(path: ApiPath, options?: RequestOptions): Promise<T> {
    return this.request<T>("GET", path, undefined, options);
  }

  post<T>(path: ApiPath, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>("POST", path, body, options);
  }

  put<T>(path: ApiPath, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>("PUT", path, body, options);
  }

  patch<T>(path: ApiPath, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>("PATCH", path, body, options);
  }

  delete<T>(path: ApiPath, options?: RequestOptions): Promise<T> {
    return this.request<T>("DELETE", path, undefined, options);
  }

  private async request<T>(
    method: HttpMethod,
    path: ApiPath,
    body: unknown,
    options?: RequestOptions,
  ): Promise<T> {
    const headers = new Headers(options?.headers);
    if (body !== undefined) headers.set("Content-Type", "application/json");

    const token = this.getAuthToken?.();
    if (token !== undefined && token.length > 0) headers.set("Authorization", `Bearer ${token}`);

    let response: Response;
    try {
      response = await this.fetchImpl(this.buildUrl(path, options?.query), {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: options?.signal,
      });
    } catch (error) {
      if (error instanceof ApiError) throw error;
      throw networkError();
    }

    if (!response.ok) throw await toApiError(response);
    if (response.status === 204) return undefined as T;

    const text = await response.text();
    if (text.length === 0) return undefined as T;
    return JSON.parse(text) as T;
  }

  private buildUrl(path: ApiPath, query?: Record<string, QueryValue>): string {
    const url = new URL(`${this.baseUrl}${path}`);
    if (query !== undefined) {
      for (const [key, value] of Object.entries(query)) {
        if (value === undefined || value === null) continue;
        url.searchParams.set(key, String(value));
      }
    }
    return url.toString();
  }
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const apiClient = new ApiClient({ baseUrl: API_BASE_URL });
