import { FormEvent, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { ROUTES } from "../constants/routes";

export function ProfilePage() {
  const { user, updateProfile } = useAuth();
  const navigate = useNavigate();
  const [name, setName] = useState(user?.name ?? "");
  const [email, setEmail] = useState(user?.email ?? "");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    setName(user?.name ?? "");
    setEmail(user?.email ?? "");
  }, [user]);

  const submit = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault();
    if (isSaving) return;
    setError("");
    setMessage("");
    setIsSaving(true);
    try {
      await updateProfile({ name, email });
      setMessage("プロフィールを更新しました。");
    } catch (caught: unknown) {
      setError(caught instanceof ApiRequestError ? caught.message : "プロフィールを更新できませんでした。");
    } finally {
      setIsSaving(false);
    }
  };

  return <main className="profile-page">
    <section className="profile-edit-card">
      <button type="button" className="question-top-button" onClick={() => navigate("/")}>TOPページへ戻る</button>
      <p className="top-kicker">Profile</p>
      <h1>プロフィール変更</h1>
      <p className="profile-edit-description">表示名とメールアドレスを変更できます。<br />音楽の好みはTOPページの「好みを変更」から更新してください。</p>
      <form onSubmit={(event) => void submit(event)}>
        <label className="field"><span>名前</span><input value={name} maxLength={50} required onChange={(event) => setName(event.target.value)} /></label>
        <label className="field"><span>メールアドレス</span><input type="email" value={email} maxLength={254} required onChange={(event) => setEmail(event.target.value)} /></label>
        {error && <p role="alert" className="form-error">{error}</p>}
        {message && <p role="status" className="form-success">{message}</p>}
        <button className="top-primary-button profile-save-button" type="submit" disabled={isSaving}>{isSaving ? "更新中..." : "変更を保存"}</button>
      </form>
      <div className="profile-password-reset">
        <div>
          <h2>パスワード再設定</h2>
          <p>登録メールアドレスに再設定用の案内を送ります。</p>
        </div>
        <button
          type="button"
          className="profile-password-reset-button"
          onClick={() => navigate(ROUTES.forgotPassword, { state: { email: user?.email ?? "" } })}
        >
          パスワードを再設定
        </button>
      </div>
    </section>
  </main>;
}
