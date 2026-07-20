import { useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { useAuth } from "../auth/AuthContext";

export function TopPage() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const logoutStartedRef = useRef(false);

  const handleLogout = async (): Promise<void> => {
    if (logoutStartedRef.current) return;
    logoutStartedRef.current = true;
    setIsLoggingOut(true);

    try {
      await logout();
      navigate("/login", { replace: true });
    } catch (error) {
      const message = error instanceof ApiRequestError
        ? error.message
        : "ログアウト処理に失敗しました。再度ログインしてください。";
      navigate("/login", { replace: true, state: { logoutError: message } });
    }
  };

  return (
    <main>
      <h1>Music Discovery</h1>
      <p>{user?.name}さん、ログインしました。</p>
      <dl>
        <dt>Email</dt><dd>{user?.email}</dd>
        <dt>Role</dt><dd>{user?.role}</dd>
      </dl>
      <button type="button" onClick={() => void handleLogout()} disabled={isLoggingOut}>
        {isLoggingOut ? "ログアウト中..." : "ログアウト"}
      </button>
    </main>
  );
}
