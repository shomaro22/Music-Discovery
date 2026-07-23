import { useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { Navigation } from "../components/navigation/Navigation";
import { ROLE } from "../constants/role";

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
    <><Navigation /><main className="top-page">
      <section className="top-card" aria-labelledby="top-title">
        <p className="top-kicker">Music Discovery</p>
        <h1 id="top-title">ようこそ、{user?.name}さん</h1>
        <p className="top-description">ログインしています。</p>

        <dl className="top-profile">
          <div>
            <dt>Email</dt>
            <dd>{user?.email}</dd>
          </div>
          <div>
            <dt>Role</dt>
            <dd>{user?.role}</dd>
          </div>
        </dl>

        <div className="top-actions">
          {user?.role === ROLE.USER && <button
            className="top-withdraw-button"
            type="button"
            onClick={() => navigate("/withdraw")}
          >
            退会手続きへ
          </button>}
          <button
            className="top-logout-button"
            type="button"
            onClick={() => void handleLogout()}
            disabled={isLoggingOut}
          >
            {isLoggingOut ? "ログアウト中..." : "ログアウト"}
          </button>
        </div>
      </section>
    </main></>
  );
}
