import type { ApiError, RefreshResponse } from "../types/user";
import { getCSRFCookie } from "./csrf";

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

let accessToken: string | null = null;
let refreshPromise: Promise<string> | null = null;

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

export class ApiRequestError extends Error {
  constructor(public readonly response: ApiError) {
    super(response.message);
  }
}
export { ApiRequestError as ApiClientError };

async function parseError(response: Response): Promise<ApiRequestError> {
  const fallback: ApiError = {
    status: response.status,
    code: "UNKNOWN_ERROR",
    message: "処理に失敗しました",
    request_id: response.headers.get("X-Request-ID") ?? "",
  };
  try {
    return new ApiRequestError((await response.json()) as ApiError);
  } catch {
    return new ApiRequestError(fallback);
  }
}

async function refreshAccessToken(): Promise<string> {
  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    const csrfToken = getCSRFCookie();
    const response = await fetch(`${API_BASE_URL}/api/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: csrfToken ? { "X-CSRF-Token": csrfToken } : {},
    });
    if (!response.ok) {
      setAccessToken(null);
      throw await parseError(response);
    }
    const result = (await response.json()) as RefreshResponse;
    setAccessToken(result.access_token);
    return result.access_token;
  })().finally(() => {
    refreshPromise = null;
  });

  return refreshPromise;
}

export async function request<T>(path: string, init: RequestInit, retried = false): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      ...init,
      credentials: "include",
      headers,
    });
  } catch {
    throw new ApiRequestError({
      status: 0,
      code: "NETWORK_ERROR",
      message: "サーバーに接続できませんでした",
      request_id: "",
    });
  }

  if (response.status === 401 && path !== "/api/auth/refresh" && !retried) {
    const token = await refreshAccessToken();
    const retryHeaders = new Headers(init.headers);
    retryHeaders.set("Authorization", `Bearer ${token}`);
    return request<T>(path, { ...init, headers: retryHeaders }, true);
  }
  if (!response.ok) throw await parseError(response);
  return (await response.json()) as T;
}
