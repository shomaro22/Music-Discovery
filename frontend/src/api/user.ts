import type { AuthUser, LoginInput, LoginResponse, RefreshResponse, SignUpInput, SignUpResponse } from "../types/user";
import { refreshAccessToken, request } from "./client";
import { getCSRFCookie } from "./csrf";

export function signUp(input: SignUpInput): Promise<SignUpResponse> { return request<SignUpResponse>("/api/auth/signup", { method: "POST", body: JSON.stringify(input) }); }
export function login(input: LoginInput): Promise<LoginResponse> { return request<LoginResponse>("/api/auth/login", { method: "POST", body: JSON.stringify(input) }); }
export function refresh(): Promise<RefreshResponse> { return refreshAccessToken(); }
export function logout(): Promise<void> {
  const csrfToken = getCSRFCookie();
  return request<void>("/api/auth/logout", { method: "POST", headers: csrfToken ? { "X-CSRF-Token": csrfToken } : undefined });
}
export function getCurrentUser(): Promise<AuthUser> { return request<AuthUser>("/api/users/me", { method: "GET" }); }
