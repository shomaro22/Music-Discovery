export type UserRole = "user" | "content_admin" | "system_admin";
export type UserStatus = "active" | "suspended";

export interface SignUpInput {
  name: string;
  email: string;
  password: string;
}

export interface SignUpResponse {
  id: string;
  name: string;
  email: string;
  role: UserRole;
  status: UserStatus;
  created_at: string;
}

export interface ApiError {
  status: number;
  code: string;
  message: string;
  request_id: string;
}
