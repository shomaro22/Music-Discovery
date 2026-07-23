import type { ReactNode } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { AuthLoading } from "../components/common/AuthLoading";
import type { UserRole } from "../types/user";
import { useAuth } from "../hooks/useAuth";

interface AdminRouteProps {
  children: ReactNode;
  allowedRoles: readonly UserRole[];
}

export function AdminRoute({ children, allowedRoles }: AdminRouteProps) {
  const { user, isAuthenticated, isLoading, authExpired } = useAuth();
  const location = useLocation();
  if (isLoading) return <AuthLoading />;
  if (!isAuthenticated || user === null) {
    return <Navigate to="/login" replace state={{ from: location, authExpired }} />;
  }
  if (allowedRoles.length === 0 || !allowedRoles.includes(user.role)) {
    return <Navigate to="/forbidden" replace />;
  }
  return <>{children}</>;
}
