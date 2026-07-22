import { useEffect } from "react";

interface WithdrawConfirmDialogProps {
  isOpen: boolean;
  isSubmitting: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function WithdrawConfirmDialog({ isOpen, isSubmitting, onConfirm, onCancel }: WithdrawConfirmDialogProps) {
  useEffect(() => {
    if (!isOpen) return undefined;

    const handleKeyDown = (event: KeyboardEvent): void => {
      if (event.key === "Escape" && !isSubmitting) {
        event.preventDefault();
        onCancel();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, isSubmitting, onCancel]);

  if (!isOpen) return null;
  return (
    <div className="dialog-backdrop" role="presentation" onMouseDown={() => { if (!isSubmitting) onCancel(); }}>
      <div className="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="withdraw-dialog-title" onMouseDown={(event) => event.stopPropagation()}>
        <h2 id="withdraw-dialog-title">本当に退会しますか？</h2>
        <p>退会処理は取り消せません。すべての端末でログイン状態が無効になります。</p>
        <div className="dialog-actions">
          <button type="button" className="secondary-button" onClick={onCancel} disabled={isSubmitting}>キャンセル</button>
          <button type="button" className="danger-button" onClick={onConfirm} disabled={isSubmitting}>{isSubmitting ? "退会処理中..." : "退会を確定"}</button>
        </div>
      </div>
    </div>
  );
}
