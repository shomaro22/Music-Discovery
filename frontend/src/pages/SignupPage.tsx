import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { signUp } from "../api/user";
import { ApiRequestError } from "../api/client";
import type { SignUpInput } from "../types/user";
import { countUnicodeCharacters, countUtf8Bytes } from "../utils/stringLength";

type FieldErrors = Partial<Record<keyof SignUpInput, string>>;

export function SignupPage() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
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

    if (nameLength < 1 || nameLength > 50)
      errors.name = "名前は1文字以上50文字以内で入力してください";
    if (!/^\S+@\S+\.\S+$/.test(normalizedEmail) || normalizedEmail.length > 254)
      errors.email = "有効なメールアドレスを入力してください";
    if (passwordLength < 8)
      errors.password = "パスワードは8文字以上で入力してください";
    else if (passwordByteLength > 72)
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
      navigate("/login", { replace: true, state: { signedUp: true } });
    } catch (error: unknown) {
      setPassword("");
      if (error instanceof ApiRequestError) {
        if (error.response.code === "EMAIL_ALREADY_EXISTS") {
          setFieldErrors({ email: error.response.message });
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
      <h1>会員新規登録</h1>
      <form onSubmit={handleSubmit} noValidate>
        {apiError && <p role="alert">{apiError}</p>}
        <label>
          名前
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={isSubmitting}
            autoComplete="name"
          />
        </label>
        {fieldErrors.name && <p role="alert">{fieldErrors.name}</p>}
        <label>
          メールアドレス
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            disabled={isSubmitting}
            autoComplete="email"
          />
        </label>
        {fieldErrors.email && <p role="alert">{fieldErrors.email}</p>}
        <label>
          パスワード
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={isSubmitting}
            autoComplete="new-password"
          />
        </label>
        {fieldErrors.password && <p role="alert">{fieldErrors.password}</p>}
        <button type="submit" disabled={isSubmitting}>
          {isSubmitting ? "登録中..." : "登録する"}
        </button>
      </form>
      <Link to="/login">ログインはこちら</Link>
    </main>
  );
}
