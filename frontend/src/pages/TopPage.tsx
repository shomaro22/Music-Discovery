import { useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

import { ApiRequestError } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { ROLE } from "../constants/role";
import { ROUTES } from "../constants/routes";

export function TopPage() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const logoutStartedRef = useRef(false);

  const handleLogout = async (): Promise<void> => {
    if (logoutStartedRef.current) {
      return;
    }

    logoutStartedRef.current = true;
    setIsLoggingOut(true);

    try {
      await logout();
      navigate(ROUTES.login, { replace: true });
    } catch (error) {
      const message =
        error instanceof ApiRequestError
          ? error.message
          : "ログアウト処理に失敗しました。再度ログインしてください。";

      navigate(ROUTES.login, {
        replace: true,
        state: { logoutError: message },
      });
    }
  };

  return (
    <main className="top-page top-dashboard-page">
      <section className="top-dashboard" aria-labelledby="top-title">
        <div className="top-dashboard-hero">
          <div className="top-dashboard-copy">
            <p className="top-kicker">Music Discovery</p>
            <h1 id="top-title">次の一曲を、見つけよう。</h1>
            <p className="top-dashboard-lead">
              気分や好みを手がかりに、まだ知らない音楽との出会いをつくります。
            </p>

            <button
              className="top-discovery-button"
              type="button"
              aria-label="おすすめを探す"
              onClick={() =>
                navigate(ROUTES.questions, {
                  state: { loadLatest: true },
                })
              }
            >
              <span>おすすめを探す</span>
              <span aria-hidden="true">→</span>
            </button>
          </div>
        </div>

        <div className="top-dashboard-content">
          <section className="top-welcome-panel" aria-label="ユーザー情報">
            <p className="top-panel-label">Welcome back</p>
            <h2>{user?.name}さん</h2>
            <p>今日も新しい音楽を探しにいきましょう。</p>

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
          </section>

          <section className="top-menu-panel" aria-label="アカウントメニュー">
            <div className="top-menu-heading">
              <p className="top-panel-label">Your menu</p>
              <h2>設定とアカウント</h2>
            </div>

            <div className="top-menu-grid">
              {user?.role === ROLE.USER && (
                <>
                  <button
                    className="top-menu-button"
                    type="button"
                    aria-label="Daily Discoveryを開く"
                    onClick={() => navigate(ROUTES.dailyDiscovery)}
                  >
                    <span className="top-menu-icon" aria-hidden="true">D</span>
                    <span>
                      <strong>Daily Discovery</strong>
                      <small>今日のおすすめ3曲を確認</small>
                    </span>
                  </button>

                  <button
                    className="top-menu-button"
                    type="button"
                    aria-label="推薦履歴を開く"
                    onClick={() => navigate(ROUTES.recommendationHistory)}
                  >
                    <span className="top-menu-icon" aria-hidden="true">R</span>
                    <span>
                      <strong>推薦履歴</strong>
                      <small>通常推薦とDaily Discoveryを確認</small>
                    </span>
                  </button>
                </>
              )}

              <button
                className="top-menu-button"
                type="button"
                aria-label="プロフィールを変更"
                onClick={() => navigate(ROUTES.profile)}
              >
                <span className="top-menu-icon" aria-hidden="true">P</span>
                <span>
                  <strong>プロフィールを変更</strong>
                  <small>名前・メールアドレスを編集</small>
                </span>
              </button>
            </div>

            <div className="top-account-actions">
              {user?.role === ROLE.USER && (
                <button
                  className="top-withdraw-button"
                  type="button"
                  onClick={() => navigate(ROUTES.withdraw)}
                >
                  退会手続きへ
                </button>
              )}

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
        </div>
      </section>
    </main>
  );
}
