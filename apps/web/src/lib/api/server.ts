import { cookies } from "next/headers";
import { API_BASE_URL, TENANT_SLUG, COOKIE_ACCESS } from "./config";
import type { ProblemDetails } from "./types";

export class ApiError extends Error {
  constructor(public problem: ProblemDetails) {
    super(problem.detail || problem.title);
  }
}

export async function getAccessToken(): Promise<string | null> {
  const store = await cookies();
  return store.get(COOKIE_ACCESS)?.value ?? null;
}

interface FetchOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  skipAuth?: boolean;
  query?: Record<string, string | number | undefined>;
}

export async function apiFetch<T>(path: string, opts: FetchOptions = {}): Promise<T> {
  const { body, skipAuth, query, headers: extraHeaders, ...rest } = opts;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    "X-Tenant-Slug": TENANT_SLUG,
    ...((extraHeaders as Record<string, string>) ?? {}),
  };

  if (!skipAuth) {
    const token = await getAccessToken();
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  let url = `${API_BASE_URL}${path}`;
  if (query) {
    const params = new URLSearchParams();
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null && v !== "") params.set(k, String(v));
    }
    const qs = params.toString();
    if (qs) url += `?${qs}`;
  }

  const res = await fetch(url, {
    ...rest,
    headers,
    body: body ? JSON.stringify(body) : undefined,
    cache: "no-store",
  });

  if (!res.ok) {
    let problem: ProblemDetails;
    try {
      problem = await res.json();
    } catch {
      problem = {
        type: "about:blank",
        title: res.statusText,
        status: res.status,
        detail: res.statusText,
      };
    }
    throw new ApiError(problem);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

export const apiGet = <T>(path: string, query?: Record<string, string | number | undefined>) =>
  apiFetch<T>(path, { method: "GET", query });

export const apiPost = <T>(path: string, body?: unknown, opts: Partial<FetchOptions> = {}) =>
  apiFetch<T>(path, { method: "POST", body, ...opts });

export const apiPut = <T>(path: string, body?: unknown) =>
  apiFetch<T>(path, { method: "PUT", body });

export const apiDelete = <T>(path: string) => apiFetch<T>(path, { method: "DELETE" });
