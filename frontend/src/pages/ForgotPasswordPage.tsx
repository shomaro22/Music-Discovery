import { FormEvent, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { requestPasswordReset } from "../api/user";
import { useAuth } from "../auth/AuthContext";
import { ROUTES } from "../constants/routes";

export function ForgotPasswordPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const { isAuthenticated, logout } = useAuth();
  const locationState = location.state as { email?: string } | null;
  const [email, setEmail] = useState(locationState?.email ?? "");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [resetToken, setResetToken] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [returningToLogin, setReturningToLogin] = useState(false);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) return;
    setError("");
    setMessage("");
    setResetToken("");
    setSubmitting(true);
    try {
      const response = await requestPasswordReset({ email: email.trim().toLowerCase() });
      setMessage(response.message);
      setResetToken(response.reset_token ?? "");
    } catch (caught: unknown) {
      setError(caught instanceof ApiRequestError ? caught.message : "パスワード再設定の受付に失敗しました。");
    } finally {
      setSubmitting(false);
    }
  };

  const returnToLogin = async () => {
    if (returningToLogin) return;
    setReturningToLogin(true);
    try {
      if (isAuthenticated) await logout();
    } finally {
      navigate(ROUTES.login, { replace: true });
    }
  };

  return (
    <main className="auth-page password-reset-page">
      <div className="auth-layout single-auth-layout">
        <section className="auth-card password-reset-card">
          <div className="auth-card-header">
            <h2>パスワードを再設定</h2>
            <p>登録済みのメールアドレスを入力してください。</p>
          </div>
          {error && <div className="api-alert" role="alert">{error}</div>}
          {message && <div className="success-alert" role="status">{message}</div>}
          <form className="auth-form" onSubmit={submit} noValidate>
            <div className="field">
              <label htmlFor="reset-email">メールアドレス</label>
              <input id="reset-email" type="email" required autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} disabled={submitting} />
            </div>
            <button className="primary-button" type="submit" disabled={submitting || !email.trim()}>{submitting ? "送信中..." : "再設定メールを送る"}</button>
          </form>
          {resetToken && (
            <div className="development-reset-link">
              <p>開発環境用の再設定リンクです。</p>
              <Link to={`/reset-password?token=${encodeURIComponent(resetToken)}`}>パスワード再設定へ進む</Link>
            </div>
          )}
          <p className="auth-switch">
            <button className="auth-switch-button" type="button" onClick={() => void returnToLogin()} disabled={returningToLogin}>
              {returningToLogin ? "ログアウト中..." : "ログインへ戻る"}
            </button>
          </p>
        </section>
      </div>
    </main>
  );
}
