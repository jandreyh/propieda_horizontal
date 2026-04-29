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

export async function apiRequest<T>(
  method: string,
  path: string,
  body?: unknown,
  headers?: Record<string, string>
): Promise<ApiResponse<T>> {
  const url = `${BASE_URL}${path}`;

  const defaultHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
  };

  try {
    const response = await fetch(url, {
      method,
      headers: { ...defaultHeaders, ...headers },
      body: body ? JSON.stringify(body) : undefined,
    });

    const status = response.status;

    if (!response.ok) {
      const errorBody = await response.text();
      return { error: errorBody || `Request failed with status ${status}`, status };
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
