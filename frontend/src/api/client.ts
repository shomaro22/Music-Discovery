import type { ApiError } from "../types/user";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export class ApiRequestError extends Error {
  constructor(public readonly response: ApiError) {
    super(response.message);
  }
}

export async function request<T>(path: string, init: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init.headers ?? {}) },
  });

  const body: unknown = await response.json();
  if (!response.ok) {
    throw new ApiRequestError(body as ApiError);
  }
  return body as T;
}
