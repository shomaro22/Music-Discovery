import type { ReactNode } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { AuthLoading } from "../components/common/AuthLoading";
import { useAuth } from "../hooks/useAuth";

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading, authExpired } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return <AuthLoading />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location, authExpired }} />;
  }

  return <>{children}</>;
}
