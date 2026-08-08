import { useState, type FormEvent } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import type { LoginInput } from "../types/user";
import { countUnicodeCharacters, countUtf8Bytes } from "../utils/stringLength";

type FieldErrors = Partial<Record<keyof LoginInput, string>>;

function EyeIcon({ hidden }: { hidden: boolean }) {
  return hidden ? (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      width="20"
      height="20"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="m3 3 18 18" />
      <path d="M10.6 10.7a2 2 0 0 0 2.7 2.7" />
      <path d="M9.9 4.2A10.7 10.7 0 0 1 12 4c5.4 0 9 5.5 9 8a9.2 9.2 0 0 1-2.1 3.5" />
      <path d="M6.6 6.6C4.4 8.1 3 10.5 3 12c0 2.5 3.6 8 9 8 1.3 0 2.5-.3 3.6-.8" />
    </svg>
  ) : (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      width="20"
      height="20"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M2.25 12s3.75-6.75 9.75-6.75S21.75 12 21.75 12 18 18.75 12 18.75 2.25 12 2.25 12Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { login, isAuthenticated, user } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [apiError, setApiError] = useState<string | null>(null);
  const [requestId, setRequestId] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const locationState = location.state as {
    signedUp?: boolean;
    logoutError?: string;
    authExpired?: boolean;
    from?: { pathname?: string };
    withdrawn?: boolean;
  } | null;
  const signedUp = Boolean(locationState?.signedUp);
  const logoutError = locationState?.logoutError ?? "";
  const authExpired = Boolean(locationState?.authExpired);
  const withdrawn = Boolean(locationState?.withdrawn);

  const destinationForRole = (
    role: "user" | "content_admin" | "system_admin",
  ): string => {
    if (role === "content_admin") return "/admin/content";
    if (role === "system_admin") return "/system/users";
    return "/";
  };

  if (isAuthenticated && user)
    return <Navigate to={destinationForRole(user.role)} replace />;

  const validate = (): FieldErrors => {
    const errors: FieldErrors = {};
    const normalizedEmail = email.trim();
    if (!/^\S+@\S+\.\S+$/.test(normalizedEmail) || normalizedEmail.length > 254)
      errors.email = "有効なメールアドレスを入力してください";
    if (countUnicodeCharacters(password) < 8)
      errors.password = "パスワードは8文字以上で入力してください";
    else if (countUtf8Bytes(password) > 72)
      errors.password = "パスワードが長すぎます";
    return errors;
  };

  const handleSubmit = async (
    event: FormEvent<HTMLFormElement>,
  ): Promise<void> => {
    event.preventDefault();
    if (isSubmitting) return;
    setFieldErrors({});
    setApiError(null);
    setRequestId("");
    const errors = validate();
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }
    setIsSubmitting(true);
    try {
      const loggedInUser = await login({
        email: email.trim().toLowerCase(),
        password,
      });
      navigate(
        locationState?.from?.pathname ?? destinationForRole(loggedInUser.role),
        { replace: true },
      );
    } catch (error: unknown) {
      setPassword("");
      if (error instanceof ApiRequestError) {
        setApiError(error.response.message);
        setRequestId(error.response.request_id);
      } else
        setApiError("通信に失敗しました。時間を置いて再度お試しください。");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="auth-page">
      <header className="auth-header">
        <Link className="auth-brand" to="/">
          Music Discovery
        </Link>
      </header>
      <div className="auth-layout">
        <section className="auth-copy">
          <p className="auth-kicker">Music Discovery</p>
          <h1>Welcome Back.</h1>
          <p className="auth-copy-lead">あなたの次の一曲を、また見つけよう。</p>
          <p className="auth-copy-description">
            ログインして、あなたの好みに合わせた音楽との出会いを続けましょう。
          </p>
        </section>
        <section className="auth-card" aria-labelledby="login-title">
          <div className="auth-card-header">
            <h2 id="login-title">ログイン</h2>
            <p>
              {signedUp
                ? "会員登録が完了しました。ログインしてください。"
                : "登録済みのメールアドレスでログインしてください。"}
            </p>
          </div>
          {withdrawn && (
            <div className="success-alert" role="status">
              退会が完了しました。
            </div>
          )}
          {authExpired && (
            <div className="api-alert" role="alert">
              認証期限が切れました。もう一度ログインしてください。
            </div>
          )}
          {logoutError && (
            <div className="api-alert" role="alert">
              {logoutError}
            </div>
          )}
          {apiError && (
            <div className="api-alert" role="alert">
              {apiError}
              {requestId && <small>Request ID: {requestId}</small>}
            </div>
          )}
          <form className="auth-form" onSubmit={handleSubmit} noValidate>
            <div className="field">
              <label htmlFor="login-email">メールアドレス</label>
              <input
                id="login-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={isSubmitting}
                autoComplete="email"
                aria-invalid={Boolean(fieldErrors.email)}
                aria-describedby={
                  fieldErrors.email ? "login-email-error" : undefined
                }
              />
              {fieldErrors.email && (
                <p className="field-error" id="login-email-error" role="alert">
                  {fieldErrors.email}
                </p>
              )}
            </div>
            <div className="field">
              <label htmlFor="login-password">パスワード</label>
              <div className="password-field">
                <input
                  id="login-password"
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  disabled={isSubmitting}
                  autoComplete="current-password"
                  aria-invalid={Boolean(fieldErrors.password)}
                  aria-describedby={
                    fieldErrors.password ? "login-password-error" : undefined
                  }
                />
                <button
                  className="password-toggle"
                  type="button"
                  disabled={isSubmitting}
                  onClick={() => setShowPassword((v) => !v)}
                  aria-label={
                    showPassword
                      ? "パスワードを非表示にする"
                      : "パスワードを表示する"
                  }
                >
                  <EyeIcon hidden={showPassword} />
                </button>
              </div>
              {fieldErrors.password && (
                <p
                  className="field-error"
                  id="login-password-error"
                  role="alert"
                >
                  {fieldErrors.password}
                </p>
              )}
            </div>
            <button
              className="primary-button"
              type="submit"
              disabled={isSubmitting}
            >
              {isSubmitting ? "ログイン中..." : "ログイン"}
            </button>
          </form>
          <p className="auth-switch">
            アカウントをお持ちでない方<Link to="/signup">会員登録</Link>
          </p>
        </section>
      </div>
    </div>
  );
}
