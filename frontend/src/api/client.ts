import type { ApiError } from "../types/user";

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export class ApiRequestError extends Error {
  constructor(public readonly response: ApiError) {
    super(response.message);
  }
}
export { ApiRequestError as ApiClientError };

export async function request<T>(path: string, init: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      ...init,
      credentials: "include",
      headers: { "Content-Type": "application/json", ...init.headers },
    });
  } catch {
    throw new ApiRequestError({
      status: 0,
      code: "NETWORK_ERROR",
      message: "サーバーに接続できませんでした",
      request_id: "",
    });
  }
  if (!response.ok) {
    const fallback: ApiError = {
      status: response.status,
      code: "UNKNOWN_ERROR",
      message: "処理に失敗しました",
      request_id: response.headers.get("X-Request-ID") ?? "",
    };
    let error = fallback;
    try {
      error = (await response.json()) as ApiError;
    } catch {
      /* fallbackを使用 */
    }
    throw new ApiRequestError(error);
  }
  return (await response.json()) as T;
}
