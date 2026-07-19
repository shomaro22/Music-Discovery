import type { LoginInput, LoginResponse, SignUpInput, SignUpResponse } from "../types/user";
import { request } from "./client";

export function signUp(input: SignUpInput): Promise<SignUpResponse> {
  return request<SignUpResponse>("/api/auth/signup", { method: "POST", body: JSON.stringify(input) });
}

export function login(input: LoginInput): Promise<LoginResponse> {
  return request<LoginResponse>("/api/auth/login", { method: "POST", body: JSON.stringify(input) });
}
