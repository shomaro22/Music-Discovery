import { useId, useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";

import { ApiRequestError } from "../api/client";
import { signUp } from "../api/user";
import type { SignUpInput } from "../types/user";
import { countUnicodeCharacters, countUtf8Bytes } from "../utils/stringLength";

type FormField = keyof SignUpInput | "passwordConfirmation";
type FieldErrors = Partial<Record<FormField, string>>;

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

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
      <path d="M2.5 12s3.5-7 9.5-7 9.5 7 9.5 7-3.5 7-9.5 7-9.5-7-9.5-7Z" />
      <circle cx="12" cy="12" r="2.5" />
    </svg>
  );
}

export function SignupPage() {
  const navigate = useNavigate();
  const nameId = useId();
  const emailId = useId();
  const passwordId = useId();
  const passwordConfirmationId = useId();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirmation, setPasswordConfirmation] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [showPasswordConfirmation, setShowPasswordConfirmation] =
    useState(false);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [apiError, setApiError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const validate = (): { input: SignUpInput; errors: FieldErrors } => {
    const normalizedName = name.trim();
    const normalizedEmail = email.trim().toLowerCase();
    const errors: FieldErrors = {};

    const nameLength = countUnicodeCharacters(normalizedName);
    if (nameLength < 1 || nameLength > 50) {
      errors.name = "名前は1文字以上50文字以内で入力してください";
    }

    if (!EMAIL_PATTERN.test(normalizedEmail)) {
      errors.email = "有効なメールアドレスを入力してください";
    }

    const passwordBytes = countUtf8Bytes(password);
    if (passwordBytes < 8) {
      errors.password = "パスワードは8文字以上で入力してください";
    } else if (passwordBytes > 72) {
      errors.password = "パスワードは72バイト以内で入力してください";
    }

    if (passwordConfirmation !== password) {
      errors.passwordConfirmation = "パスワードが一致しません";
    }

    return {
      input: {
        name: normalizedName,
        email: normalizedEmail,
        password,
      },
      errors,
    };
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (isSubmitting) {
      return;
    }

    setApiError("");
    setFieldErrors({});

    const { input, errors } = validate();
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    setIsSubmitting(true);

    try {
      await signUp(input);
      navigate("/login");
    } catch (error: unknown) {
      if (error instanceof ApiRequestError) {
        const response = error.response;
        const code = response?.code;
        const message = response?.message ?? error.message;

        if (
          code === "EMAIL_ALREADY_EXISTS" ||
          message === "このメールアドレスは既に登録されています"
        ) {
          setFieldErrors({
            email: "このメールアドレスは既に登録されています",
          });
          setPassword("");
          setPasswordConfirmation("");
          return;
        }

        setApiError(message || "登録に失敗しました");
        return;
      }

      setApiError("登録に失敗しました");
    } finally {
      setIsSubmitting(false);
    }
  };

  const nameErrorId = fieldErrors.name ? `${nameId}-error` : undefined;
  const emailErrorId = fieldErrors.email ? `${emailId}-error` : undefined;
  const passwordHintId = `${passwordId}-hint`;
  const passwordErrorId = fieldErrors.password
    ? `${passwordId}-error`
    : undefined;
  const passwordDescription = [passwordHintId, passwordErrorId]
    .filter(Boolean)
    .join(" ");
  const confirmationErrorId = fieldErrors.passwordConfirmation
    ? `${passwordConfirmationId}-error`
    : undefined;

  return (
    <main className="auth-page signup-auth-page">
      <header className="auth-header">
        <Link className="auth-brand" to="/">Music Discovery</Link>
      </header>

      <div className="auth-layout signup-auth-layout">
        <section className="auth-copy" aria-label="Music Discoveryの紹介">
          <p className="auth-kicker">Music Discovery</p>
          <h1>Start Your<br />Discovery.</h1>
          <p className="auth-copy-lead">あなたの次の一曲を、ここから見つけよう。</p>
          <p className="auth-copy-description">
            アカウントを作成して、あなたの好みに合わせた音楽との出会いを始めましょう。
          </p>
        </section>

        <section className="auth-card signup-auth-card">
          <div className="auth-card-header">
            <h2>アカウントを作成</h2>
            <p>あなたに合う音楽を見つける準備を始めましょう。</p>
          </div>

          {apiError && <p className="api-alert" role="alert">{apiError}</p>}

          <form className="auth-form signup-auth-form" noValidate onSubmit={handleSubmit}>
            <div className="field">
              <label htmlFor={nameId}>名前</label>
              <input id={nameId} value={name} onChange={(event) => setName(event.target.value)} autoComplete="name" placeholder="山田 太郎" aria-invalid={Boolean(fieldErrors.name)} aria-describedby={nameErrorId} disabled={isSubmitting} />
              {fieldErrors.name && <p className="field-error" id={nameErrorId}>{fieldErrors.name}</p>}
            </div>

            <div className="field">
              <label htmlFor={emailId}>メールアドレス</label>
              <input id={emailId} type="email" value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="email" placeholder="you@example.com" aria-invalid={Boolean(fieldErrors.email)} aria-describedby={emailErrorId} disabled={isSubmitting} />
              {fieldErrors.email && <p className="field-error" id={emailErrorId}>{fieldErrors.email}</p>}
            </div>

            <div className="field">
              <label htmlFor={passwordId}>パスワード</label>
              <div className="password-field">
                <input id={passwordId} type={showPassword ? "text" : "password"} value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="new-password" placeholder="8文字以上で入力" aria-invalid={Boolean(fieldErrors.password)} aria-describedby={passwordDescription} disabled={isSubmitting} />
                <button type="button" className="password-toggle" aria-label={showPassword ? "パスワードを非表示にする" : "パスワードを表示する"} onClick={() => setShowPassword((current) => !current)} disabled={isSubmitting}>
                  <EyeIcon hidden={showPassword} />
                </button>
              </div>
              <p className="field-hint" id={passwordHintId}>8文字以上で入力してください。</p>
              {fieldErrors.password && <p className="field-error" id={passwordErrorId}>{fieldErrors.password}</p>}
            </div>

            <div className="field">
              <label htmlFor={passwordConfirmationId}>パスワード確認</label>
              <div className="password-field">
                <input id={passwordConfirmationId} type={showPasswordConfirmation ? "text" : "password"} value={passwordConfirmation} onChange={(event) => setPasswordConfirmation(event.target.value)} autoComplete="new-password" placeholder="パスワードを再入力" aria-invalid={Boolean(fieldErrors.passwordConfirmation)} aria-describedby={confirmationErrorId} disabled={isSubmitting} />
                <button type="button" className="password-toggle" aria-label={showPasswordConfirmation ? "パスワード確認を非表示にする" : "パスワード確認を表示する"} onClick={() => setShowPasswordConfirmation((current) => !current)} disabled={isSubmitting}>
                  <EyeIcon hidden={showPasswordConfirmation} />
                </button>
              </div>
              {fieldErrors.passwordConfirmation && <p className="field-error" id={confirmationErrorId}>{fieldErrors.passwordConfirmation}</p>}
            </div>

            <p className="signup-terms">
              アカウントを作成することで、<a href="/terms">利用規約</a>と<a href="/privacy">プライバシーポリシー</a>に同意したものとみなされます。
            </p>

            <button className="primary-button" type="submit" disabled={isSubmitting}>
              {isSubmitting ? "登録中..." : "アカウントを作成"}
            </button>
          </form>

          <p className="auth-switch">すでにアカウントをお持ちですか？ <Link to="/login">ログイン</Link></p>
        </section>
      </div>
    </main>
  );
}
