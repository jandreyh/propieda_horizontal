import { Platform } from "react-native";

// Android emulator uses 10.0.2.2 to reach the host machine's localhost.
// iOS simulator and web can use localhost directly.
const DEFAULT_BASE_URL =
  Platform.OS === "android" ? "http://10.0.2.2:8080" : "http://localhost:8080";

const BASE_URL = DEFAULT_BASE_URL;

interface ApiResponse<T> {
  data?: T;
  error?: string;
  status: number;
}

let bearerToken: string | null = null;

export function setBearer(token: string | null) {
  bearerToken = token;
}

export function getBearer(): string | null {
  return bearerToken;
}

export async function apiRequest<T>(
  method: string,
  path: string,
  body?: unknown,
  headers?: Record<string, string>,
): Promise<ApiResponse<T>> {
  const url = `${BASE_URL}${path}`;

  const defaultHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
  };
  if (bearerToken) {
    defaultHeaders["Authorization"] = `Bearer ${bearerToken}`;
  }

  try {
    const response = await fetch(url, {
      method,
      headers: { ...defaultHeaders, ...headers },
      body: body ? JSON.stringify(body) : undefined,
    });

    const status = response.status;

    if (!response.ok) {
      const errorBody = await response.text();
      return {
        error: errorBody || `Request failed with status ${status}`,
        status,
      };
    }

    const data = (await response.json()) as T;
    return { data, status };
  } catch (err) {
    const message = err instanceof Error ? err.message : "Unknown network error";
    return { error: message, status: 0 };
  }
}

export function get<T>(path: string, headers?: Record<string, string>) {
  return apiRequest<T>("GET", path, undefined, headers);
}

export function post<T>(path: string, body: unknown, headers?: Record<string, string>) {
  return apiRequest<T>("POST", path, body, headers);
}

export function put<T>(path: string, body: unknown, headers?: Record<string, string>) {
  return apiRequest<T>("PUT", path, body, headers);
}

export function del<T>(path: string, headers?: Record<string, string>) {
  return apiRequest<T>("DELETE", path, undefined, headers);
}

// --- Tipos compartidos post-Fase 16 ---

export interface Membership {
  tenant_id: string;
  tenant_slug: string;
  tenant_name: string;
  logo_url?: string | null;
  primary_color?: string | null;
  role: string;
  status: string;
}

export interface LoginResponse {
  access_token?: string;
  refresh_token?: string;
  token_type?: string;
  expires_in?: number;
  memberships?: Membership[];
  needs_tenant?: boolean;
  mfa_required?: boolean;
  pre_auth_token?: string;
}

export interface SwitchTenantResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  current_tenant: Membership;
}
