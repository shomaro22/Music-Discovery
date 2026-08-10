import type { AuthUser, LoginInput, LoginResponse, RefreshResponse, SignUpInput, SignUpResponse, WithdrawInput, WithdrawResponse, UpdateProfileInput, PasswordResetRequestInput, PasswordResetRequestResponse, PasswordResetConfirmInput, PasswordResetConfirmResponse } from "../types/user";
import { ApiRequestError, bootstrapCSRF, refreshAccessToken, request } from "./client";
import { API_PATHS } from "../constants/apiPaths";
import { clearCSRFToken, getCSRFToken, setCSRFToken } from "./csrf";

export async function signUp(input: SignUpInput): Promise<SignUpResponse> { const response = await request<SignUpResponse>("/api/auth/signup", { method: "POST", body: JSON.stringify(input) }); setCSRFToken(response.csrf_token); return response; }
export async function login(input: LoginInput): Promise<LoginResponse> { const response = await request<LoginResponse>("/api/auth/login", { method: "POST", body: JSON.stringify(input) }); setCSRFToken(response.csrf_token); return response; }
export function refresh(): Promise<RefreshResponse> { return refreshAccessToken(); }
export async function logout(): Promise<void> {
  try {
    let csrfToken = getCSRFToken();
    if (!csrfToken) {
      const bootstrap = await bootstrapCSRF();
      csrfToken = bootstrap.csrf_token;
    }

    try {
      await request<void>("/api/auth/logout", {
        method: "POST",
        headers: { "X-CSRF-Token": csrfToken },
      });
    } catch (error: unknown) {
      if (!(error instanceof ApiRequestError) || error.response.code !== "CSRF_TOKEN_INVALID") {
        throw error;
      }

      // Recover once from an old/stale CSRF pair after a deployment or page restore.
      // The backend still validates both the trusted Origin and the cookie/header pair.
      const bootstrap = await bootstrapCSRF();
      await request<void>("/api/auth/logout", {
        method: "POST",
        headers: { "X-CSRF-Token": bootstrap.csrf_token },
      });
    }
  } finally {
    clearCSRFToken();
  }
}
export function getCurrentUser(): Promise<AuthUser> { return request<AuthUser>("/api/users/me", { method: "GET" }); }

export function withdraw(input: WithdrawInput): Promise<WithdrawResponse> {
  return request<WithdrawResponse>("/api/users/me", {
    method: "DELETE",
    body: JSON.stringify(input),
  });
}

export function updateProfile(input: UpdateProfileInput): Promise<AuthUser> {
  return request<AuthUser>("/api/users/me", { method: "PATCH", body: JSON.stringify(input) });
}

export function requestPasswordReset(input: PasswordResetRequestInput): Promise<PasswordResetRequestResponse> { return request<PasswordResetRequestResponse>(API_PATHS.passwordResetRequest, { method: "POST", body: JSON.stringify(input) }); }
export function confirmPasswordReset(input: PasswordResetConfirmInput): Promise<PasswordResetConfirmResponse> { return request<PasswordResetConfirmResponse>(API_PATHS.passwordResetConfirm, { method: "POST", body: JSON.stringify(input) }); }
