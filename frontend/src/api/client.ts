import type { ApiError, RefreshResponse } from "../types/user";
import { getCSRFCookie } from "./csrf";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";
let accessToken: string | null = null;
let refreshPromise: Promise<RefreshResponse> | null = null;
let authStateListener: ((token: string | null, reason?: "expired") => void) | null = null;
let authExpirationNotified = false;

export function setAccessToken(token: string | null, reason?: "expired"): void {
  accessToken = token;
  if (token !== null) authExpirationNotified = false;
  authStateListener?.(token, reason);
}

export function setAuthStateListener(
  listener: ((token: string | null, reason?: "expired") => void) | null,
): void {
  authStateListener = listener;
}

export function hasAccessToken(): boolean {
  return accessToken !== null;
}

function expireAuthenticationOnce(): void {
  if (authExpirationNotified) return;
  authExpirationNotified = true;
  accessToken = null;
  authStateListener?.(null, "expired");
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

export function isAuthenticationFailure(error: unknown): boolean {
  return (
    error instanceof ApiRequestError &&
    (error.response.status === 401 ||
      error.response.code === "INVALID_REFRESH_TOKEN" ||
      error.response.code === "REFRESH_TOKEN_REUSED" ||
      error.response.code === "USER_SUSPENDED")
  );
}

export async function refreshAccessToken(): Promise<RefreshResponse> {
  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    try {
      const csrfToken = getCSRFCookie();
      if (!csrfToken) {
        throw new ApiRequestError({
          status: 401,
          code: "INVALID_REFRESH_TOKEN",
          message: "認証期限が切れています",
          request_id: "",
        });
      }

      const response = await fetch(`${API_BASE_URL}/api/auth/refresh`, {
        method: "POST",
        credentials: "include",
        headers: { "X-CSRF-Token": csrfToken },
      });
      if (!response.ok) throw await parseError(response);

      const result = (await response.json()) as RefreshResponse;
      setAccessToken(result.access_token);
      return result;
    } catch (error: unknown) {
      // This block belongs to the shared refresh promise, so concurrent callers
      // initialize the expired state only once.
      if (isAuthenticationFailure(error)) expireAuthenticationOnce();
      throw error;
    }
  })().finally(() => {
    refreshPromise = null;
  });

  return refreshPromise;
}

export async function request<T>(
  path: string,
  init: RequestInit,
  retried = false,
): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const requestHadAccessToken = accessToken !== null;
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

  const canRefresh =
    response.status === 401 &&
    requestHadAccessToken &&
    path !== "/api/auth/refresh" &&
    path !== "/api/auth/login" &&
    path !== "/api/auth/signup" &&
    !retried;

  if (canRefresh) {
    const originalError = await parseError(response);
    if (originalError.response.code === "CURRENT_PASSWORD_MISMATCH") {
      throw originalError;
    }
    try {
      await refreshAccessToken();
      // await is required so a 401 from the single retry is handled below.
      return await request<T>(path, init, true);
    } catch (error: unknown) {
      if (isAuthenticationFailure(error)) expireAuthenticationOnce();
      throw error;
    }
  }

  if (!response.ok) {
    const error = await parseError(response);
    const shouldExpire =
      error.response.code === "USER_SUSPENDED" ||
      (requestHadAccessToken && isAuthenticationFailure(error));
    if (shouldExpire) expireAuthenticationOnce();
    throw error;
  }

  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}
