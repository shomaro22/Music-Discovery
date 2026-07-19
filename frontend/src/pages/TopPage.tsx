import { Navigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

export function TopPage() {
  const { user, isAuthenticated } = useAuth();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <main><h1>Music Discovery</h1><p>{user?.name}さん、ログインしました。</p></main>;
}
