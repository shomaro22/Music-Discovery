export type UserRole = "user" | "content_admin" | "system_admin";
export type UserStatus = "active" | "suspended";

export interface SignUpInput { name: string; email: string; password: string; }
export interface SignUpResponse { id: string; name: string; email: string; role: UserRole; status: UserStatus; created_at: string; }
export interface AuthUser { id: string; name: string; email: string; role: UserRole; status: UserStatus; }
export interface LoginInput { email: string; password: string; }
export interface LoginResponse { access_token: string; token_type: "Bearer"; expires_in: number; user: AuthUser; }
export interface ApiError { status: number; code: string; message: string; request_id: string; }
