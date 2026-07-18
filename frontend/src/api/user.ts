import type { SignUpInput, SignUpResponse } from "../types/user";
import { request } from "./client";

export function signUp(input: SignUpInput): Promise<SignUpResponse> {
  return request<SignUpResponse>("/api/auth/signup", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
