import { ROLE, type AppRole } from "../constants/role";
import { useAuth } from "./useAuth";

export function useRole() {
  const { user } = useAuth();
  const role: AppRole = user?.role ?? ROLE.GUEST;
  const hasRole = (allowedRole: AppRole): boolean => role === allowedRole;
  const hasAnyRole = (allowedRoles: readonly AppRole[]): boolean => allowedRoles.includes(role);

  return {
    role,
    isGuest: role === ROLE.GUEST,
    isUser: role === ROLE.USER,
    isContentAdmin: role === ROLE.CONTENT_ADMIN,
    isSystemAdmin: role === ROLE.SYSTEM_ADMIN,
    isAdmin: role === ROLE.CONTENT_ADMIN || role === ROLE.SYSTEM_ADMIN,
    hasRole,
    hasAnyRole,
  };
}
