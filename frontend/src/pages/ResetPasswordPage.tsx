import { FormEvent, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { confirmPasswordReset } from "../api/user";
import { useAuth } from "../auth/AuthContext";
import { PasswordToggleIcon } from "../components/common/PasswordToggleIcon";
import { ROUTES } from "../constants/routes";
import { countUtf8Bytes } from "../utils/stringLength";

export function ResetPasswordPage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const { isAuthenticated, logout } = useAuth();
  const [token, setToken] = useState(params.get("token") ?? "");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmation, setShowConfirmation] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [returningToLogin, setReturningToLogin] = useState(false);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) return;
    setError("");
    setMessage("");
    if (!token.trim()) return setError("再設定トークンを入力してください。");
    const bytes = countUtf8Bytes(password);
    if (bytes < 8 || bytes > 72) return setError("パスワードは8文字以上72バイト以内で入力してください。");
    if (password !== confirmation) return setError("パスワードが一致しません。");
    setSubmitting(true);
    try {
      const response = await confirmPasswordReset({ token: token.trim(), new_password: password });
      setMessage(response.message);
      setPassword("");
      setConfirmation("");
    } catch (caught: unknown) {
      setError(caught instanceof ApiRequestError ? caught.message : "パスワードを再設定できませんでした。");
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
          <div className="auth-card-header"><h2>新しいパスワード</h2><p>再設定トークンと新しいパスワードを入力してください。</p></div>
          {error && <div className="api-alert" role="alert">{error}</div>}
          {message && <div className="success-alert" role="status">{message}</div>}
          {!message && <form className="auth-form" onSubmit={submit}>
            <div className="field"><label htmlFor="reset-token">再設定トークン</label><input id="reset-token" value={token} onChange={(event) => setToken(event.target.value)} autoComplete="off" disabled={submitting} /></div>
            <div className="field"><label htmlFor="new-password">新しいパスワード</label><div className="password-field"><input id="new-password" type={showPassword ? "text" : "password"} value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="new-password" disabled={submitting} /><button className="password-toggle" type="button" onClick={() => setShowPassword((value) => !value)} aria-label={showPassword ? "パスワードを非表示にする" : "パスワードを表示する"}><PasswordToggleIcon hidden={showPassword} /></button></div></div>
            <div className="field"><label htmlFor="new-password-confirmation">新しいパスワード確認</label><div className="password-field"><input id="new-password-confirmation" type={showConfirmation ? "text" : "password"} value={confirmation} onChange={(event) => setConfirmation(event.target.value)} autoComplete="new-password" disabled={submitting} /><button className="password-toggle" type="button" onClick={() => setShowConfirmation((value) => !value)} aria-label={showConfirmation ? "パスワード確認を非表示にする" : "パスワード確認を表示する"}><PasswordToggleIcon hidden={showConfirmation} /></button></div></div>
            <button className="primary-button" type="submit" disabled={submitting}>{submitting ? "変更中..." : "パスワードを変更"}</button>
          </form>}
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
