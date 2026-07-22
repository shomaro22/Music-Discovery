import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { WithdrawForm } from "../components/withdraw/WithdrawForm";

export function WithdrawPage() {
  const navigate = useNavigate();
  const { withdraw } = useAuth();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const handleWithdraw = async (currentPassword: string): Promise<void> => {
    if (isSubmitting) return;
    setIsSubmitting(true);
    setErrorMessage(null);
    try {
      await withdraw({ current_password: currentPassword });
      navigate("/login", { replace: true, state: { withdrawn: true } });
    } catch (error: unknown) {
      if (error instanceof ApiRequestError) {
        setErrorMessage(error.response.code === "CURRENT_PASSWORD_MISMATCH" ? "現在のパスワードが正しくありません。" : error.response.message);
      } else {
        setErrorMessage("処理に失敗しました。時間を置いて再度お試しください。");
      }
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <main className="withdraw-page">
      <section className="withdraw-card" aria-labelledby="withdraw-title">
        <p className="withdraw-kicker">Account settings</p>
        <h1 id="withdraw-title">アカウントの退会</h1>
        <div className="withdraw-warning">
          <p>退会すると同じアカウントではログインできません。</p>
          <p>発行済みの認証情報はすべての端末で無効になります。</p>
          <p>この操作は取り消せません。</p>
        </div>
        <WithdrawForm isSubmitting={isSubmitting} errorMessage={errorMessage} onWithdraw={handleWithdraw} onCancel={() => navigate(-1)} />
      </section>
    </main>
  );
}
