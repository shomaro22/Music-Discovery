import { useRef, useState, type FormEvent } from "react";
import { PasswordToggleIcon } from "../common/PasswordToggleIcon";
import { WithdrawConfirmDialog } from "./WithdrawConfirmDialog";

interface WithdrawFormProps {
  isSubmitting: boolean;
  errorMessage: string | null;
  onWithdraw: (currentPassword: string) => Promise<void>;
  onCancel: () => void;
}

export function WithdrawForm({ isSubmitting, errorMessage, onWithdraw, onCancel }: WithdrawFormProps) {
  const [currentPassword, setCurrentPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const passwordRef = useRef<HTMLInputElement>(null);

  const handleSubmit = (event: FormEvent<HTMLFormElement>): void => {
    event.preventDefault();
    if (!currentPassword || isSubmitting) return;
    setIsDialogOpen(true);
  };

  const handleConfirm = async (): Promise<void> => {
    if (isSubmitting) return;
    try {
      await onWithdraw(currentPassword);
      setCurrentPassword("");
    } catch {
      setIsDialogOpen(false);
      passwordRef.current?.focus();
    }
  };

  return (
    <>
      <form className="withdraw-form" onSubmit={handleSubmit}>
        {errorMessage && <div className="api-alert" role="alert">{errorMessage}</div>}
        <div className="field">
          <label htmlFor="current-password">現在のパスワード</label>
          <div className="password-field">
            <input ref={passwordRef} id="current-password" type={showPassword ? "text" : "password"} autoComplete="current-password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} disabled={isSubmitting} />
            <button type="button" className="password-toggle" aria-label={showPassword ? "パスワードを非表示にする" : "パスワードを表示する"} onClick={() => setShowPassword((value) => !value)} disabled={isSubmitting}>
              <PasswordToggleIcon hidden={showPassword} />
            </button>
          </div>
        </div>
        <div className="withdraw-actions">
          <button type="button" className="secondary-button" onClick={onCancel} disabled={isSubmitting}>キャンセル</button>
          <button type="submit" className="danger-button" disabled={!currentPassword || isSubmitting}>退会する</button>
        </div>
      </form>
      <WithdrawConfirmDialog isOpen={isDialogOpen} isSubmitting={isSubmitting} onCancel={() => setIsDialogOpen(false)} onConfirm={() => void handleConfirm()} />
    </>
  );
}
