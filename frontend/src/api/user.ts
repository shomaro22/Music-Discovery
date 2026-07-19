import type { LoginInput, LoginResponse, RefreshResponse, SignUpInput, SignUpResponse } from "../types/user";
import { request } from "./client";
import { getCsrfToken } from "./csrf";

export function signUp(input: SignUpInput): Promise<SignUpResponse> {
  return request<SignUpResponse>("/api/auth/signup", { method: "POST", body: JSON.stringify(input) });
}

export function login(input: LoginInput): Promise<LoginResponse> {
  return request<LoginResponse>("/api/auth/login", { method: "POST", body: JSON.stringify(input) });
}

export function refreshToken(): Promise<RefreshResponse> { const csrf=getCsrfToken(); return request<RefreshResponse>("/api/auth/refresh",{method:"POST",headers:csrf?{"X-CSRF-Token":csrf}:{}},{skipAuth:true,skipRefresh:true}); }
