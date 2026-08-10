export type UserRole = "user" | "content_admin" | "system_admin";
export type UserStatus = "active" | "suspended";

export interface SignUpInput { name: string; email: string; password: string; }
export interface SignUpUser extends AuthUser { created_at: string; }
export interface SignUpResponse { access_token: string; token_type: "Bearer"; expires_in: number; csrf_token: string; user: SignUpUser; }
export interface AuthUser { id: string; name: string; email: string; role: UserRole; status: UserStatus; }
export interface LoginInput { email: string; password: string; }
export interface LoginResponse { access_token: string; token_type: "Bearer"; expires_in: number; csrf_token: string; user: AuthUser; }
export interface ApiError { status: number; code: string; message: string; request_id: string; }

export interface RefreshResponse { access_token: string; token_type: "Bearer"; expires_in: number; csrf_token: string; }
export interface CSRFResponse { csrf_token: string; refresh_available: boolean; }

export interface WithdrawInput { current_password: string; }
export interface WithdrawResponse { message: string; }

export interface UpdateProfileInput { name: string; email: string; }

export interface PasswordResetRequestInput { email: string; }
export interface PasswordResetRequestResponse { message: string; reset_token?: string; }
export interface PasswordResetConfirmInput { token: string; new_password: string; }
export interface PasswordResetConfirmResponse { message: string; }
