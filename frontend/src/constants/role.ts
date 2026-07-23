import type { UserRole } from "../types/user";

export const ROLE = {
  GUEST: "guest",
  USER: "user",
  CONTENT_ADMIN: "content_admin",
  SYSTEM_ADMIN: "system_admin",
} as const;

export type AppRole = (typeof ROLE)[keyof typeof ROLE];

export function isUserRole(value: unknown): value is UserRole {
  return value === ROLE.USER || value === ROLE.CONTENT_ADMIN || value === ROLE.SYSTEM_ADMIN;
}
