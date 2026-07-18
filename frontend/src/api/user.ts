import { request } from "./client";
import type { SignUpInput, SignUpResponse } from "../types/user";

export function signUp(input: SignUpInput): Promise<SignUpResponse> {
  return request<SignUpResponse>("/api/auth/signup", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
