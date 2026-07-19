import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";

import { ApiRequestError } from "../api/client";
import { signUp } from "../api/user";
import type { SignUpInput } from "../types/user";
import { countUnicodeCharacters, countUtf8Bytes } from "../utils/stringLength";

type SignupFormValues = SignUpInput & {
  passwordConfirmation: string;
};

type FieldName = keyof SignupFormValues;
type FieldErrors = Partial<Record<FieldName, string>>;

export function SignupPage() {
  const navigate = useNavigate();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirmation, setPasswordConfirmation] = useState("");

  const [showPassword, setShowPassword] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [apiError, setApiError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const validate = (): FieldErrors => {
    const errors: FieldErrors = {};

    const normalizedName = name.trim();
    const normalizedEmail = email.trim();

    const nameLength = countUnicodeCharacters(normalizedName);
    const passwordLength = countUnicodeCharacters(password);
    const passwordByteLength = countUtf8Bytes(password);

    if (nameLength < 1 || nameLength > 50) {
      errors.name = "名前は1文字以上50文字以内で入力してください";
    }

    if (
      !/^\S+@\S+\.\S+$/.test(normalizedEmail) ||
      normalizedEmail.length > 254
    ) {
      errors.email = "有効なメールアドレスを入力してください";
    }

    if (passwordLength < 8) {
      errors.password = "パスワードは8文字以上で入力してください";
    } else if (passwordByteLength > 72) {
      errors.password = "パスワードが長すぎます";
    }

    if (password !== passwordConfirmation) {
      errors.passwordConfirmation = "パスワードが一致しません";
    }

    return errors;
  };

  const clearPasswords = (): void => {
    setPassword("");
    setPasswordConfirmation("");
  };

  const handleSubmit = async (
    event: FormEvent<HTMLFormElement>,
  ): Promise<void> => {
    event.preventDefault();

    if (isSubmitting) {
      return;
    }

    setFieldErrors({});
    setApiError(null);

    const errors = validate();

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    setIsSubmitting(true);

    try {
      await signUp({
        name: name.trim(),
        email: email.trim().toLowerCase(),
        password,
      });

      navigate("/login", {
        replace: true,
        state: { signedUp: true },
      });
    } catch (error: unknown) {
      clearPasswords();

      if (error instanceof ApiRequestError) {
        if (error.response.code === "EMAIL_ALREADY_EXISTS") {
          setFieldErrors({
            email: error.response.message,
          });
        } else {
          setApiError(error.response.message);
        }
      } else {
        setApiError("通信に失敗しました。時間を置いて再度お試しください。");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <main>
      <section>
        <p>Discover Your Next Favorite.</p>
        <p>あなたに合う、まだ知らない音楽と出会おう。</p>

        <h1>アカウントを作成</h1>

        {apiError && (
          <p className="api-alert" role="alert">
            {apiError}
          </p>
        )}

        <form onSubmit={handleSubmit} noValidate>
          <div className="field">
            <label htmlFor="signup-name">名前</label>
            <input
              id="signup-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              disabled={isSubmitting}
              autoComplete="name"
              aria-invalid={Boolean(fieldErrors.name)}
              aria-describedby={
                fieldErrors.name ? "signup-name-error" : undefined
              }
            />

            {fieldErrors.name && (
              <p id="signup-name-error" role="alert">
                {fieldErrors.name}
              </p>
            )}
          </div>

          <div className="field">
            <label htmlFor="signup-email">メールアドレス</label>
            <input
              id="signup-email"
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              disabled={isSubmitting}
              autoComplete="email"
              aria-invalid={Boolean(fieldErrors.email)}
              aria-describedby={
                fieldErrors.email ? "signup-email-error" : undefined
              }
            />

            {fieldErrors.email && (
              <p id="signup-email-error" role="alert">
                {fieldErrors.email}
              </p>
            )}
          </div>

          <div className="field">
            <label htmlFor="signup-password">パスワード</label>

            <div>
              <input
                id="signup-password"
                type={showPassword ? "text" : "password"}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                disabled={isSubmitting}
                autoComplete="new-password"
                aria-invalid={Boolean(fieldErrors.password)}
                aria-describedby={
                  fieldErrors.password ? "signup-password-error" : undefined
                }
              />

              <button
                type="button"
                onClick={() => setShowPassword((current) => !current)}
                disabled={isSubmitting}
                aria-label={
                  showPassword
                    ? "パスワードを非表示にする"
                    : "パスワードを表示する"
                }
              >
                {showPassword ? "非表示" : "表示"}
              </button>
            </div>

            {fieldErrors.password && (
              <p id="signup-password-error" role="alert">
                {fieldErrors.password}
              </p>
            )}
          </div>

          <div className="field">
            <label htmlFor="signup-password-confirmation">パスワード確認</label>

            <input
              id="signup-password-confirmation"
              type={showPassword ? "text" : "password"}
              value={passwordConfirmation}
              onChange={(event) => setPasswordConfirmation(event.target.value)}
              disabled={isSubmitting}
              autoComplete="new-password"
              aria-invalid={Boolean(fieldErrors.passwordConfirmation)}
              aria-describedby={
                fieldErrors.passwordConfirmation
                  ? "signup-password-confirmation-error"
                  : undefined
              }
            />

            {fieldErrors.passwordConfirmation && (
              <p id="signup-password-confirmation-error" role="alert">
                {fieldErrors.passwordConfirmation}
              </p>
            )}
          </div>

          <button type="submit" disabled={isSubmitting}>
            {isSubmitting ? "作成中..." : "アカウントを作成"}
          </button>
        </form>

        <Link to="/login">ログインはこちら</Link>
      </section>
    </main>
  );
}
